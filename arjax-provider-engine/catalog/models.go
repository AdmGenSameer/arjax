package catalog

import (
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/identity"
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
	PolicyRecommended      RecommendationPolicy = "recommended"       // Best overall source based on trust & compatibility
	PolicyOfficialOnly     RecommendationPolicy = "official_only"     // Avoid community-maintained/third-party sources
	PolicyNativePreferred  RecommendationPolicy = "native_preferred"  // Prefers official distro repos (Pacman/APT/DNF)
	PolicySandboxPreferred RecommendationPolicy = "sandbox_preferred" // Prefers containerized runtimes (Flatpak/Snap)
	PolicyLatestVersion    RecommendationPolicy = "latest_version"    // Prefers highest compatible version using provider-specific comparator
	PolicyMinimalDeps      RecommendationPolicy = "minimal_deps"      // Prefers offers with lower additional dependency cost
)

// PackageOffer represents a distinct, uniquely identified distribution channel/format for an application.
// Authority note: executable commands are never accepted from the frontend; installations execute by OfferID.
type PackageOffer struct {
	ID                   string       `json:"id"`                   // Stable identity: provider:repo:pkg:arch:branch:channel
	AppID                string       `json:"appId"`                // Canonical AppStream ID, e.g. "org.mozilla.firefox"
	PackageName          string       `json:"packageName"`          // Technical package name
	Provider             string       `json:"provider"`             // "pacman", "aur", "flatpak", "snap", "deb", "rpm", "appimage", "github"
	Repository           string       `json:"repository"`           // "extra", "core", "flathub", "aur", "snapcraft"
	Source               string       `json:"source"`               // Compatibility alias for Repository
	Architecture         string       `json:"architecture"`         // "x86_64", "aarch64", "any", "noarch"
	Version              string       `json:"version"`              // Provider version string, e.g. "1:124.0.1-1"
	InstalledVersion     string       `json:"installedVersion"`     // Installed version if THIS specific offer is installed
	Branch               string       `json:"branch,omitempty"`     // Flatpak branch (e.g. "stable", "beta")
	Channel              string       `json:"channel,omitempty"`    // Snap tracking channel (e.g. "latest/stable", "latest/edge")
	DownloadSize         string       `json:"downloadSize"`         // Formatted download size
	InstalledSize        string       `json:"installedSize"`        // Formatted installed size
	InstallScope         InstallScope `json:"installScope"`         // Isolation level ("sandboxed", "system", "user")
	Scope                InstallScope `json:"scope"`                // Explicit scope field
	Permissions          []string     `json:"permissions"`          // Sandbox / host capability permissions
	DependencyCount      int          `json:"dependencyCount"`      // Estimated extra dependencies needed
	Dependencies         []string     `json:"dependencies,omitempty"`
	IsInstalled          bool         `json:"isInstalled"`          // Whether THIS specific offer is installed
	IsEligible           bool         `json:"isEligible"`           // Whether offer is eligible on current host under current policy
	IneligibleReason     string       `json:"ineligibleReason"`     // Explanation if not eligible
	TrustLevel           TrustLevel   `json:"trustLevel"`           // Separated trust category ("official", "verified", "community", ...)
	SignatureStatus      string       `json:"signatureStatus"`      // "verified-keyring", "ostree-signed", "unverified-script", etc.
	Trust                SourceTrust  `json:"trust"`                // Full provenance and trust metadata
	Score                float64      `json:"score"`                // Policy recommendation ranking score (0.0 - 100.0)
	RecommendationBadge  string       `json:"recommendationBadge"`  // "Recommended", "Official", "Sandboxed", "Community", "Installed", "Unavailable"
	RecommendationReason string       `json:"recommendationReason"` // Human-readable explanation of recommendation
	CommandPreview       string       `json:"commandPreview,omitempty"` // Read-only educational preview; never executed directly
}

