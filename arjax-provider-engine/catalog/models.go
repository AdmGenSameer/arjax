package catalog

import (
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/icons"
)

// InstallScope represents execution and sandbox isolation levels
type InstallScope string

const (
	ScopeSandboxed InstallScope = "sandboxed" // Flatpak, Snap strict
	ScopeSystem    InstallScope = "system"    // Pacman, APT, DNF (root/system)
	ScopeUser      InstallScope = "user"      // Flatpak user, AUR user build
)

// RecommendationPolicy represents preference profiles for ranking package offers
type RecommendationPolicy string

const (
	PolicyNativePreferred  RecommendationPolicy = "native_preferred"  // Prefers official distro repos (Pacman/APT/DNF)
	PolicySandboxPreferred RecommendationPolicy = "sandbox_preferred" // Prefers containerized runtimes (Flatpak/Snap)
	PolicyLatestVersion    RecommendationPolicy = "latest_version"    // Prefers highest semantic version
)

// PackageOffer represents a distinct distribution channel or package format for an application
type PackageOffer struct {
	ID                   string       `json:"id"`                   // Stable offer ID, e.g. "pacman:extra:firefox"
	AppID                string       `json:"appId"`                // Canonical AppStream ID, e.g. "org.mozilla.firefox"
	PackageName          string       `json:"packageName"`          // Technical package name, e.g. "firefox"
	Provider             string       `json:"provider"`             // "pacman", "aur", "flatpak", "snap", "github"
	Source               string       `json:"source"`               // "extra", "flathub", "aur", "snapcraft"
	Version              string       `json:"version"`              // Version string, e.g. "124.0.1-1"
	DownloadSize         string       `json:"downloadSize"`         // Formatted download size, e.g. "78 MB"
	InstalledSize        string       `json:"installedSize"`        // Formatted installed size, e.g. "210 MB"
	InstallScope         InstallScope `json:"installScope"`         // "sandboxed", "system", "user"
	Permissions          []string     `json:"permissions"`          // e.g. ["network", "pulseaudio", "wayland"]
	IsInstalled          bool         `json:"isInstalled"`          // Whether THIS specific offer is installed
	InstalledVersion     string       `json:"installedVersion"`     // Installed version if applicable
	IsEligible           bool         `json:"isEligible"`           // Whether offer can be installed on current host
	IneligibleReason     string       `json:"ineligibleReason"`     // Reason if not eligible
	Score                float64      `json:"score"`                // Recommendation ranking score (0.0 - 100.0)
	RecommendationBadge  string       `json:"recommendationBadge"`  // "Recommended", "Alternative", "Sandboxed", "Community", "Installed"
	RecommendationReason string       `json:"recommendationReason"` // Human-readable explanation for this offer
	InstallCmd           string       `json:"installCmd,omitempty"` // Preview command, e.g. "pacman -S firefox"
}

// UnifiedApplication unifies multiple package offers under a single canonical AppStream identity
type UnifiedApplication struct {
	AppID            string                `json:"appId"`            // Canonical reverse-DNS ID, e.g. "org.mozilla.firefox"
	Name             string                `json:"name"`             // Display name, e.g. "Firefox"
	Summary          string                `json:"summary"`          // One-line headline
	Description      string                `json:"description"`      // Full description / markdown
	Developer        string                `json:"developer"`        // Publisher or project group
	Homepage         string                `json:"homepage"`         // Official upstream website
	License          string                `json:"license"`          // SPDX license expression
	Icon             icons.IconResponseDTO `json:"icon"`             // Universal Icon metadata
	IconURL          string                `json:"iconUrl"`          // Direct icon URL or /api/icons endpoint
	Categories       []string              `json:"categories"`       // Desktop categories
	Screenshots      []string              `json:"screenshots"`      // List of screenshot image URLs
	Releases         []string              `json:"releases"`         // Recent release notes / changelogs
	Aliases          []string              `json:"aliases"`          // Alternative AppStream IDs and package names
	Offers           []PackageOffer        `json:"offers"`           // Distinct available offers across package managers
	DefaultOfferID   string                `json:"defaultOfferId"`   // The offer selected by default/recommendation
	IsInstalled      bool                  `json:"isInstalled"`      // True if ANY offer is installed
	InstalledOfferID string                `json:"installedOfferId"` // ID of the installed offer, if any
	Rating           float64               `json:"rating"`           // Community rating (0.0 - 5.0)
	Reviews          string                `json:"reviews"`          // Review count or summary string
	Badge            string                `json:"badge"`            // Presentation badge, e.g. "Verified"
}

// ProgressiveSearchBatchDTO is emitted over Wails runtime events as search results arrive
type ProgressiveSearchBatchDTO struct {
	SearchID   string               `json:"searchId"`
	Apps       []UnifiedApplication `json:"apps"`
	Provider   string               `json:"provider"`
	IsFinal    bool                 `json:"isFinal"`
	DurationMs int64                `json:"durationMs"`
}

// ProgressiveSearchDoneDTO signals the completion of all search providers
type ProgressiveSearchDoneDTO struct {
	SearchID   string `json:"searchId"`
	TotalApps  int    `json:"totalApps"`
	DurationMs int64  `json:"durationMs"`
}

// CatalogConfig holds configuration for the catalog and recommendation engine
type CatalogConfig struct {
	Policy             RecommendationPolicy `json:"policy"`
	PreferredPM        string               `json:"preferredPm"`
	PreferVerified     bool                 `json:"preferVerified"`
	MaxSearchResults   int                  `json:"maxSearchResults"`
	ProviderTimeoutSec int                  `json:"providerTimeoutSec"`
}

// DefaultCatalogConfig returns standard defaults for Arch/Linux environments
func DefaultCatalogConfig() CatalogConfig {
	return CatalogConfig{
		Policy:             PolicyNativePreferred,
		PreferredPM:        "pacman",
		PreferVerified:     true,
		MaxSearchResults:   50,
		ProviderTimeoutSec: 8,
	}
}

// Ensure timestamp imports are used if needed
var _ = time.Now

