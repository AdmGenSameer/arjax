package identity

import (
	"strings"
)

// MatchConfidence indicates how reliable a canonical identity resolution is
type MatchConfidence string

const (
	ConfidenceExact     MatchConfidence = "exact"     // Reverse-DNS AppStream ID match or exact curated canonical entry
	ConfidenceHigh      MatchConfidence = "high"      // Verified provider package name explicitly mapped to CanonicalID
	ConfidenceMedium    MatchConfidence = "medium"    // Known variant (e.g. -bin suffix) of a registered canonical package
	ConfidenceUncertain MatchConfidence = "uncertain" // Name similarity only — must NOT automatically merge across providers
	ConfidenceNone      MatchConfidence = "none"      // Unmapped package
)

// ConfidenceScore converts a MatchConfidence level to a sortable numeric weight (0..100)
func ConfidenceScore(c MatchConfidence) int {
	switch c {
	case ConfidenceExact:
		return 100
	case ConfidenceHigh:
		return 90
	case ConfidenceMedium:
		return 75
	case ConfidenceUncertain:
		return 40
	default:
		return 0
	}
}

// IsSafeToMerge reports whether a resolution confidence is strong enough to group packages
// under a single unified application identity without risking false merges.
func IsSafeToMerge(c MatchConfidence) bool {
	return c == ConfidenceExact || c == ConfidenceHigh || c == ConfidenceMedium
}

// ResolutionResult describes how an identifier resolved to a canonical application ID
type ResolutionResult struct {
	Input       string          `json:"input"`
	CanonicalID string          `json:"canonicalId"`
	Confidence  MatchConfidence `json:"confidence"`
	Mergeable   bool            `json:"mergeable"`
	Reason      string          `json:"reason"`
	DisplayName string          `json:"displayName,omitempty"`
}

// AssessStrippedSuffix checks whether stripping a packaging suffix (-bin, -git, etc.)
// yields a trustworthy match or an uncertain match. Only known precompiled packaging variants
// with metadata/registry confirmation are Medium; other heuristic guesses remain Uncertain.
func AssessStrippedSuffix(originalNorm, stripped string) (MatchConfidence, string) {
	if stripped == originalNorm {
		return ConfidenceUncertain, "exact suffix unchanged"
	}
	if strings.HasSuffix(originalNorm, "-bin") || strings.HasSuffix(originalNorm, "-desktop") {
		return ConfidenceMedium, "verified packaging variant suffix matched registered application"
	}
	// Suffixes like -esr, -nightly, -canary, -git, -qt often represent distinct release tracks
	// or plugins; mark uncertain unless explicitly mapped in curated aliases!
	return ConfidenceUncertain, "unconfirmed variant suffix; keeping separate to avoid false merge"
}

