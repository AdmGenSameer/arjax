package updates

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/identity"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/transaction"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// UpdateProvider defines the ecosystem contract for update discovery and execution
type UpdateProvider interface {
	Name() string
	CheckUpdates(ctx context.Context) ([]AvailableUpdate, error)
	Update(ctx context.Context, offerID string) error
}

// CommandRunner abstracts subprocess execution for deterministic exit-code testing
type CommandRunner func(ctx context.Context, name string, args ...string) utils.ExecResult

func isSecuritySensitivePkg(pkgName string) bool {
	low := strings.ToLower(pkgName)
	for _, kw := range []string{"openssl", "linux", "sudo", "systemd", "chromium", "firefox", "brave", "gnutls", "curl", "nss", "polkit", "openssh", "glibc"} {
		if strings.Contains(low, kw) {
			return true
		}
	}
	return false
}

// PacmanUpdateProvider discovers official Arch Linux updates via `checkupdates`.
// Exit codes per checkupdates(8):
//   0 = normal completion (updates present)
//   1 = unknown failure -> MUST return error
//   2 = no updates available -> MUST return empty list with nil error
type PacmanUpdateProvider struct {
	txMgr  *transaction.Manager
	runner CommandRunner
}

func NewPacmanUpdateProvider(txMgr *transaction.Manager) *PacmanUpdateProvider {
	return &PacmanUpdateProvider{
		txMgr:  txMgr,
		runner: utils.RunCommand,
	}
}

func NewPacmanUpdateProviderWithRunner(txMgr *transaction.Manager, runner CommandRunner) *PacmanUpdateProvider {
	if runner == nil {
		runner = utils.RunCommand
	}
	return &PacmanUpdateProvider{
		txMgr:  txMgr,
		runner: runner,
	}
}

func (p *PacmanUpdateProvider) Name() string { return "pacman" }

func (p *PacmanUpdateProvider) CheckUpdates(ctx context.Context) ([]AvailableUpdate, error) {
	if p.runner == nil {
		p.runner = utils.RunCommand
	}
	// Only enforce binary check when using real host runner
	if !utils.CommandExists("checkupdates") && fmt.Sprintf("%p", p.runner) == fmt.Sprintf("%p", utils.RunCommand) {
		if !utils.CommandExists("pacman") {
			return nil, fmt.Errorf("pacman not installed on host")
		}
		return []AvailableUpdate{}, nil
	}

	res := p.runner(ctx, "checkupdates")
	switch res.ExitCode {
	case 0:
		return ParseCheckupdatesOutput(res.Stdout), nil
	case 2:
		// checkupdates(8): Exit status 2 means no updates are available (healthy zero state)
		return []AvailableUpdate{}, nil
	default:
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" && res.Error != nil {
			errMsg = res.Error.Error()
		}
		return nil, fmt.Errorf("checkupdates failed with exit code %d: %s", res.ExitCode, errMsg)
	}
}

// ParseCheckupdatesOutput parses `pkg oldver -> newver` lines and marks PartialUpgradeSafe=false
func ParseCheckupdatesOutput(output string) []AvailableUpdate {
	resolver := identity.NewResolver(identity.GetDefaultRegistry())
	var out []AvailableUpdate
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, " -> ")
		if len(parts) != 2 {
			continue
		}
		left := strings.Fields(parts[0])
		if len(left) != 2 {
			continue
		}
		name := left[0]
		oldVer := left[1]
		newVer := strings.TrimSpace(parts[1])
		res := resolver.Resolve(name, "pacman")

		out = append(out, AvailableUpdate{
			OfferID:            fmt.Sprintf("pacman:extra:%s", strings.ToLower(name)),
			AppID:              res.CanonicalID,
			PackageName:        name,
			DisplayName:        pickDisplayName(res.DisplayName, name),
			Provider:           "pacman",
			Repository:         "extra",
			InstallScope:       "system",
			InstalledVersion:   oldVer,
			AvailableVersion:   newVer,
			DownloadSize:       "~18 MB",
			IsSecurity:         isSecuritySensitivePkg(name),
			PartialUpgradeSafe: false,
			UpgradeGroupPolicy: "full_system_upgrade",
			TrustLevel:         "official",
			Summary:            "Requires full Arch system upgrade (pacman -Syu)",
		})
	}
	return out
}

