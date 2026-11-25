package identity

import (
	"fmt"
	"strings"
	"sync"
)

// Resolver owns canonical AppStream identity resolution and package alias matching.
// Icon resolvers consume resolved identities without owning package identity rules.
type Resolver struct {
	mu       sync.RWMutex
	registry *Registry
}

// NewResolver constructs an identity Resolver backed by the provided Registry (or the default singleton).
func NewResolver(reg *Registry) *Resolver {
	if reg == nil {
		reg = GetDefaultRegistry()
	}
	return &Resolver{
		registry: reg,
	}
}

// Registry returns the underlying identity registry.
func (res *Resolver) Registry() *Registry {
	return res.registry
}

// RegisterRuntimeAppStream adds or updates a canonical identity discovered dynamically from AppStream/Flathub metadata.
func (res *Resolver) RegisterRuntimeAppStream(canonicalID, displayName string, pkgAliases ...string) {
	if strings.TrimSpace(canonicalID) == "" {
		return
	}
	res.mu.Lock()
	defer res.mu.Unlock()

	if existing, ok := res.registry.GetByCanonicalID(canonicalID); ok {
		// Merge new package aliases if absent
		for _, a := range pkgAliases {
			n := strings.ToLower(strings.TrimSpace(a))
			if n == "" {
				continue
			}
			if _, exists := res.registry.pkgAliasToCanon[n]; !exists {
				existing.PackageAliases = append(existing.PackageAliases, n)
				res.registry.pkgAliasToCanon[n] = existing.CanonicalID
			}
		}
		return
	}

	res.registry.Register(AppIdentityEntry{
		CanonicalID:    canonicalID,
		Name:           displayName,
		PackageAliases: pkgAliases,
	})
}

// Resolve resolves an input identifier (AppStream ID or package name) and returns full confidence metadata.
func (res *Resolver) Resolve(input string, providerHint string) ResolutionResult {
	res.mu.RLock()
	defer res.mu.RUnlock()

	trimmed := strings.TrimSpace(input)
	norm := strings.ToLower(trimmed)
	if norm == "" {
		return ResolutionResult{
			Input:      input,
			Confidence: ConfidenceNone,
			Mergeable:  false,
			Reason:     "empty input",
		}
	}

	// 1. Exact reverse-DNS / AppStream ID match
	if canon, ok := res.registry.appstreamToCanon[norm]; ok {
		entry, _ := res.registry.GetByCanonicalID(canon)
		name := ""
		if entry != nil {
			name = entry.Name
		}
		return ResolutionResult{
			Input:       input,
			CanonicalID: canon,
			Confidence:  ConfidenceExact,
			Mergeable:   true,
			Reason:      "exact AppStream component ID match",
			DisplayName: name,
		}
	}

	// 2. Confirmed package alias match
	if canon, ok := res.registry.pkgAliasToCanon[norm]; ok {
		entry, _ := res.registry.GetByCanonicalID(canon)
		name := ""
		if entry != nil {
			name = entry.Name
		}
		return ResolutionResult{
			Input:       input,
			CanonicalID: canon,
			Confidence:  ConfidenceHigh,
			Mergeable:   true,
			Reason:      "verified package alias mapped to canonical identity",
			DisplayName: name,
		}
	}

	// 3. Conservative packaging-suffix stripping (-bin, -desktop).
	// Variant tracks such as -esr, -nightly, -canary, -git remain unmerged unless explicitly curated.
	for _, sfx := range []string{"-bin", "-desktop"} {
		if strings.HasSuffix(norm, sfx) {
			stripped := strings.TrimSuffix(norm, sfx)
			if canon, ok := res.registry.pkgAliasToCanon[stripped]; ok {
				conf, reason := AssessStrippedSuffix(norm, stripped)
				entry, _ := res.registry.GetByCanonicalID(canon)
				name := ""
				if entry != nil {
					name = entry.Name
				}
				return ResolutionResult{
					Input:       input,
					CanonicalID: canon,
					Confidence:  conf,
					Mergeable:   IsSafeToMerge(conf),
					Reason:      reason,
					DisplayName: name,
				}
			}
		}
	}

	// 4. Reverse-DNS syntax not in curated list (likely Flatpak/AppStream ID)
	if isReverseDNS(norm) {
		return ResolutionResult{
			Input:       input,
			CanonicalID: trimmed,
			Confidence:  ConfidenceExact,
			Mergeable:   true,
			Reason:      "reverse-DNS component ID",
		}
	}

	// 5. Unmapped package name: assign non-colliding canonical ID without forced merging
	unmappedID := norm
	if prov := strings.ToLower(strings.TrimSpace(providerHint)); prov != "" {
		// Still keep clean package name for deterministic fallback display,
		// while marking Mergeable=false so dissimilar providers do not auto-merge ambiguously.
		_ = prov
	}
	return ResolutionResult{
		Input:       input,
		CanonicalID: unmappedID,
		Confidence:  ConfidenceNone,
		Mergeable:   false,
		Reason:      fmt.Sprintf("unverified package %q; kept separate until metadata confirms identity", trimmed),
	}
}

// ResolveCanonical returns the canonical ID if resolved with sufficient confidence,
// otherwise returning a clean unmerged identifier.
func (res *Resolver) ResolveCanonical(input string) string {
	result := res.Resolve(input, "")
	return result.CanonicalID
}

// GetPackageAliases returns known distribution package aliases for a canonical ID or alias.
func (res *Resolver) GetPackageAliases(appID string) []string {
	res.mu.RLock()
	defer res.mu.RUnlock()

	resolved := res.Resolve(appID, "")
	if entry, ok := res.registry.GetByCanonicalID(resolved.CanonicalID); ok {
		return append([]string(nil), entry.PackageAliases...)
	}

	norm := strings.ToLower(strings.TrimSpace(appID))
	parts := strings.Split(norm, ".")
	if len(parts) >= 3 {
		return []string{norm, parts[len(parts)-1]}
	}
	return []string{norm}
}

// GetIconNames returns separated icon theme/asset lookup names for an application identity.
// Used by icons.IconResolverService without conflating icon names with package aliases.
func (res *Resolver) GetIconNames(appID string) []string {
	res.mu.RLock()
	defer res.mu.RUnlock()

	resolved := res.Resolve(appID, "")
	if entry, ok := res.registry.GetByCanonicalID(resolved.CanonicalID); ok {
		return append([]string(nil), entry.IconNames...)
	}
	norm := strings.ToLower(strings.TrimSpace(appID))
	parts := strings.Split(norm, ".")
	if len(parts) >= 3 {
		return []string{norm, parts[len(parts)-1]}
	}
	return []string{norm}
}

// isReverseDNS checks heuristic structure like "org.mozilla.firefox" (>= 3 dot-separated non-empty segments, no spaces)
func isReverseDNS(s string) bool {
	if strings.ContainsAny(s, " /:") {
		return false
	}
	parts := strings.Split(s, ".")
	if len(parts) < 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
	}
	return true
}

