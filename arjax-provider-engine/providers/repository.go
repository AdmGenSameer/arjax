package providers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

type RepositoryProvider struct {
	pm string // pacman, apt, dnf, zypper
}

func NewRepositoryProvider(preferredPM string) *RepositoryProvider {
	pm := preferredPM
	if pm == "" {
		pm = utils.DetectPackageManager()
	}
	return &RepositoryProvider{pm: pm}
}

func (rp *RepositoryProvider) Name() string {
	return "repository"
}

// PackageManager returns the underlying distro package manager binary (pacman, paru, apt, dnf, zypper).
func (rp *RepositoryProvider) PackageManager() string {
	return rp.pm
}

func (rp *RepositoryProvider) IsSupported() bool {
	return rp.pm != "" && utils.CommandExists(rp.pm)
}

func (rp *RepositoryProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	if !rp.IsSupported() {
		return nil, fmt.Errorf("repository provider not supported on this system")
	}

	terms := strings.Fields(strings.TrimSpace(query))
	if len(terms) == 0 {
		return nil, nil
	}

	var args []string
	switch rp.pm {
	case "paru", "pacman":
		args = append([]string{"-Ss"}, terms...)
	case "apt":
		args = append([]string{"search"}, terms...)
	case "dnf":
		args = append([]string{"search"}, terms...)
	case "zypper":
		args = append([]string{"search"}, terms...)
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", rp.pm)
	}

	res := utils.RunCommand(ctx, rp.pm, args...)
	if len(terms) > 1 && (res.Stdout == "" || strings.TrimSpace(res.Stdout) == "") {
		// Fallback to searching primary term if multi-term search yielded no results
		var fallbackArgs []string
		switch rp.pm {
		case "paru", "pacman":
			fallbackArgs = []string{"-Ss", terms[0]}
		case "apt", "dnf", "zypper":
			fallbackArgs = []string{"search", terms[0]}
		}
		if len(fallbackArgs) > 0 {
			fallbackRes := utils.RunCommand(ctx, rp.pm, fallbackArgs...)
			if fallbackRes.Stdout != "" {
				res = fallbackRes
			}
		}
	}

	if res.Error != nil && res.ExitCode != 0 && res.Stdout == "" {
		return nil, res.Error
	}

	return rp.parseSearchResults(res.Stdout, query), nil
}

func (rp *RepositoryProvider) parseSearchResults(output string, query string) []models.Package {
	var pkgs []models.Package
	lines := strings.Split(output, "\n")

	switch rp.pm {
	case "paru", "pacman":
		for i := 0; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 && strings.Contains(parts[0], "/") {
				repoPkg := strings.Split(parts[0], "/")
				repo := repoPkg[0]
				name := repoPkg[1]
				version := parts[1]
				desc := ""
				if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "    ") {
					desc = strings.TrimSpace(lines[i+1])
					i++
				}
				pkgs = append(pkgs, models.Package{
					Name:        name,
					Version:     version,
					Description: desc,
					Provider:    "repository",
					Source:      rp.pm,
					Repo:        repo,
				})
			}
		}
	case "apt":
		for i := 0; i < len(lines); i++ {
			rawLine := lines[i]
			line := strings.TrimSpace(rawLine)
			if line == "" || strings.HasPrefix(line, "Sorting...") || strings.HasPrefix(line, "Full Text Search...") {
				continue
			}
			// apt search package headers are unindented and formatted as:
			// name/suite[,now] version arch [installed]
			if !strings.HasPrefix(rawLine, " ") && !strings.HasPrefix(rawLine, "\t") {
				parts := strings.Fields(line)
				if len(parts) >= 1 && strings.Contains(parts[0], "/") {
					slashIdx := strings.Index(parts[0], "/")
					name := parts[0][:slashIdx]
					suite := strings.SplitN(parts[0][slashIdx+1:], ",", 2)[0]
					if suite == "" {
						suite = "apt"
					}
					version := ""
					if len(parts) >= 2 {
						version = parts[1]
					}
					arch := ""
					if len(parts) >= 3 && !strings.HasPrefix(parts[2], "[") {
						arch = parts[2]
					}
					desc := ""
					if i+1 < len(lines) && (strings.HasPrefix(lines[i+1], "  ") || strings.HasPrefix(lines[i+1], "\t")) {
						desc = strings.TrimSpace(lines[i+1])
						i++
					}
					if name != "" {
						pkgs = append(pkgs, models.Package{
							Name:        name,
							Version:     version,
							Arch:        arch,
							Description: desc,
							Provider:    "repository",
							Source:      "apt",
							Repo:        suite,
						})
					}
				}
			}
		}
	default:
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Sorting...") || strings.HasPrefix(line, "Full Text Search...") || strings.HasPrefix(line, "==") || strings.HasPrefix(line, "--") {
				continue
			}
			var nameVer, desc string
			if strings.Contains(line, " : ") {
				parts := strings.SplitN(line, " : ", 2)
				nameVer = strings.TrimSpace(parts[0])
				desc = strings.TrimSpace(parts[1])
			} else if strings.Contains(line, " - ") {
				parts := strings.SplitN(line, " - ", 2)
				nameVer = strings.TrimSpace(parts[0])
				desc = strings.TrimSpace(parts[1])
			} else {
				nameVer = line
			}
			nameParts := strings.Fields(nameVer)
			if len(nameParts) > 0 {
				pkgs = append(pkgs, models.Package{
					Name:        nameParts[0],
					Description: desc,
					Provider:    "repository",
					Source:      rp.pm,
					Repo:        rp.pm,
				})
			}
		}
	}
	return pkgs
}

