package updates

import "time"

// AvailableUpdate describes a single upgradable package offer discovered on the host
type AvailableUpdate struct {
	OfferID            string `json:"offerId"`
	AppID              string `json:"appId"`
	PackageName        string `json:"packageName"`
	DisplayName        string `json:"displayName"`
	Provider           string `json:"provider"`
	Repository         string `json:"repository"`
	InstallScope       string `json:"installScope,omitempty"` // Default "user" or "system" for Flatpak, "system" for Pacman
	Branch             string `json:"branch,omitempty"`       // Flatpak branch
	InstalledVersion   string `json:"installedVersion"`
	AvailableVersion   string `json:"availableVersion"`
	DownloadSize       string `json:"downloadSize"`
	IsSecurity         bool   `json:"isSecurity"`
	PartialUpgradeSafe bool   `json:"partialUpgradeSafe"` // False for Arch Pacman (requires full pacman -Syu)
	UpgradeGroupPolicy string `json:"upgradeGroupPolicy"` // "full_system_upgrade" | "aur_helper_upgrade" | "scoped_ref" | "snap_refresh"
	TrustLevel         string `json:"trustLevel"`
	Summary            string `json:"summary"`
}

// UpdateSummary aggregates pending updates and per-provider status across ecosystems
type UpdateSummary struct {
	TotalCount         int               `json:"totalCount"`
	ProviderCount      int               `json:"providerCount"`
	SecurityCount      int               `json:"securityCount"`
	RequiresFullSysUpg bool              `json:"requiresFullSysUpgrade"`
	ByProvider         map[string]int    `json:"byProvider"`
	Updates            []AvailableUpdate `json:"updates"`
	ProviderErrors     map[string]string `json:"providerErrors,omitempty"`
	Notes              []string          `json:"notes,omitempty"`
	CheckedAt          time.Time         `json:"checkedAt"`
}

// TransactionPreview presents the resolved changes, download footprint, and safety warnings
// to the user for explicit approval prior to running system upgrades without --noconfirm.
type TransactionPreview struct {
	PreviewID               string            `json:"previewId"`
	RequestedOfferIDs       []string          `json:"requestedOfferIds"`
	EffectiveOfferIDs       []string          `json:"effectiveOfferIds"`
	PackagesToUpgrade       []AvailableUpdate `json:"packagesToUpgrade"`
	PackagesToInstall       []string          `json:"packagesToInstall"` // Newly pulled replacements/deps if reported
	PackagesToRemove        []string          `json:"packagesToRemove"`  // Replacements/conflicts if reported
	EstimatedDownloadSize   string            `json:"estimatedDownloadSize"`
	RequiresOfficialFirst   bool              `json:"requiresOfficialFirst"`   // True when AUR selected while Pacman pending
	IncludesFullSysUpgrade  bool              `json:"includesFullSysUpgrade"`  // True when full pacman -Syu runs
	IncludesAURBuild        bool              `json:"includesAurBuild"`        // True when paru -Sua / PKGBUILD review needed
	Warnings                []string          `json:"warnings"`
	ProviderErrors          map[string]string `json:"providerErrors,omitempty"`
	RequiresExplicitApprove bool              `json:"requiresExplicitApproval"`
	CreatedAt               time.Time         `json:"createdAt"`
	ExpiresAt               time.Time         `json:"expiresAt"`
}

// UpdateItemOutcome records the result of an independent provider transaction during batch upgrade
type UpdateItemOutcome struct {
	OfferID         string `json:"offerId"`
	PackageName     string `json:"packageName"`
	Provider        string `json:"provider"`
	InstallScope    string `json:"installScope,omitempty"`
	Success         bool   `json:"success"`
	SkippedUpToDate bool   `json:"skippedUpToDate,omitempty"`
	ErrorCode       string `json:"errorCode,omitempty"`
	TransactionID   string `json:"transactionId,omitempty"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
}

// UpdateBatchResult reports coordinated multi-provider update outcomes cleanly.
type UpdateBatchResult struct {
	TotalRequested int                 `json:"totalRequested"`
	Succeeded      int                 `json:"succeeded"`
	Failed         int                 `json:"failed"`
	Results        []UpdateItemOutcome `json:"results"`
	Note           string              `json:"note"`
}
