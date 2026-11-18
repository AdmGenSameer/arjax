package icons

import (
	"time"
)

// ResolutionMode dictates the resolution priority based on install state
type ResolutionMode string

const (
	ModeCatalog   ResolutionMode = "catalog"   // Prioritizes upstream repos, AppStream, and CDNs
	ModeInstalled ResolutionMode = "installed" // Prioritizes local .desktop files and system themes
)

// AppIdentity decouples application identity from any single package manager
type AppIdentity struct {
	AppID       string `json:"appId"`       // Canonical reverse-DNS e.g. "org.mozilla.firefox"
	Name        string `json:"name"`        // e.g. "Firefox"
	Description string `json:"description"` // e.g. "Fast, Private Web Browser"
	Category    string `json:"category"`    // e.g. "Network", "AudioVideo"
	Developer   string `json:"developer"`   // e.g. "Mozilla"
}

// PackageSource represents a distribution package that provides the application
type PackageSource struct {
	AppID          string `json:"appId"`
	PackageName    string `json:"packageName"`    // e.g. "firefox"
	PackageManager string `json:"packageManager"` // e.g. "pacman", "flatpak", "aur", "apt", "dnf", "snap", "appimage"
	Repository     string `json:"repository"`     // e.g. "extra", "flathub", "stable"
	Architecture   string `json:"architecture"`   // e.g. "x86_64"
	Version        string `json:"version"`
	Installed      bool   `json:"installed"`
}

// IconProvenance provides an immutable audit trail of how and where an icon was obtained
type IconProvenance struct {
	Source           string    `json:"source"`           // Provider identifier: "appstream", "flatpak", "snap", "desktop_file", etc.
	Repository       string    `json:"repository"`       // Origin repo: "flathub", "arch-extra", "ubuntu-main", "snapcraft", etc.
	OriginalURL      string    `json:"originalUrl"`      // Upstream download URL (if remote)
	LocalOriginPath  string    `json:"localOriginPath"`  // Local path of .desktop or theme file (if local)
	ContentHash      string    `json:"contentHash"`      // SHA-256 hash of validated asset bytes
	FetchedAt        time.Time `json:"fetchedAt"`        // Time asset was fetched
	ValidatedAt      time.Time `json:"validatedAt"`      // Time asset passed security and format validation
	ValidationStatus string    `json:"validationStatus"` // "validated", "sanitized", "fallback", "failed"
}

// IconAsset represents raw binary or vector data retrieved from a provider
type IconAsset struct {
	AppID       string         `json:"appId"`
	Source      string         `json:"source"` // e.g. "appstream", "flatpak", "system_theme", "snap", "appimage"
	Repository  string         `json:"repository,omitempty"`
	Format      string         `json:"format"` // "png", "svg", "webp", "jpeg"
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	Data        []byte         `json:"-"`
	OriginalURL string         `json:"originalUrl,omitempty"`
	LocalPath   string         `json:"localPath,omitempty"`
	ContentHash string         `json:"contentHash"`
	IsLocal     bool           `json:"isLocal"`
	Provenance  IconProvenance `json:"provenance"`
}

// IconRecord represents persistent catalog metadata for a cached icon
type IconRecord struct {
	AppID          string         `json:"appId"`
	PackageSource  string         `json:"packageSource"`
	Repository     string         `json:"repository"`
	OriginalURL    string         `json:"originalUrl"`
	OriginalFormat string         `json:"originalFormat"`
	ContentHash    string         `json:"contentHash"`
	OriginalPath   string         `json:"originalPath"`
	Variants       map[int]string `json:"variants"` // size -> absolute file path
	Width          int            `json:"width"`
	Height         int            `json:"height"`
	Status         string         `json:"status"` // "available", "cached", "fallback", "failed"
	LastChecked    time.Time      `json:"lastChecked"`
	LastAccessed   time.Time      `json:"lastAccessed"`
	Provenance     IconProvenance `json:"provenance"`
	IsNegative     bool           `json:"isNegative"` // true if lookup failed and cached negative
}

// IconResponseDTO is the clean, distribution-independent payload sent to the frontend
type IconResponseDTO struct {
	AppID       string         `json:"appId"`
	Name        string         `json:"name,omitempty"`
	IconURL     string         `json:"iconUrl"`     // Local asset route e.g. "/api/icons/org.mozilla.firefox?size=128"
	DataURI     string         `json:"dataUri"`     // base64 data URI fallback for instant rendering
	IconStatus  string         `json:"iconStatus"`  // "available", "cached", "fallback"
	Format      string         `json:"format"`      // "png", "svg", "webp"
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	ContentHash string         `json:"contentHash"`
	Source      string         `json:"source"`      // Which resolver supplied the icon
	Repository  string         `json:"repository"`  // Origin repository
	LocalPath   string         `json:"localPath,omitempty"`
	Provenance  IconProvenance `json:"provenance"`
}
