package config

import "time"

// Config represents engine configuration settings
type Config struct {
	Provider   ProviderGlobalConfig `json:"provider"`
	Repository RepositoryConfig     `json:"repository"`
	AUR        AURConfig            `json:"aur"`
	GitHub     GitHubConfig         `json:"github"`
	Flatpak    FlatpakConfig        `json:"flatpak"`
	Snap       SnapConfig           `json:"snap"`
	Vendor     VendorConfig         `json:"vendor"`
}

type ProviderGlobalConfig struct {
	TimeoutSeconds         int `json:"timeout_seconds"`
	MaxConcurrentDownloads int `json:"max_concurrent_downloads"`
	CacheTTLHours          int `json:"cache_ttl_hours"`
}

type RepositoryConfig struct {
	Enabled     bool   `json:"enabled"`
	PreferredPM string `json:"preferred_pm"`
}

type AURConfig struct {
	Enabled  bool   `json:"enabled"`
	BuildDir string `json:"build_dir"`
	UseParu  bool   `json:"use_paru"`
}

type GitHubConfig struct {
	Enabled bool `json:"enabled"`
}

type FlatpakConfig struct {
	Enabled bool     `json:"enabled"`
	Remotes []string `json:"remotes"`
}

type SnapConfig struct {
	Enabled bool `json:"enabled"`
}

type VendorConfig struct {
	Enabled bool `json:"enabled"`
}

// DefaultConfig returns default configuration parameters
func DefaultConfig() Config {
	return Config{
		Provider: ProviderGlobalConfig{
			TimeoutSeconds:         30,
			MaxConcurrentDownloads: 3,
			CacheTTLHours:          24,
		},
		Repository: RepositoryConfig{
			Enabled:     true,
			PreferredPM: "",
		},
		AUR: AURConfig{
			Enabled:  true,
			BuildDir: "~/.cache/arjax/aur",
			UseParu:  true,
		},
		GitHub: GitHubConfig{
			Enabled: true,
		},
		Flatpak: FlatpakConfig{
			Enabled: true,
			Remotes: []string{"flathub"},
		},
		Snap: SnapConfig{
			Enabled: true,
		},
		Vendor: VendorConfig{
			Enabled: true,
		},
	}
}

// Timeout returns context timeout duration
func (c *Config) Timeout() time.Duration {
	if c.Provider.TimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.Provider.TimeoutSeconds) * time.Second
}

// CacheTTL returns cache expiration TTL
func (c *Config) CacheTTL() time.Duration {
	if c.Provider.CacheTTLHours <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(c.Provider.CacheTTLHours) * time.Hour
}