func (p *PacmanUpdateProvider) Update(ctx context.Context, offerID string) error {
	if p.txMgr == nil {
		return fmt.Errorf("transaction manager unavailable")
	}
	_, err := p.txMgr.ExecuteOfferTransaction(ctx, offerID, transaction.OpUpdate, nil)
	return err
}

// FlatpakUpdateProvider discovers default --user and --system Flatpak installation updates independently.
// Note: Custom named `--installation=NAME` Flatpak remotes are outside v1.0 scope.
type FlatpakUpdateProvider struct {
	txMgr *transaction.Manager
}

func NewFlatpakUpdateProvider(txMgr *transaction.Manager) *FlatpakUpdateProvider {
	return &FlatpakUpdateProvider{txMgr: txMgr}
}

func (f *FlatpakUpdateProvider) Name() string { return "flatpak" }

func (f *FlatpakUpdateProvider) CheckUpdates(ctx context.Context) ([]AvailableUpdate, error) {
	if !utils.CommandExists("flatpak") {
		return nil, fmt.Errorf("flatpak CLI not installed")
	}

	var all []AvailableUpdate
	var errs []string
	for _, scope := range []string{"user", "system"} {
		out, err := exec.CommandContext(ctx, "flatpak", "remote-ls", "--updates", "--"+scope, "--columns=name,application,version,branch,origin").Output()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", scope, err))
			continue
		}
		all = append(all, ParseFlatpakUpdatesOutputScoped(string(out), scope)...)
	}

	if len(all) == 0 && len(errs) == 2 {
		return nil, fmt.Errorf("flatpak update check failed (%s)", strings.Join(errs, "; "))
	}
	return all, nil
}

func ParseFlatpakUpdatesOutput(output string) []AvailableUpdate {
	return ParseFlatpakUpdatesOutputScoped(output, "system")
}

func ParseFlatpakUpdatesOutputScoped(output, scope string) []AvailableUpdate {
	normScope := strings.ToLower(strings.TrimSpace(scope))
	if normScope != "user" {
		normScope = "system"
	}

	var out []AvailableUpdate
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 2 {
			continue
		}
		name := strings.TrimSpace(cols[0])
		appID := strings.TrimSpace(cols[1])
		newVer := "latest"
		if len(cols) >= 3 && strings.TrimSpace(cols[2]) != "" {
			newVer = strings.TrimSpace(cols[2])
		}
		branch := "stable"
		if len(cols) >= 4 && strings.TrimSpace(cols[3]) != "" {
			branch = strings.TrimSpace(cols[3])
		}
		origin := "flathub"
		if len(cols) >= 5 && strings.TrimSpace(cols[4]) != "" {
			origin = strings.TrimSpace(cols[4])
		}

		offerID := fmt.Sprintf("flatpak:%s:%s:scope=%s:branch=%s", strings.ToLower(origin), strings.ToLower(appID), normScope, strings.ToLower(branch))
		out = append(out, AvailableUpdate{
			OfferID:            offerID,
			AppID:              appID,
			PackageName:        appID,
			DisplayName:        pickDisplayName(name, appID),
			Provider:           "flatpak",
			Repository:         origin,
			InstallScope:       normScope,
			Branch:             branch,
			InstalledVersion:   "installed",
			AvailableVersion:   newVer,
			DownloadSize:       "~45 MB",
			IsSecurity:         isSecuritySensitivePkg(appID),
			PartialUpgradeSafe: true,
			UpgradeGroupPolicy: "scoped_ref",
			TrustLevel:         "verified",
			Summary:            fmt.Sprintf("Flatpak update (%s · %s/%s)", normScope, origin, branch),
		})
	}
	return out
}

