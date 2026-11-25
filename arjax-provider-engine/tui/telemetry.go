package tui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/transaction"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// HostTelemetry captures live host metadata, lock status, and memory stats for the TUI bars.
type HostTelemetry struct {
	Hostname   string
	Arch       string
	Kernel     string
	LockPath   string
	Locked     bool
	MemUsedGB  float64
	MemTotalGB float64
	TermName   string
}

// InstalledEntry represents a host-installed package item for the Installed & Health tab.
type InstalledEntry struct {
	ID       string
	Name     string
	Version  string
	Provider string
	Scope    string
	Format   string
}

// PermissionRow describes a structured sandbox or host permission badge for the Inspector.
type PermissionRow struct {
	Granted bool
	Label   string
}

func collectHostTelemetry(pmProvider string) HostTelemetry {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "linux-host"
	}

	arch := "x86_64"
	if runtime.GOARCH == "arm64" {
		arch = "aarch64"
	} else if runtime.GOARCH != "amd64" {
		arch = runtime.GOARCH
	}

	kernel := "linux"
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
			kernel = trimmed
		}
	}

	lockTarget := pmProvider
	if lockTarget == "" {
		lockTarget = utils.DetectPackageManager()
	}
	locked, lockPath := transaction.DefaultHostLockChecker(lockTarget)
	if lockPath == "" {
		switch strings.ToLower(lockTarget) {
		case "apt", "deb":
			lockPath = "/var/lib/dpkg/lock-frontend"
		case "dnf", "rpm":
			lockPath = "/var/run/dnf.pid"
		case "zypper":
			lockPath = "/var/run/zypp.pid"
		default:
			lockPath = "/var/lib/pacman/db.lck"
		}
	}

	usedGB, totalGB := readProcMemGB()
	termName := os.Getenv("TERM")
	if termName == "" {
		termName = "xterm-256color"
	}

	return HostTelemetry{
		Hostname:   host,
		Arch:       arch,
		Kernel:     kernel,
		LockPath:   lockPath,
		Locked:     locked,
		MemUsedGB:  usedGB,
		MemTotalGB: totalGB,
		TermName:   termName,
	}
}

func readProcMemGB() (usedGB float64, totalGB float64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	var totalKB, availKB float64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			totalKB = val
		case "MemAvailable:":
			availKB = val
		}
		if totalKB > 0 && availKB > 0 {
			break
		}
	}
	if totalKB <= 0 {
		return 0, 0
	}
	totalGB = totalKB / (1024 * 1024)
	usedGB = (totalKB - availKB) / (1024 * 1024)
	if usedGB < 0 {
		usedGB = 0
	}
	return usedGB, totalGB
}

// evaluateOfferTrustScore calculates a 0-100 display integrity score & badge from SourceTrust metadata.
func evaluateOfferTrustScore(offer catalog.PackageOffer) (score int, badge string) {
	switch offer.Trust.TrustLevel {
	case catalog.TrustOfficial:
		score = 96
		badge = "[HIGH TRUST]"
	case catalog.TrustVerified:
		score = 92
		badge = "[HIGH TRUST]"
	case catalog.TrustCommunity:
		score = 68
		badge = "[COMMUNITY]"
	case catalog.TrustThirdParty:
		score = 60
		badge = "[THIRD-PARTY]"
	default:
		score = 45
		badge = "[UNVERIFIED]"
	}
	if offer.Trust.SignatureChecked && score < 95 {
		score += 2
	}
	if offer.Trust.OfficialRepo && score < 98 {
		score += 2
	}
	if score > 100 {
		score = 100
	}
	return score, badge
}

func formatOfferPermissions(offer catalog.PackageOffer) []PermissionRow {
	rows := make([]PermissionRow, 0, 4)
	seen := make(map[string]bool)
	add := func(granted bool, label string) {
		if !seen[label] {
			seen[label] = true
			rows = append(rows, PermissionRow{Granted: granted, Label: label})
		}
	}

	if offer.InstallScope == catalog.ScopeSandboxed || offer.Provider == "flatpak" || offer.Provider == "snap" {
		for _, p := range offer.Permissions {
			switch strings.ToLower(p) {
			case "network":
				add(true, "Network Access (remote fetch)")
			case "pulseaudio":
				add(true, "PipeWire / PulseAudio")
			case "wayland/x11":
				add(true, "Wayland + X11 Display socket")
			case "gpu-acceleration":
				add(true, "GPU Hardware Acceleration")
			case "removable-media":
				add(true, "Removable Media Access")
			case "gamepad":
				add(true, "Controller / Gamepad Input")
			case "ipc":
				add(true, "D-Bus / IPC Portal")
			case "desktop":
				add(true, "Desktop Portal Integration")
			}
		}
		if len(rows) == 0 {
			add(true, "Sandboxed App Runtime")
			add(true, "Desktop Portal Integration")
		}
		add(false, "Host Root Filesystem (Blocked)")
		return rows
	}

	add(true, "System-Native Host Linkage")
	add(true, "Host Filesystem (/usr /etc)")
	add(true, "Direct Hardware & GPU Access")
	return rows
}

