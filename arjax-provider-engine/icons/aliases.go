package icons

import (
	"strings"
	"sync"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/identity"
)

// AppAliasEntry links a canonical reverse-DNS application ID with icon theme lookup names.
// Canonical package identity is owned by catalog/identity; icons only consumes resolved identities.
type AppAliasEntry struct {
	CanonicalID string
	Name        string
	Category    string
	Aliases     []string // Icon theme / desktop file candidate names
}

// AliasRegistry provides icon-name lookup backed by catalog/identity for canonical resolution.
type AliasRegistry struct {
	mu               sync.RWMutex
	identityResolver *identity.Resolver
	iconOverrides    map[string][]string // canonicalID (lower) -> extra icon-only aliases
}

var defaultRegistry *AliasRegistry
var onceInitRegistry sync.Once

// GetAliasRegistry returns the singleton icon alias registry backed by the canonical identity resolver.
func GetAliasRegistry() *AliasRegistry {
	onceInitRegistry.Do(func() {
		defaultRegistry = NewAliasRegistry()
	})
	return defaultRegistry
}

// NewAliasRegistry constructs an AliasRegistry delegating canonical IDs to catalog/identity.
func NewAliasRegistry() *AliasRegistry {
	return &AliasRegistry{
		identityResolver: identity.NewResolver(identity.GetDefaultRegistry()),
		iconOverrides:    make(map[string][]string),
	}
}

// Register registers an entry into the identity registry and stores optional icon-specific aliases.
func (r *AliasRegistry) Register(canonicalID, name, category string, aliases ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cleanID := strings.TrimSpace(canonicalID)
	if cleanID == "" {
		return
	}

	r.identityResolver.Registry().Register(identity.AppIdentityEntry{
		CanonicalID:    cleanID,
		Name:           name,
		Category:       category,
		PackageAliases: aliases,
		IconNames:      aliases,
	})

	normID := strings.ToLower(cleanID)
	r.iconOverrides[normID] = append([]string(nil), aliases...)
}

// ResolveCanonical consumes catalog/identity to resolve canonical reverse-DNS AppIDs.
func (r *AliasRegistry) ResolveCanonical(input string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := r.identityResolver.Resolve(input, "")
	if res.CanonicalID != "" {
		return res.CanonicalID
	}
	return input
}

// GetAliases returns icon lookup candidate names (plus resolved package aliases) for icon resolution.
func (r *AliasRegistry) GetAliases(appID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	canon := r.ResolveCanonical(appID)
	seen := make(map[string]bool)
	var out []string
	add := func(v string) {
		n := strings.ToLower(strings.TrimSpace(v))
		if n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}

	add(canon)
	for _, ic := range r.identityResolver.GetIconNames(canon) {
		add(ic)
	}
	for _, pa := range r.identityResolver.GetPackageAliases(canon) {
		add(pa)
	}
	if extra, ok := r.iconOverrides[strings.ToLower(canon)]; ok {
		for _, a := range extra {
			add(a)
		}
	}
	return out
}