func (f *FlatpakUpdateProvider) Update(ctx context.Context, offerID string) error {
	if f.txMgr == nil {
		return fmt.Errorf("transaction manager unavailable")
	}
	_, err := f.txMgr.ExecuteOfferTransaction(ctx, offerID, transaction.OpUpdate, nil)
	return err
}

// AURUpdateProvider discovers AUR updates via `paru -Qua` and upgrades via `paru -Sua` (after Pacman)
type AURUpdateProvider struct {
	txMgr *transaction.Manager
}

func NewAURUpdateProvider(txMgr *transaction.Manager) *AURUpdateProvider {
	return &AURUpdateProvider{txMgr: txMgr}
}

func (a *AURUpdateProvider) Name() string { return "aur" }

func (a *AURUpdateProvider) CheckUpdates(ctx context.Context) ([]AvailableUpdate, error) {
	if !utils.CommandExists("paru") {
		if !utils.CommandExists("yay") {
			return nil, fmt.Errorf("no AUR helper (paru/yay) installed")
		}
		out, err := exec.CommandContext(ctx, "yay", "-Qua").Output()
		if err != nil && len(strings.TrimSpace(string(out))) == 0 {
			return []AvailableUpdate{}, nil
		}
		return ParseAURUpdatesOutput(string(out)), nil
	}
	out, err := exec.CommandContext(ctx, "paru", "-Qua").Output()
	if err != nil && len(strings.TrimSpace(string(out))) == 0 {
		return []AvailableUpdate{}, nil
	}
	return ParseAURUpdatesOutput(string(out)), nil
}

func ParseAURUpdatesOutput(output string) []AvailableUpdate {
	resolver := identity.NewResolver(identity.GetDefaultRegistry())
	var out []AvailableUpdate
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, " -> ")
		if len(parts) != 2 {
			continue
		}
		left := strings.Fields(parts[0])
		if len(left) != 2 {
			continue
		}
		name := left[0]
		oldVer := left[1]
		newVer := strings.TrimSpace(parts[1])
		res := resolver.Resolve(name, "aur")

		out = append(out, AvailableUpdate{
			OfferID:            fmt.Sprintf("aur:aur:%s", strings.ToLower(name)),
			AppID:              res.CanonicalID,
			PackageName:        name,
			DisplayName:        pickDisplayName(res.DisplayName, name),
			Provider:           "aur",
			Repository:         "aur",
			InstallScope:       "user",
			InstalledVersion:   oldVer,
			AvailableVersion:   newVer,
			DownloadSize:       "PKGBUILD",
			IsSecurity:         isSecuritySensitivePkg(name),
			PartialUpgradeSafe: false,
			UpgradeGroupPolicy: "aur_helper_upgrade",
			TrustLevel:         "community",
			Summary:            "AUR community upgrade (requires official Pacman repos updated first; applies via paru -Sua)",
		})
	}
	return out
}

func (a *AURUpdateProvider) Update(ctx context.Context, offerID string) error {
	if a.txMgr == nil {
		return fmt.Errorf("transaction manager unavailable")
	}
	_, err := a.txMgr.ExecuteOfferTransaction(ctx, offerID, transaction.OpUpdate, nil)
	return err
}

// SnapUpdateProvider checks Snap updates via `snap refresh --list` (snapshot revalidated pre-execution)
type SnapUpdateProvider struct {
	txMgr *transaction.Manager
}

func NewSnapUpdateProvider(txMgr *transaction.Manager) *SnapUpdateProvider {
	return &SnapUpdateProvider{txMgr: txMgr}
}

func (s *SnapUpdateProvider) Name() string { return "snap" }

func (s *SnapUpdateProvider) CheckUpdates(ctx context.Context) ([]AvailableUpdate, error) {
	if !utils.CommandExists("snap") {
		return nil, fmt.Errorf("snap not installed on host")
	}
	out, err := exec.CommandContext(ctx, "snap", "refresh", "--list").Output()
	if err != nil {
		return nil, fmt.Errorf("snapd refresh check unavailable: %w", err)
	}
	return ParseSnapUpdatesOutput(string(out)), nil
}

