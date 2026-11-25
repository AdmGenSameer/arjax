package identity

import "strings"

// AppIdentityEntry defines a canonical desktop application, its package-manager aliases,
// and separately its desktop icon lookup names (decoupled from package identity).
type AppIdentityEntry struct {
	CanonicalID    string            `json:"canonicalId"`
	Name           string            `json:"name"`
	Category       string            `json:"category"`
	AppStreamIDs   []string          `json:"appstreamIds"`             // Primary & alternative reverse-DNS IDs
	PackageAliases []string          `json:"packageAliases"`           // Confirmed package names across distros
	ProviderHints  map[string]string `json:"providerHints,omitempty"`  // Optional provider -> package name map
	IconNames      []string          `json:"iconNames"`                // Icon theme / filename aliases (owned separately from pkg identity)
}

// Registry maintains the authoritative bidirectional mapping for canonical application identities.
type Registry struct {
	canonicalMap     map[string]*AppIdentityEntry // lowercase canonicalID -> entry
	appstreamToCanon map[string]string            // lowercase appstreamID -> CanonicalID
	pkgAliasToCanon  map[string]string            // lowercase confirmed pkg alias -> CanonicalID
}

// NewRegistry creates an empty identity Registry.
func NewRegistry() *Registry {
	return &Registry{
		canonicalMap:     make(map[string]*AppIdentityEntry),
		appstreamToCanon: make(map[string]string),
		pkgAliasToCanon:  make(map[string]string),
	}
}

// Register adds a canonical application entry with explicit AppStream IDs, package aliases, and icon names.
func (r *Registry) Register(entry AppIdentityEntry) {
	cleanID := strings.TrimSpace(entry.CanonicalID)
	if cleanID == "" {
		return
	}
	normID := strings.ToLower(cleanID)

	asIDs := make([]string, 0, len(entry.AppStreamIDs)+1)
	seenAS := make(map[string]bool)
	addAS := func(s string) {
		n := strings.ToLower(strings.TrimSpace(s))
		if n != "" && !seenAS[n] {
			seenAS[n] = true
			asIDs = append(asIDs, n)
			r.appstreamToCanon[n] = cleanID
		}
	}
	addAS(cleanID)
	for _, id := range entry.AppStreamIDs {
		addAS(id)
	}

	pkgList := make([]string, 0, len(entry.PackageAliases)+2)
	seenPkg := make(map[string]bool)
	addPkg := func(s string) {
		n := strings.ToLower(strings.TrimSpace(s))
		if n != "" && !seenPkg[n] {
			seenPkg[n] = true
			pkgList = append(pkgList, n)
			r.pkgAliasToCanon[n] = cleanID
		}
	}
	// Register leaf of reverse-DNS ID as confirmed package alias unless explicit aliases omit it intentionally
	parts := strings.Split(normID, ".")
	if len(parts) > 1 {
		addPkg(parts[len(parts)-1])
	}
	for _, a := range entry.PackageAliases {
		addPkg(a)
	}
	for _, pkg := range entry.ProviderHints {
		addPkg(pkg)
	}

	iconList := make([]string, 0, len(entry.IconNames)+1)
	seenIcon := make(map[string]bool)
	for _, ic := range entry.IconNames {
		n := strings.ToLower(strings.TrimSpace(ic))
		if n != "" && !seenIcon[n] {
			seenIcon[n] = true
			iconList = append(iconList, n)
		}
	}
	if len(iconList) == 0 && len(pkgList) > 0 {
		iconList = append(iconList, pkgList[0])
	}

	stored := &AppIdentityEntry{
		CanonicalID:    cleanID,
		Name:           entry.Name,
		Category:       entry.Category,
		AppStreamIDs:   asIDs,
		PackageAliases: pkgList,
		ProviderHints:  entry.ProviderHints,
		IconNames:      iconList,
	}
	r.canonicalMap[normID] = stored
}

// GetByCanonicalID retrieves an entry by its canonical ID.
func (r *Registry) GetByCanonicalID(canonicalID string) (*AppIdentityEntry, bool) {
	entry, ok := r.canonicalMap[strings.ToLower(strings.TrimSpace(canonicalID))]
	return entry, ok
}

