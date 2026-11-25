package providers

import (
	"strings"

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

// CanPerform checks if the provider supports a given operation and is available on host.
func (c ProviderCapability) CanPerform(op string) (bool, string) {
	if !c.AvailableOnHost {
		if c.Search == SupportPlanned && c.Install == SupportPlanned {
			return false, c.DisplayName + " provider integration is planned and not yet implemented"
		}
		return false, c.DisplayName + " is not installed or available on this host"
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
		pacmanCap.StatusNote = "Pacman binary not present on this host"
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
		StatusNote:         "Community-maintained PKGBUILD recipes (inspect before building)",
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
		StatusNote:         "Sandboxed desktop runtimes and Flathub remote",
	}
	if !hasFlatpak {
		flatpakCap.StatusNote = "Flatpak CLI not installed on this host"
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
		snapCap.StatusNote = "Snapd daemon/CLI not installed on this host"
	}

	rpmCap := ProviderCapability{
		Provider:           "rpm",
		DisplayName:        "DNF / RPM",
		Search:             SupportPartial,
		Install:            SupportPartial,
		Remove:             SupportPartial,
		Update:             SupportPlanned,
		InstalledDetection: SupportPlanned,
		DependencyPreview:  SupportPlanned,
		LockDetection:      false,
		OfficialDistro:     hasDnf,
		AvailableOnHost:    hasDnf,
		StatusNote:         "Basic dnf repository search/install; full transaction & update hooks pending Fedora verification",
	}

	debCap := ProviderCapability{
		Provider:           "deb",
		DisplayName:        "APT / DEB",
		Search:             SupportPartial,
		Install:            SupportPartial,
		Remove:             SupportPartial,
		Update:             SupportPlanned,
		InstalledDetection: SupportPlanned,
		DependencyPreview:  SupportPlanned,
		LockDetection:      true,
		OfficialDistro:     hasApt,
		AvailableOnHost:    hasApt,
		StatusNote:         "Basic apt repository search/install; full transaction & update hooks pending Debian/Ubuntu verification",
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
		StatusNote:         "Planned: portable AppImage integration and desktop-file lifecycle not yet implemented",
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
		"flatpak":  flatpakCap,
		"snap":     snapCap,
		"rpm":      rpmCap,
		"deb":      debCap,
		"appimage": appimageCap,
		"github":   githubCap,
	}
}

// GetCapabilitiesOrdered returns all tracked provider capabilities in deterministic priority order.
func GetCapabilitiesOrdered(matrix map[string]ProviderCapability) []ProviderCapability {
	order := []string{"pacman", "aur", "flatpak", "snap", "deb", "rpm", "appimage", "github"}
	out := make([]ProviderCapability, 0, len(order))
	for _, k := range order {
		if c, ok := matrix[k]; ok {
			out = append(out, c)
		}
	}
	return out
}