func runPrivilegedPM(ctx context.Context, bin string, args ...string) utils.ExecResult {
	if os.Geteuid() == 0 || bin == "paru" || bin == "yay" {
		return utils.RunCommand(ctx, bin, args...)
	}
	sudoArgs := append([]string{bin}, args...)
	return utils.RunCommand(ctx, "sudo", sudoArgs...)
}

func (rp *RepositoryProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	if !rp.IsSupported() {
		return models.Result{}, fmt.Errorf("repository provider not supported")
	}

	var installCmd string
	bin := rp.pm
	var args []string

	switch rp.pm {
	case "paru":
		installCmd = fmt.Sprintf("paru -S --noconfirm %s", pkg.Name)
		args = []string{"-S", "--noconfirm", pkg.Name}
	case "pacman":
		installCmd = fmt.Sprintf("sudo pacman -S --noconfirm %s", pkg.Name)
		args = []string{"-S", "--noconfirm", pkg.Name}
	case "apt":
		bin = "apt-get"
		installCmd = fmt.Sprintf("sudo apt-get install -y %s", pkg.Name)
		args = []string{"install", "-y", pkg.Name}
	case "dnf":
		installCmd = fmt.Sprintf("sudo dnf install -y %s", pkg.Name)
		args = []string{"install", "-y", pkg.Name}
	case "zypper":
		installCmd = fmt.Sprintf("sudo zypper install -n %s", pkg.Name)
		args = []string{"install", "-n", pkg.Name}
	}

	execRes := runPrivilegedPM(ctx, bin, args...)
	endTime := time.Now()
	duration := endTime.Sub(start).Milliseconds()

	success := execRes.ExitCode == 0
	errMsg := ""
	if !success {
		errMsg = execRes.Stderr
		if errMsg == "" {
			errMsg = fmt.Sprintf("command exited with code %d", execRes.ExitCode)
		}
	}

	return models.Result{
		Package:    pkg.Name,
		Provider:   "repository",
		Success:    success,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: duration,
		Output:     execRes.Stdout,
		Error:      errMsg,
		InstallCmd: installCmd,
	}, nil
}

func (rp *RepositoryProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	var removeCmd string
	bin := rp.pm
	var args []string

	switch rp.pm {
	case "paru":
		removeCmd = fmt.Sprintf("paru -R --noconfirm %s", pkg.Name)
		args = []string{"-R", "--noconfirm", pkg.Name}
	case "pacman":
		removeCmd = fmt.Sprintf("sudo pacman -R --noconfirm %s", pkg.Name)
		args = []string{"-R", "--noconfirm", pkg.Name}
	case "apt":
		bin = "apt-get"
		removeCmd = fmt.Sprintf("sudo apt-get remove -y %s", pkg.Name)
		args = []string{"remove", "-y", pkg.Name}
	case "dnf":
		removeCmd = fmt.Sprintf("sudo dnf remove -y %s", pkg.Name)
		args = []string{"remove", "-y", pkg.Name}
	case "zypper":
		removeCmd = fmt.Sprintf("sudo zypper remove -n %s", pkg.Name)
		args = []string{"remove", "-n", pkg.Name}
	}

	execRes := runPrivilegedPM(ctx, bin, args...)
	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "repository",
		Success:    execRes.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     execRes.Stdout,
		Error:      execRes.Stderr,
		InstallCmd: removeCmd,
	}, nil
}