func discoverHostInstalled(ctx context.Context) ([]InstalledEntry, map[string]string) {
	subCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	var entries []InstalledEntry
	installedMap := make(map[string]string)

	record := func(e InstalledEntry) {
		entries = append(entries, e)
		prov := strings.ToLower(e.Provider)
		if e.ID != "" {
			idLow := strings.ToLower(e.ID)
			installedMap[idLow] = e.Version
			installedMap[prov+":"+idLow] = e.Version
		}
		if e.Name != "" {
			nameLow := strings.ToLower(e.Name)
			installedMap[nameLow] = e.Version
			installedMap[prov+":"+nameLow] = e.Version
		}
	}

	if utils.CommandExists("flatpak") {
		for _, scope := range []string{"user", "system"} {
			cmd := exec.CommandContext(subCtx, "flatpak", "list", "--app", "--"+scope, "--columns=name,application,version")
			out, err := cmd.Output()
			if err != nil {
				continue
			}
			formatLabel := fmt.Sprintf("Flatpak (%s)", scope)
			scanner := bufio.NewScanner(strings.NewReader(string(out)))
			for scanner.Scan() {
				parts := strings.Split(scanner.Text(), "\t")
				if len(parts) < 2 {
					continue
				}
				name := strings.TrimSpace(parts[0])
				appID := strings.TrimSpace(parts[1])
				ver := ""
				if len(parts) >= 3 {
					ver = strings.TrimSpace(parts[2])
				}
				if appID == "" {
					continue
				}
				record(InstalledEntry{
					ID:       appID,
					Name:     name,
					Version:  ver,
					Provider: "flatpak",
					Scope:    scope,
					Format:   formatLabel,
				})
			}
		}
	}

	pm := utils.DetectPackageManager()
	switch pm {
	case "apt":
		if utils.CommandExists("dpkg-query") {
			cmd := exec.CommandContext(subCtx, "dpkg-query", "-W", "-f=${Package}\t${Version}\n")
			if out, err := cmd.Output(); err == nil {
				scanner := bufio.NewScanner(strings.NewReader(string(out)))
				for scanner.Scan() && len(entries) < 250 {
					parts := strings.SplitN(strings.TrimSpace(scanner.Text()), "\t", 2)
					if len(parts) == 2 && parts[0] != "" {
						record(InstalledEntry{ID: parts[0], Name: parts[0], Version: parts[1], Provider: "apt", Scope: "system", Format: "Native APT"})
					}
				}
			}
		}
	case "dnf", "zypper":
		if utils.CommandExists("rpm") {
			cmd := exec.CommandContext(subCtx, "rpm", "-qa", "--qf", "%{NAME}\t%{VERSION}-%{RELEASE}\n")
			if out, err := cmd.Output(); err == nil {
				scanner := bufio.NewScanner(strings.NewReader(string(out)))
				for scanner.Scan() && len(entries) < 250 {
					parts := strings.SplitN(strings.TrimSpace(scanner.Text()), "\t", 2)
					if len(parts) == 2 && parts[0] != "" {
						record(InstalledEntry{ID: parts[0], Name: parts[0], Version: parts[1], Provider: pm, Scope: "system", Format: "Native " + strings.ToUpper(pm)})
					}
				}
			}
		}
	default:
		if utils.CommandExists("pacman") {
			cmd := exec.CommandContext(subCtx, "pacman", "-Qe")
			if out, err := cmd.Output(); err == nil {
				scanner := bufio.NewScanner(strings.NewReader(string(out)))
				for scanner.Scan() && len(entries) < 250 {
					parts := strings.Fields(strings.TrimSpace(scanner.Text()))
					if len(parts) >= 2 {
						record(InstalledEntry{ID: parts[0], Name: parts[0], Version: parts[1], Provider: "pacman", Scope: "system", Format: "Native Pacman"})
					}
				}
			}
		}
	}

	return entries, installedMap
}

func truncateText(s string, maxLen int) string {
	if maxLen <= 1 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

func clampStyledWidth(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(maxW).Render(s)
}