// UnifiedApplication unifies multiple distinct offers under a single canonical AppStream identity
type UnifiedApplication struct {
	AppID              string                   `json:"appId"`
	Name               string                   `json:"name"`
	Summary            string                   `json:"summary"`
	Description        string                   `json:"description"`
	Developer          string                   `json:"developer"`
	Homepage           string                   `json:"homepage"`
	License            string                   `json:"license"`
	Icon               icons.IconResponseDTO    `json:"icon"`
	IconURL            string                   `json:"iconUrl"`
	Categories         []string                 `json:"categories"`
	Screenshots        []string                 `json:"screenshots"`
	Releases           []string                 `json:"releases"`
	Aliases            []string                 `json:"aliases"`
	IdentityConfidence identity.MatchConfidence `json:"identityConfidence"`
	Offers             []PackageOffer           `json:"offers"`
	DefaultOfferID     string                   `json:"defaultOfferId"`
	IsInstalled        bool                     `json:"isInstalled"`
	InstalledOfferID   string                   `json:"installedOfferId"`
	Rating             float64                  `json:"rating"`
	Reviews            string                   `json:"reviews"`
	Badge              string                   `json:"badge"`
}

// SearchEventBase provides ordering and query-isolation fields on all progressive search events
type SearchEventBase struct {
	SearchID   string `json:"searchId"`
	Sequence   int64  `json:"sequence"`
	Provider   string `json:"provider,omitempty"`
	Status     string `json:"status"` // "started" | "running" | "completed" | "failed" | "cancelled"
	DurationMs int64  `json:"durationMs"`
}

// SearchStartedDTO emits when a progressive search begins
type SearchStartedDTO struct {
	SearchEventBase
	Query     string   `json:"query"`
	Providers []string `json:"providers"`
}

// SearchProviderStatusDTO emits per-provider lifecycle updates
type SearchProviderStatusDTO struct {
	SearchEventBase
	ResultCount int    `json:"resultCount"`
	Error       string `json:"error,omitempty"`
}

// ProgressiveSearchBatchDTO emits incremental deltas (added/updated apps) over Wails runtime events
type ProgressiveSearchBatchDTO struct {
	SearchEventBase
	AddedApps   []UnifiedApplication `json:"addedApps"`             // Newly discovered applications in this batch
	UpdatedApps []UnifiedApplication `json:"updatedApps"`           // Existing applications enriched with new offers
	Apps        []UnifiedApplication `json:"apps,omitempty"`        // Full sorted snapshot for compatibility with sync/simple consumers
	IsFinal     bool                 `json:"isFinal"`
}

// ProgressiveSearchDoneDTO signals termination of a progressive search
type ProgressiveSearchDoneDTO struct {
	SearchEventBase
	TotalApps int  `json:"totalApps"`
	Cancelled bool `json:"cancelled"`
}

// ProgressiveSearchCallbacks defines all expressive event hooks for progressive search
type ProgressiveSearchCallbacks struct {
	OnStarted        func(SearchStartedDTO)
	OnProviderStatus func(SearchProviderStatusDTO)
	OnBatch          func(ProgressiveSearchBatchDTO)
	OnProviderError  func(SearchProviderStatusDTO)
	OnDone           func(ProgressiveSearchDoneDTO)
	OnCancelled      func(ProgressiveSearchDoneDTO)
}

// CatalogConfig holds configuration for the catalog and recommendation engine
type CatalogConfig struct {
	Policy             RecommendationPolicy `json:"policy"`
	PreferredPM        string               `json:"preferredPm"`
	PreferVerified     bool                 `json:"preferVerified"`
	MaxSearchResults   int                  `json:"maxSearchResults"`
	ProviderTimeoutSec int                  `json:"providerTimeoutSec"`
}

// DefaultCatalogConfig returns standard defaults for Arch Linux environments
func DefaultCatalogConfig() CatalogConfig {
	return CatalogConfig{
		Policy:             PolicyRecommended,
		PreferredPM:        "pacman",
		PreferVerified:     true,
		MaxSearchResults:   50,
		ProviderTimeoutSec: 8,
	}
}

var _ = time.Now