func ParseSnapUpdatesOutput(output string) []AvailableUpdate {
	var out []AvailableUpdate
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if first && strings.HasPrefix(line, "Name") {
			first = false
			continue
		}
		first = false
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			name := fields[0]
			ver := fields[1]
			out = append(out, AvailableUpdate{
				OfferID:            fmt.Sprintf("snap:snapcraft:%s", strings.ToLower(name)),
				AppID:              strings.ToLower(name),
				PackageName:        name,
				DisplayName:        strings.Title(name),
				Provider:           "snap",
				Repository:         "snapcraft",
				InstallScope:       "system",
				InstalledVersion:   "installed",
				AvailableVersion:   ver,
				DownloadSize:       "~35 MB",
				IsSecurity:         isSecuritySensitivePkg(name),
				PartialUpgradeSafe: true,
				UpgradeGroupPolicy: "snap_refresh",
				TrustLevel:         "verified",
				Summary:            "Snapcraft pending refresh snapshot (revalidated before refresh)",
			})
		}
	}
	return out
}

func (s *SnapUpdateProvider) Update(ctx context.Context, offerID string) error {
	if s.txMgr == nil {
		return fmt.Errorf("transaction manager unavailable")
	}
	_, err := s.txMgr.ExecuteOfferTransaction(ctx, offerID, transaction.OpUpdate, nil)
	return err
}

func pickDisplayName(preferred, fallback string) string {
	if strings.TrimSpace(preferred) != "" {
		return preferred
	}
	return fallback
}

// AptUpdateProvider discovers pending Debian / Ubuntu / Kali updates via `apt list --upgradable`.
type AptUpdateProvider struct {
	txMgr  *transaction.Manager
	runner CommandRunner
}

func NewAptUpdateProvider(txMgr *transaction.Manager) *AptUpdateProvider {
	return &AptUpdateProvider{txMgr: txMgr, runner: utils.RunCommand}
}

func NewAptUpdateProviderWithRunner(txMgr *transaction.Manager, runner CommandRunner) *AptUpdateProvider {
	if runner == nil {
		runner = utils.RunCommand
	}
	return &AptUpdateProvider{txMgr: txMgr, runner: runner}
}

func (a *AptUpdateProvider) Name() string { return "apt" }

func (a *AptUpdateProvider) CheckUpdates(ctx context.Context) ([]AvailableUpdate, error) {
	if a.runner == nil {
		a.runner = utils.RunCommand
	}
	if !utils.CommandExists("apt") && fmt.Sprintf("%p", a.runner) == fmt.Sprintf("%p", utils.RunCommand) {
		return nil, fmt.Errorf("apt not installed on host")
	}
	res := a.runner(ctx, "apt", "list", "--upgradable")
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("apt list --upgradable failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return ParseAptUpdatesOutput(res.Stdout), nil
}

// ParseAptUpdatesOutput parses `pkg/suite newver arch [upgradable from: oldver]`.
func ParseAptUpdatesOutput(output string) []AvailableUpdate {
	resolver := identity.NewResolver(identity.GetDefaultRegistry())
	var out []AvailableUpdate
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "Listing...") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.Contains(fields[0], "/") {
			continue
		}
		slashIdx := strings.Index(fields[0], "/")
		name := fields[0][:slashIdx]
		suite := strings.SplitN(fields[0][slashIdx+1:], ",", 2)[0]
		if suite == "" {
			suite = "main"
		}
		newVer := fields[1]
		oldVer := "installed"
		if idx := strings.Index(line, "[upgradable from:"); idx >= 0 {
			rest := strings.TrimSpace(line[idx+len("[upgradable from:"):])
			oldVer = strings.TrimSuffix(rest, "]")
			oldVer = strings.TrimSpace(oldVer)
		}
		res := resolver.Resolve(name, "apt")
		out = append(out, AvailableUpdate{
			OfferID:            fmt.Sprintf("apt:%s:%s", strings.ToLower(suite), strings.ToLower(name)),
			AppID:              res.CanonicalID,
			PackageName:        name,
			DisplayName:        pickDisplayName(res.DisplayName, name),
			Provider:           "apt",
			Repository:         suite,
			InstallScope:       "system",
			InstalledVersion:   oldVer,
			AvailableVersion:   newVer,
			DownloadSize:       "~20 MB",
			IsSecurity:         isSecuritySensitivePkg(name),
			PartialUpgradeSafe: true,
			UpgradeGroupPolicy: "apt_upgrade",
			TrustLevel:         "official",
			Summary:            fmt.Sprintf("APT upgrade (%s)", suite),
		})
	}
	return out
}

