package providers

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// SupportLevel expresses how completely a provider capability is implemented
type SupportLevel string

const (
	SupportFull    SupportLevel = "implemented"
	SupportPartial SupportLevel = "partial"
	SupportPlanned SupportLevel = "planned"
)

// ProviderCapability explicitly tracks what operations a package ecosystem genuinely supports
// so data-model presence is never mistaken for completed integration.
type ProviderCapability struct {
	Provider           string       `json:"provider"`
	DisplayName        string       `json:"displayName"`
	Search             SupportLevel `json:"search"`
	Install            SupportLevel `json:"install"`
	Remove             SupportLevel `json:"remove"`
	Update             SupportLevel `json:"update"`
	InstalledDetection SupportLevel `json:"installedDetection"`
	DependencyPreview  SupportLevel `json:"dependencyPreview"`
	LockDetection      bool         `json:"lockDetection"`
	OfficialDistro     bool         `json:"officialDistro"`
	AvailableOnHost    bool         `json:"availableOnHost"`
	StatusNote         string       `json:"statusNote"`
}

// ProviderStatus separates code-level Arjax support from whether a provider is installed,
// available, and currently healthy on the user's host (e.g., Snap installed vs snapd service down).
type ProviderStatus struct {
	Provider        string             `json:"provider"`
	DisplayName     string             `json:"displayName"`
	Supported       bool               `json:"supported"`       // Arjax has a provider implementation
	Installed       bool               `json:"installed"`       // Client binary / tooling present on host
	Available       bool               `json:"available"`       // Usable for at least read/search operations
	Healthy         bool               `json:"healthy"`         // Service/DB/remotes responding cleanly
	StatusMessage   string             `json:"statusMessage"`   // Distinct actionable UI message
	Capabilities    ProviderCapability `json:"capabilities"`    // Underlying capability breakdown
	LastHealthCheck time.Time          `json:"lastHealthCheck"` // Timestamp of last probe
}

// CanPerform checks if the provider supports a given operation and is available on host.
func (c ProviderCapability) CanPerform(op string) (bool, string) {
	if !c.AvailableOnHost {
		if c.Search == SupportPlanned && c.Install == SupportPlanned {
			return false, c.DisplayName + " provider integration is planned and not yet implemented"
		}
		return false, c.DisplayName + " is supported by Arjax, but not installed or enabled on this host"
	}
	var level SupportLevel
	switch strings.ToLower(op) {
	case "search":
		level = c.Search
	case "install":
		level = c.Install
	case "remove":
		level = c.Remove
	case "update":
		level = c.Update
	case "installed":
		level = c.InstalledDetection
	default:
		level = SupportPlanned
	}
	if level == SupportPlanned {
		return false, c.DisplayName + " " + op + " is not yet implemented"
	}
	return true, ""
}

