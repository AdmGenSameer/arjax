package models

import "time"

// Package represents a software package available from a provider
type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Size        string `json:"size,omitempty"`
	Provider    string `json:"provider"`
	Source      string `json:"source,omitempty"` // "repository", "aur", "vendor", etc.
	Repo        string `json:"repo,omitempty"`   // for repo-based packages
}

// Result represents the outcome of a provider operation
type Result struct {
	Command    string    `json:"command,omitempty"`
	Package    string    `json:"package"`
	Provider   string    `json:"provider"`
	Success    bool      `json:"success"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	DurationMs int64     `json:"duration_ms"`
	Output     string    `json:"output"`
	Error      string    `json:"error,omitempty"`
	InstallCmd string    `json:"install_cmd,omitempty"`
}

// PackageInfo contains detailed metadata about a package
type PackageInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Size         string   `json:"size,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Files        []string `json:"files,omitempty"`
	LastUpdated  string   `json:"last_updated,omitempty"`
	Repo         string   `json:"repo,omitempty"`
	Provider     string   `json:"provider,omitempty"`
}

// Recipe represents a package installation recipe
type Recipe struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Providers   map[string]ProviderConfig `json:"providers"`
}

// ProviderConfig represents configuration for a provider within a recipe
type ProviderConfig struct {
	Package  string            `json:"package"`
	Priority int               `json:"priority"`
	Options  map[string]string `json:"options,omitempty"`
}

// ErrorCode identifies specific failure types
type ErrorCode string

const (
	ErrorNotSupported  ErrorCode = "not_supported"
	ErrorNotFound      ErrorCode = "not_found"
	ErrorInstallFailed ErrorCode = "install_failed"
	ErrorNetworkError  ErrorCode = "network_error"
	ErrorTimeout       ErrorCode = "timeout"
)

// ProviderError structured error type
type ProviderError struct {
	Provider string    `json:"provider"`
	Message  string    `json:"message"`
	Code     ErrorCode `json:"code"`
}

func (e *ProviderError) Error() string {
	return e.Provider + ": " + e.Message
}
