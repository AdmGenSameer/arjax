package icons

import (
	"context"
)

// IconProvider defines the unified interface that every format-specific icon resolver implements
type IconProvider interface {
	// Name returns the provider identifier (e.g. "appstream", "flatpak", "snap", "desktop_file", "system_theme", "aur", "fallback")
	Name() string

	// Priority defines resolution order according to the resolution mode (1 is highest priority)
	Priority(mode ResolutionMode) int

	// Supports returns true if this provider is applicable for the given mode and package manager
	Supports(mode ResolutionMode, packageManager string) bool

	// ResolveIcon attempts to locate and retrieve the icon asset
	ResolveIcon(ctx context.Context, appID, pkgName string, mode ResolutionMode) (*IconAsset, error)
}