// DefaultCapabilityMatrix returns the explicit, honest capability matrix across all tracked ecosystems.
func DefaultCapabilityMatrix() map[string]ProviderCapability {
	hasPacman := utils.CommandExists("pacman")
	hasParuOrYay := utils.CommandExists("paru") || utils.CommandExists("yay") || utils.CommandExists("makepkg")
	hasFlatpak := utils.CommandExists("flatpak")
	hasSnap := utils.CommandExists("snap")
	hasApt := utils.CommandExists("apt")
	hasDnf := utils.CommandExists("dnf")
	hasZypper := utils.CommandExists("zypper")

	pacmanCap := ProviderCapability{
		Provider:           "pacman",
		DisplayName:        "Pacman (Arch Official)",
		Search:             SupportFull,
		Install:            SupportFull,
		Remove:             SupportFull,
		Update:             SupportFull,
		InstalledDetection: SupportFull,
		DependencyPreview:  SupportPartial,
		LockDetection:      true,
		OfficialDistro:     true,
		AvailableOnHost:    hasPacman,
		StatusNote:         "Native Arch repositories (core, extra, multilib)",
	}
	if !hasPacman {
		pacmanCap.StatusNote = "Supported by Arjax, but pacman is not installed on this host"
	}

	aurCap := ProviderCapability{
		Provider:           "aur",
		DisplayName:        "AUR (Arch User Repository)",
		Search:             SupportFull,
		Install:            SupportFull,
		Remove:             SupportFull,
		Update:             SupportPartial,
		InstalledDetection: SupportFull,
		DependencyPreview:  SupportPartial,
		LockDetection:      true,
		OfficialDistro:     false,
		AvailableOnHost:    hasParuOrYay || hasPacman,
		StatusNote:         "Community PKGBUILD recipes via AUR helper / makepkg",
	}
	if !aurCap.AvailableOnHost {
		aurCap.StatusNote = "Requires Arch Linux with paru, yay, or makepkg"
	}

	flatpakCap := ProviderCapability{
		Provider:           "flatpak",
		DisplayName:        "Flatpak (Flathub)",
		Search:             SupportFull,
		Install:            SupportFull,
		Remove:             SupportFull,
		Update:             SupportFull,
		InstalledDetection: SupportFull,
		DependencyPreview:  SupportPartial,
		LockDetection:      false,
		OfficialDistro:     false,
		AvailableOnHost:    hasFlatpak,
		StatusNote:         "Supports default --user and --system Flatpak installations; named --installation=NAME remotes not supported in v1.0",
	}
	if !hasFlatpak {
		flatpakCap.StatusNote = "Supported by Arjax, but Flatpak is not installed on this host"
	}

	snapCap := ProviderCapability{
		Provider:           "snap",
		DisplayName:        "Snapcraft",
		Search:             SupportFull,
		Install:            SupportFull,
		Remove:             SupportFull,
		Update:             SupportPartial,
		InstalledDetection: SupportFull,
		DependencyPreview:  SupportPlanned,
		LockDetection:      false,
		OfficialDistro:     false,
		AvailableOnHost:    hasSnap,
		StatusNote:         "Canonical Snap packages via snapd",
	}
	if !hasSnap {
		snapCap.StatusNote = "Supported by Arjax, but Snap is not installed on this host"
	}

	rpmCap := ProviderCapability{
		Provider:           "rpm",
		DisplayName:        "DNF / RPM",
		Search:             SupportFull,
		Install:            SupportFull,
		Remove:             SupportFull,
		Update:             SupportFull,
		InstalledDetection: SupportFull,
		DependencyPreview:  SupportPartial,
		LockDetection:      true,
		OfficialDistro:     hasDnf,
		AvailableOnHost:    hasDnf,
		StatusNote:         "Fedora / RHEL official DNF & RPM repositories",
	}
	if !hasDnf {
		rpmCap.StatusNote = "Supported by Arjax, but dnf is not installed on this host"
	}

	debCap := ProviderCapability{
		Provider:           "deb",
		DisplayName:        "APT / DEB",
		Search:             SupportFull,
		Install:            SupportFull,
		Remove:             SupportFull,
		Update:             SupportFull,
		InstalledDetection: SupportFull,
		DependencyPreview:  SupportPartial,
		LockDetection:      true,
		OfficialDistro:     hasApt,
		AvailableOnHost:    hasApt,
		StatusNote:         "Debian / Ubuntu / Kali APT & dpkg repositories",
	}
	if !hasApt {
		debCap.StatusNote = "Supported by Arjax, but apt is not installed on this host"
	}

	zypperCap := ProviderCapability{
		Provider:           "zypper",
		DisplayName:        "Zypper (openSUSE)",
		Search:             SupportFull,
		Install:            SupportFull,
		Remove:             SupportFull,
		Update:             SupportFull,
		InstalledDetection: SupportFull,
		DependencyPreview:  SupportPartial,
		LockDetection:      true,
		OfficialDistro:     hasZypper,
		AvailableOnHost:    hasZypper,
		StatusNote:         "openSUSE / SLES official Zypper repositories",
	}
	if !hasZypper {
		zypperCap.StatusNote = "Supported by Arjax, but zypper is not installed on this host"
	}

	appimageCap := ProviderCapability{
		Provider:           "appimage",
		DisplayName:        "AppImage",
		Search:             SupportPlanned,
		Install:            SupportPlanned,
		Remove:             SupportPlanned,
		Update:             SupportPlanned,
		InstalledDetection: SupportPlanned,
		DependencyPreview:  SupportPlanned,
		LockDetection:      false,
		OfficialDistro:     false,
		AvailableOnHost:    false,
		StatusNote:         "Planned: portable AppImage lifecycle not yet implemented",
	}

	githubCap := ProviderCapability{
		Provider:           "github",
		DisplayName:        "GitHub Releases",
		Search:             SupportPartial,
		Install:            SupportPartial,
		Remove:             SupportPlanned,
		Update:             SupportPlanned,
		InstalledDetection: SupportPlanned,
		DependencyPreview:  SupportPlanned,
		LockDetection:      false,
		OfficialDistro:     false,
		AvailableOnHost:    true,
		StatusNote:         "Direct upstream GitHub release binary / archive fetching",
	}

	return map[string]ProviderCapability{
		"pacman":   pacmanCap,
		"aur":      aurCap,
		"deb":      debCap,
		"rpm":      rpmCap,
		"zypper":   zypperCap,
		"flatpak":  flatpakCap,
		"snap":     snapCap,
		"appimage": appimageCap,
		"github":   githubCap,
	}
}