// UpdatePackage upgrades an individual package on APT, DNF, or Zypper.
// Arch Linux (pacman/paru) intentionally forbids partial upgrades and requires full Update().
func (rp *RepositoryProvider) UpdatePackage(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	switch rp.pm {
	case "apt":
		res := runPrivilegedPM(ctx, "apt-get", "install", "--only-upgrade", "-y", pkg.Name)
		end := time.Now()
		return models.Result{
			Package:    pkg.Name,
			Provider:   "apt",
			Success:    res.ExitCode == 0,
			StartTime:  start,
			EndTime:    end,
			DurationMs: end.Sub(start).Milliseconds(),
			Output:     res.Stdout,
			Error:      res.Stderr,
			InstallCmd: fmt.Sprintf("sudo apt-get install --only-upgrade -y %s", pkg.Name),
		}, res.Error
	case "dnf":
		res := runPrivilegedPM(ctx, "dnf", "upgrade", "-y", pkg.Name)
		end := time.Now()
		return models.Result{
			Package:    pkg.Name,
			Provider:   "dnf",
			Success:    res.ExitCode == 0,
			StartTime:  start,
			EndTime:    end,
			DurationMs: end.Sub(start).Milliseconds(),
			Output:     res.Stdout,
			Error:      res.Stderr,
			InstallCmd: fmt.Sprintf("sudo dnf upgrade -y %s", pkg.Name),
		}, res.Error
	case "zypper":
		res := runPrivilegedPM(ctx, "zypper", "update", "-n", pkg.Name)
		end := time.Now()
		return models.Result{
			Package:    pkg.Name,
			Provider:   "zypper",
			Success:    res.ExitCode == 0,
			StartTime:  start,
			EndTime:    end,
			DurationMs: end.Sub(start).Milliseconds(),
			Output:     res.Stdout,
			Error:      res.Stderr,
			InstallCmd: fmt.Sprintf("sudo zypper update -n %s", pkg.Name),
		}, res.Error
	default:
		return rp.Update(ctx)
	}
}

func (rp *RepositoryProvider) Update(ctx context.Context) (models.Result, error) {
	start := time.Now()
	bin := rp.pm
	var args []string

	switch rp.pm {
	case "paru", "pacman":
		// Arch Linux official repo updates ALWAYS perform a full system upgrade (pacman -Syu)
		// without unconditional --noconfirm so replacements/conflicts cannot silently execute.
		bin = "sudo"
		args = []string{"pacman", "-Syu"}
	case "apt":
		bin = "sudo"
		args = []string{"apt-get", "upgrade", "-y"}
	case "dnf":
		bin = "sudo"
		args = []string{"dnf", "upgrade", "-y"}
	case "zypper":
		bin = "sudo"
		args = []string{"zypper", "update", "-n"}
	default:
		return models.Result{
			Package:  "system-update",
			Provider: "repository",
			Success:  false,
			Error:    fmt.Sprintf("unsupported repository update manager: %s", rp.pm),
		}, fmt.Errorf("unsupported repository update manager: %s", rp.pm)
	}

	execRes := utils.RunCommand(ctx, bin, args...)
	endTime := time.Now()

	return models.Result{
		Package:    "system-update",
		Provider:   "repository",
		Success:    execRes.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     execRes.Stdout,
		Error:      execRes.Stderr,
	}, nil
}

func (rp *RepositoryProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	var args []string
	switch rp.pm {
	case "paru", "pacman":
		args = []string{"-Si", pkgName}
	case "apt":
		args = []string{"show", pkgName}
	case "dnf":
		args = []string{"info", pkgName}
	case "zypper":
		args = []string{"info", pkgName}
	default:
		return models.PackageInfo{}, fmt.Errorf("unsupported package manager: %s", rp.pm)
	}

	if len(args) == 0 {
		return models.PackageInfo{}, fmt.Errorf("no arguments generated for package manager: %s", rp.pm)
	}

	res := utils.RunCommand(ctx, rp.pm, args...)
	if res.ExitCode != 0 {
		return models.PackageInfo{}, fmt.Errorf("info failed: %s", res.Stderr)
	}

	info := models.PackageInfo{
		Name:     pkgName,
		Provider: "repository",
	}

	lines := strings.Split(res.Stdout, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			switch strings.ToLower(key) {
			case "version":
				info.Version = val
			case "description":
				info.Description = val
			case "installed size", "download size", "size":
				info.Size = val
			case "repository", "repo":
				info.Repo = val
			case "depends on", "depends", "requires":
				if val != "" && strings.ToLower(val) != "none" {
					if strings.Contains(val, ",") {
						for _, seg := range strings.Split(val, ",") {
							if idx := strings.Index(seg, "("); idx >= 0 {
								seg = seg[:idx]
							}
							clean := strings.TrimSpace(seg)
							if idx := strings.Index(clean, "|"); idx >= 0 {
								clean = strings.TrimSpace(clean[:idx])
							}
							if clean != "" && clean != "None" {
								info.Dependencies = append(info.Dependencies, clean)
							}
						}
					} else {
						for _, rawDep := range strings.Fields(val) {
							clean := strings.Trim(rawDep, ",()")
							if clean != "" && clean != "None" && !strings.HasPrefix(clean, ">=") && !strings.HasPrefix(clean, "=") && !strings.HasPrefix(clean, "<=") {
								info.Dependencies = append(info.Dependencies, clean)
							}
						}
					}
				}
			}
		}
	}
	return info, nil
}

func (rp *RepositoryProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	var results []models.Result
	for _, pkg := range pkgs {
		res, err := rp.Install(ctx, pkg)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}