func (a *AptUpdateProvider) Update(ctx context.Context, offerID string) error {
	if a.txMgr == nil {
		return fmt.Errorf("transaction manager unavailable")
	}
	_, err := a.txMgr.ExecuteOfferTransaction(ctx, offerID, transaction.OpUpdate, nil)
	return err
}

// DnfUpdateProvider discovers pending Fedora / RHEL updates via `dnf check-update -q`.
// Per dnf(8): exit code 100 = updates available, 0 = none available, anything else = error.
type DnfUpdateProvider struct {
	txMgr  *transaction.Manager
	runner CommandRunner
}

func NewDnfUpdateProvider(txMgr *transaction.Manager) *DnfUpdateProvider {
	return &DnfUpdateProvider{txMgr: txMgr, runner: utils.RunCommand}
}

func NewDnfUpdateProviderWithRunner(txMgr *transaction.Manager, runner CommandRunner) *DnfUpdateProvider {
	if runner == nil {
		runner = utils.RunCommand
	}
	return &DnfUpdateProvider{txMgr: txMgr, runner: runner}
}

func (d *DnfUpdateProvider) Name() string { return "dnf" }

func (d *DnfUpdateProvider) CheckUpdates(ctx context.Context) ([]AvailableUpdate, error) {
	if d.runner == nil {
		d.runner = utils.RunCommand
	}
	if !utils.CommandExists("dnf") && fmt.Sprintf("%p", d.runner) == fmt.Sprintf("%p", utils.RunCommand) {
		return nil, fmt.Errorf("dnf not installed on host")
	}
	res := d.runner(ctx, "dnf", "check-update", "-q")
	switch res.ExitCode {
	case 0:
		return []AvailableUpdate{}, nil
	case 100:
		return ParseDnfUpdatesOutput(res.Stdout), nil
	default:
		return nil, fmt.Errorf("dnf check-update failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
}

// ParseDnfUpdatesOutput parses `pkg.arch   newver   repo` lines.
func ParseDnfUpdatesOutput(output string) []AvailableUpdate {
	resolver := identity.NewResolver(identity.GetDefaultRegistry())
	var out []AvailableUpdate
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "Obsoleting") || strings.HasPrefix(line, "Security:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		nameArch := fields[0]
		name := nameArch
		if dotIdx := strings.LastIndex(nameArch, "."); dotIdx > 0 {
			name = nameArch[:dotIdx]
		}
		newVer := fields[1]
		repo := fields[2]
		res := resolver.Resolve(name, "dnf")
		out = append(out, AvailableUpdate{
			OfferID:            fmt.Sprintf("dnf:%s:%s", strings.ToLower(repo), strings.ToLower(name)),
			AppID:              res.CanonicalID,
			PackageName:        name,
			DisplayName:        pickDisplayName(res.DisplayName, name),
			Provider:           "dnf",
			Repository:         repo,
			InstallScope:       "system",
			InstalledVersion:   "installed",
			AvailableVersion:   newVer,
			DownloadSize:       "~22 MB",
			IsSecurity:         isSecuritySensitivePkg(name),
			PartialUpgradeSafe: true,
			UpgradeGroupPolicy: "dnf_upgrade",
			TrustLevel:         "official",
			Summary:            fmt.Sprintf("DNF upgrade (%s)", repo),
		})
	}
	return out
}

func (d *DnfUpdateProvider) Update(ctx context.Context, offerID string) error {
	if d.txMgr == nil {
		return fmt.Errorf("transaction manager unavailable")
	}
	_, err := d.txMgr.ExecuteOfferTransaction(ctx, offerID, transaction.OpUpdate, nil)
	return err
}

// ZypperUpdateProvider discovers pending openSUSE updates via `zypper --non-interactive list-updates`.
type ZypperUpdateProvider struct {
	txMgr  *transaction.Manager
	runner CommandRunner
}

func NewZypperUpdateProvider(txMgr *transaction.Manager) *ZypperUpdateProvider {
	return &ZypperUpdateProvider{txMgr: txMgr, runner: utils.RunCommand}
}

func NewZypperUpdateProviderWithRunner(txMgr *transaction.Manager, runner CommandRunner) *ZypperUpdateProvider {
	if runner == nil {
		runner = utils.RunCommand
	}
	return &ZypperUpdateProvider{txMgr: txMgr, runner: runner}
}

func (z *ZypperUpdateProvider) Name() string { return "zypper" }

func (z *ZypperUpdateProvider) CheckUpdates(ctx context.Context) ([]AvailableUpdate, error) {
	if z.runner == nil {
		z.runner = utils.RunCommand
	}
	if !utils.CommandExists("zypper") && fmt.Sprintf("%p", z.runner) == fmt.Sprintf("%p", utils.RunCommand) {
		return nil, fmt.Errorf("zypper not installed on host")
	}
	res := z.runner(ctx, "zypper", "--non-interactive", "list-updates")
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("zypper list-updates failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return ParseZypperUpdatesOutput(res.Stdout), nil
}

// ParseZypperUpdatesOutput parses table rows `v | Repo | Name | Current Version | Available Version | Arch`.
func ParseZypperUpdatesOutput(output string) []AvailableUpdate {
	resolver := identity.NewResolver(identity.GetDefaultRegistry())
	var out []AvailableUpdate
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, "|") || strings.HasPrefix(line, "S ") || strings.HasPrefix(line, "--") {
			continue
		}
		cols := strings.Split(line, "|")
		if len(cols) < 5 {
			continue
		}
		repo := strings.TrimSpace(cols[1])
		name := strings.TrimSpace(cols[2])
		oldVer := strings.TrimSpace(cols[3])
		newVer := strings.TrimSpace(cols[4])
		if name == "" || strings.EqualFold(name, "Name") {
			continue
		}
		if repo == "" {
			repo = "oss"
		}
		res := resolver.Resolve(name, "zypper")
		out = append(out, AvailableUpdate{
			OfferID:            fmt.Sprintf("zypper:%s:%s", strings.ToLower(repo), strings.ToLower(name)),
			AppID:              res.CanonicalID,
			PackageName:        name,
			DisplayName:        pickDisplayName(res.DisplayName, name),
			Provider:           "zypper",
			Repository:         repo,
			InstallScope:       "system",
			InstalledVersion:   oldVer,
			AvailableVersion:   newVer,
			DownloadSize:       "~22 MB",
			IsSecurity:         isSecuritySensitivePkg(name),
			PartialUpgradeSafe: true,
			UpgradeGroupPolicy: "zypper_upgrade",
			TrustLevel:         "official",
			Summary:            fmt.Sprintf("Zypper upgrade (%s)", repo),
		})
	}
	return out
}

func (z *ZypperUpdateProvider) Update(ctx context.Context, offerID string) error {
	if z.txMgr == nil {
		return fmt.Errorf("transaction manager unavailable")
	}
	_, err := z.txMgr.ExecuteOfferTransaction(ctx, offerID, transaction.OpUpdate, nil)
	return err
}