// ProbeProviderStatuses runs fast, explicit host availability & health checks separately from
// static capability definitions (e.g. distinguishing "Snap not installed" vs "Snapd daemon down").
func ProbeProviderStatuses(ctx context.Context, matrix map[string]ProviderCapability) []ProviderStatus {
	if matrix == nil {
		matrix = DefaultCapabilityMatrix()
	}
	now := time.Now().UTC()
	ordered := GetCapabilitiesOrdered(matrix)
	statuses := make([]ProviderStatus, 0, len(ordered))

	for _, capInfo := range ordered {
		st := ProviderStatus{
			Provider:        capInfo.Provider,
			DisplayName:     capInfo.DisplayName,
			Capabilities:    capInfo,
			LastHealthCheck: now,
		}

		isPlannedOnly := capInfo.Search == SupportPlanned && capInfo.Install == SupportPlanned
		st.Supported = !isPlannedOnly

		switch capInfo.Provider {
		case "pacman":
			st.Installed = utils.CommandExists("pacman")
			if !st.Installed {
				st.StatusMessage = "Pacman supported, but not installed on this distribution"
			} else if _, err := os.Stat("/var/lib/pacman/db.lck"); err == nil {
				st.Available = true
				st.Healthy = false
				st.StatusMessage = "Pacman available, but database lock (/var/lib/pacman/db.lck) is active"
			} else {
				st.Available = true
				st.Healthy = true
				st.StatusMessage = "Healthy · Official Arch repositories accessible"
			}

		case "aur":
			st.Installed = utils.CommandExists("paru") || utils.CommandExists("yay") || utils.CommandExists("makepkg")
			if !st.Installed {
				st.StatusMessage = "AUR supported, but no AUR helper (paru/yay) or makepkg is installed"
			} else {
				st.Available = true
				st.Healthy = true
				if utils.CommandExists("paru") {
					st.StatusMessage = "Healthy · AUR RPC search & paru builder ready"
				} else {
					st.StatusMessage = "Healthy · AUR RPC search & makepkg available"
				}
			}

		case "flatpak":
			st.Installed = utils.CommandExists("flatpak")
			if !st.Installed {
				st.StatusMessage = "Flatpak is supported by Arjax, but not installed on host"
			} else {
				subCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				out, err := exec.CommandContext(subCtx, "flatpak", "remotes").Output()
				cancel()
				st.Available = true
				if err == nil && len(strings.TrimSpace(string(out))) > 0 {
					st.Healthy = true
					st.StatusMessage = "Healthy · Flatpak remotes configured"
				} else {
					st.Healthy = false
					st.StatusMessage = "Flatpak installed, but no remote (such as flathub) is configured"
				}
			}

		case "snap":
			st.Installed = utils.CommandExists("snap")
			if !st.Installed {
				st.StatusMessage = "Snap is supported by Arjax, but snapd is not installed on host"
			} else {
				// Distinguish binary installed vs snapd daemon socket responding!
				_, sockErr := os.Stat("/run/snapd.socket")
				if sockErr != nil {
					st.Available = false
					st.Healthy = false
					st.StatusMessage = "Snap CLI installed, but snapd service socket is unavailable"
				} else {
					st.Available = true
					st.Healthy = true
					st.StatusMessage = "Healthy · snapd service active"
				}
			}

		case "deb":
			st.Installed = utils.CommandExists("apt") && utils.CommandExists("dpkg-query")
			if !st.Installed {
				st.StatusMessage = "APT/DEB supported, but apt is not installed on this distribution"
			} else {
				st.Available = true
				st.Healthy = true
				st.StatusMessage = "Healthy · APT/DEB repositories accessible"
			}

		case "rpm":
			st.Installed = utils.CommandExists("dnf") && utils.CommandExists("rpm")
			if !st.Installed {
				st.StatusMessage = "DNF/RPM supported, but dnf is not installed on this distribution"
			} else if _, err := os.Stat("/var/run/dnf.pid"); err == nil {
				st.Available = true
				st.Healthy = false
				st.StatusMessage = "DNF available, but transaction lock (/var/run/dnf.pid) is active"
			} else {
				st.Available = true
				st.Healthy = true
				st.StatusMessage = "Healthy · DNF/RPM repositories accessible"
			}

		case "zypper":
			st.Installed = utils.CommandExists("zypper")
			if !st.Installed {
				st.StatusMessage = "Zypper supported, but zypper is not installed on this distribution"
			} else if _, err := os.Stat("/var/run/zypp.pid"); err == nil {
				st.Available = true
				st.Healthy = false
				st.StatusMessage = "Zypper available, but lock (/var/run/zypp.pid) is active"
			} else {
				st.Available = true
				st.Healthy = true
				st.StatusMessage = "Healthy · Zypper repositories accessible"
			}

		default:
			st.Installed = capInfo.AvailableOnHost
			st.Available = capInfo.AvailableOnHost && st.Supported
			st.Healthy = st.Available
			st.StatusMessage = capInfo.StatusNote
		}

		statuses = append(statuses, st)
	}

	return statuses
}

// GetCapabilitiesOrdered returns all tracked provider capabilities in deterministic priority order.
func GetCapabilitiesOrdered(matrix map[string]ProviderCapability) []ProviderCapability {
	order := []string{"pacman", "aur", "deb", "rpm", "zypper", "flatpak", "snap", "appimage", "github"}
	out := make([]ProviderCapability, 0, len(order))
	for _, k := range order {
		if c, ok := matrix[k]; ok {
			out = append(out, c)
		}
	}
	return out
}
