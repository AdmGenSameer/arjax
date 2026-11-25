package catalog

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/appstream"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/config"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
)

// mockProvider simulates heterogeneous, slow, failing, or locking backends safely in tests
type mockProvider struct {
	name      string
	supported bool
	delay     time.Duration
	pkgs      []models.Package
	searchErr error
	onInstall func(ctx context.Context, pkg models.Package) (models.Result, error)
	onRemove  func(ctx context.Context, pkg models.Package) (models.Result, error)
}

func (m *mockProvider) Name() string      { return m.name }
func (m *mockProvider) IsSupported() bool { return m.supported }
func (m *mockProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.pkgs, nil
}
func (m *mockProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	if m.onInstall != nil {
		return m.onInstall(ctx, pkg)
	}
	return models.Result{Package: pkg.Name, Provider: m.name, Success: true, DurationMs: 5}, nil
}
func (m *mockProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	if m.onRemove != nil {
		return m.onRemove(ctx, pkg)
	}
	return models.Result{Package: pkg.Name, Provider: m.name, Success: true, DurationMs: 4}, nil
}
func (m *mockProvider) Update(ctx context.Context) (models.Result, error) {
	return models.Result{Package: "update", Provider: m.name, Success: true}, nil
}
func (m *mockProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	return models.PackageInfo{Name: pkgName, Provider: m.name, Version: "1.0.0"}, nil
}
func (m *mockProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	return nil, nil
}

func TestCanonicalIdentityAndAliases(t *testing.T) {
	deduper := NewDeduplicator()

	tests := []struct {
		input    string
		expected string
	}{
		{"firefox", "org.mozilla.firefox"},
		{"firefox-bin", "org.mozilla.firefox"},
		{"org.mozilla.firefox", "org.mozilla.firefox"},
		{"steam", "com.valvesoftware.Steam"},
		{"steam-native", "com.valvesoftware.Steam"},
		{"com.valvesoftware.Steam", "com.valvesoftware.Steam"},
		{"blender", "org.blender.Blender"},
		{"vlc", "org.videolan.VLC"},
		{"spotify", "com.spotify.Client"},
		{"brave-bin", "com.brave.Browser"},
		{"unknown-cli-tool", "unknown-cli-tool"},
	}

	for _, tt := range tests {
		actual := deduper.CanonicalizeID(tt.input)
		if !strings.EqualFold(actual, tt.expected) {
			t.Errorf("CanonicalizeID(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}

// Verifies uncertain/unmapped packages from different providers stay separate instead of false-merging
func TestNoFalseMergingOfUncertainMatches(t *testing.T) {
	deduper := NewDeduplicator()
	pkgs := []models.Package{
		{Name: "mycustomtool", Version: "1.0", Provider: "pacman", Repo: "extra"},
		{Name: "mycustomtool", Version: "2.0", Provider: "aur", Repo: "aur"},
	}

	apps := deduper.GroupAndDeduplicate(pkgs, nil, nil)
	if len(apps) != 2 {
		t.Fatalf("Expected 2 separate apps for unconfirmed tool names across pacman & aur, got %d", len(apps))
	}
}

func TestMultiSourceGroupingAndOfferCollisionSafety(t *testing.T) {
	deduper := NewDeduplicator()

	pkgs := []models.Package{
		{Name: "firefox", Version: "124.0.1-1", Provider: "pacman", Repo: "extra", Arch: "x86_64"},
		{Name: "firefox-bin", Version: "124.0.1", Provider: "aur", Repo: "aur", Arch: "x86_64"},
		{Name: "firefox", Version: "124.0.1-1", Provider: "pacman", Repo: "extra", Arch: "x86_64"}, // Duplicate mirror
		{Name: "org.mozilla.firefox", Version: "125.0b1", Provider: "flatpak", Repo: "flathub", Branch: "beta"},
	}

	crawlerMap := map[string]*appstream.AppstreamMetadata{
		"org.mozilla.firefox": {
			ID:        "org.mozilla.firefox",
			FlathubID: "org.mozilla.firefox",
			Name:      "Firefox Web Browser",
			Summary:   "Fast, Private & Free Web Browser",
		},
	}

	// Only native pacman package is installed
	installedMap := map[string]string{
		"pacman:firefox": "124.0.1-1",
	}

	unified := deduper.GroupAndDeduplicate(pkgs, crawlerMap, installedMap)
	if len(unified) != 1 {
		t.Fatalf("Expected 1 unified app for Firefox, got %d", len(unified))
	}

	app := unified[0]
	// Offers expected:
	// 1. flatpak:flathub:org.mozilla.firefox (from crawler stable)
	// 2. pacman:extra:firefox:arch=x86_64
	// 3. aur:aur:firefox-bin:arch=x86_64
	// 4. flatpak:flathub:org.mozilla.firefox:branch=beta (distinct branch must NOT collide!)
	if len(app.Offers) != 4 {
		t.Fatalf("Expected 4 distinct offers (including separate Flatpak beta branch), got %d", len(app.Offers))
	}

	// Ensure Pacman is trusted official, AUR is community, and Flatpak beta didn't mark installed
	for _, o := range app.Offers {
		if o.Provider == "pacman" {
			if o.Trust.TrustLevel != TrustOfficial || !o.Trust.OfficialRepo {
				t.Errorf("Expected pacman offer to have official trust, got %+v", o.Trust)
			}
			if !o.IsInstalled {
				t.Errorf("Expected pacman offer to be marked installed")
			}
		}
		if o.Provider == "aur" {
			if o.Trust.TrustLevel != TrustCommunity || o.Trust.OfficialRepo {
				t.Errorf("Expected aur offer to have community trust, got %+v", o.Trust)
			}
			if o.IsInstalled {
				t.Errorf("Expected aur offer NOT to be marked installed")
			}
		}
		if o.Provider == "flatpak" && o.IsInstalled {
			t.Errorf("Flatpak offer %s must NOT be marked installed when only pacman:firefox is installed", o.ID)
		}
	}
}

func TestVersionComparisonProviderSpecific(t *testing.T) {
	tests := []struct {
		provider string
		a, b     string
		wantSign int // 1, -1, 0
	}{
		{"pacman", "1:1.0.0-1", "2.0.0-1", 1},     // Epoch wins in ALPM
		{"pacman", "124.0.2-1", "124.0.10-1", -1}, // Numeric segment compare
		{"pacman", "124.0.1-2", "124.0.1-1", 1},   // pkgrel wins
		{"deb", "1.0~rc1", "1.0", -1},             // Debian tilde sorts earlier than final release
		{"rpm", "2.0~beta1-1", "2.0-1", -1},
		{"flatpak", "v4.2.10", "4.2.2", 1},
		{"snap", "latest", "1.0.0", -1},
	}

	for _, tc := range tests {
		got := CompareVersions(tc.provider, tc.a, tc.b)
		if (tc.wantSign > 0 && got <= 0) || (tc.wantSign < 0 && got >= 0) || (tc.wantSign == 0 && got != 0) {
			t.Errorf("CompareVersions(%q, %q, %q) = %d, want sign %d", tc.provider, tc.a, tc.b, got, tc.wantSign)
		}
	}
}

func TestSmartRecommendationPolicies(t *testing.T) {
	fullCaps := map[string]providers.ProviderCapability{
		"pacman":  {Provider: "pacman", Search: providers.SupportFull, Install: providers.SupportFull, AvailableOnHost: true, OfficialDistro: true},
		"flatpak": {Provider: "flatpak", Search: providers.SupportFull, Install: providers.SupportFull, AvailableOnHost: true},
		"aur":     {Provider: "aur", Search: providers.SupportFull, Install: providers.SupportFull, AvailableOnHost: true},
	}

	makeOffers := func() UnifiedApplication {
		return UnifiedApplication{
			AppID: "org.mozilla.firefox",
			Name:  "Firefox",
			Offers: []PackageOffer{
				{ID: "aur:aur:firefox-git", PackageName: "firefox-git", Provider: "aur", Repository: "aur", Version: "126.0.0-1", DependencyCount: 2},
				{ID: "flatpak:flathub:org.mozilla.firefox", PackageName: "org.mozilla.firefox", Provider: "flatpak", Repository: "flathub", Version: "124.0.2", DependencyCount: 3},
				{ID: "pacman:extra:firefox", PackageName: "firefox", Provider: "pacman", Repository: "extra", Version: "125.0.1-1", DependencyCount: 0},
			},
		}
	}

	// 1. PolicyOfficialOnly disables AUR
	officialEng := NewRecommendationEngineWithCapabilities(CatalogConfig{Policy: PolicyOfficialOnly}, fullCaps)
	appOff := makeOffers()
	officialEng.RankAndAnnotate(&appOff, "")
	for _, o := range appOff.Offers {
		if o.Provider == "aur" && o.IsEligible {
			t.Errorf("PolicyOfficialOnly must mark community AUR offer ineligible")
		}
	}

	// 2. PolicyLatestVersion ranks 126.0.0-1 highest using provider version comparator
	latestEng := NewRecommendationEngineWithCapabilities(CatalogConfig{Policy: PolicyLatestVersion}, fullCaps)
	appLat := makeOffers()
	latestEng.RankAndAnnotate(&appLat, "")
	if appLat.Offers[0].ID != "aur:aur:firefox-git" {
		t.Errorf("PolicyLatestVersion expected newest version offer (aur 126.0.0-1) first, got %s", appLat.Offers[0].ID)
	}

	// 3. PolicyMinimalDeps ranks DependencyCount=0 highest
	minEng := NewRecommendationEngineWithCapabilities(CatalogConfig{Policy: PolicyMinimalDeps}, fullCaps)
	appMin := makeOffers()
	minEng.RankAndAnnotate(&appMin, "")
	if appMin.Offers[0].ID != "pacman:extra:firefox" {
		t.Errorf("PolicyMinimalDeps expected lowest dependency offer first, got %s", appMin.Offers[0].ID)
	}
}

func TestProgressiveSearchExpressiveEventsAndOutOfOrderMocks(t *testing.T) {
	cfg := config.DefaultConfig()
	fastProv := &mockProvider{
		name:      "pacman",
		supported: true,
		delay:     5 * time.Millisecond,
		pkgs:      []models.Package{{Name: "vlc", Version: "3.0.20-1", Provider: "pacman", Repo: "extra"}},
	}
	slowProv := &mockProvider{
		name:      "flatpak",
		supported: true,
		delay:     40 * time.Millisecond,
		pkgs:      []models.Package{{Name: "org.videolan.VLC", Version: "3.0.21", Provider: "flatpak", Repo: "flathub"}},
	}
	errProv := &mockProvider{
		name:      "snap",
		supported: true,
		delay:     15 * time.Millisecond,
		searchErr: fmt.Errorf("snapd socket unavailable"),
	}

	pm := providers.NewProviderManager(cfg, fastProv, slowProv, errProv)
	cm := NewCatalogManager(pm, nil, nil, DefaultCatalogConfig(), nil)

	var (
		mu            sync.Mutex
		startedCalled bool
		doneCalled    bool
		errEvents     int
		batches       []ProgressiveSearchBatchDTO
		sequences     []int64
	)

	cm.ProgressiveSearch().StreamSearchWithCallbacks(context.Background(), "s1", "vlc", ProgressiveSearchCallbacks{
		OnStarted: func(ev SearchStartedDTO) {
			mu.Lock()
			startedCalled = true
			sequences = append(sequences, ev.Sequence)
			mu.Unlock()
		},
		OnProviderStatus: func(ev SearchProviderStatusDTO) {
			mu.Lock()
			sequences = append(sequences, ev.Sequence)
			mu.Unlock()
		},
		OnProviderError: func(ev SearchProviderStatusDTO) {
			mu.Lock()
			errEvents++
			sequences = append(sequences, ev.Sequence)
			mu.Unlock()
		},
		OnBatch: func(batch ProgressiveSearchBatchDTO) {
			mu.Lock()
			batches = append(batches, batch)
			sequences = append(sequences, batch.Sequence)
			mu.Unlock()
		},
		OnDone: func(done ProgressiveSearchDoneDTO) {
			mu.Lock()
			doneCalled = true
			sequences = append(sequences, done.Sequence)
			mu.Unlock()
		},
	})

	if !startedCalled || !doneCalled {
		t.Fatalf("Expected OnStarted and OnDone to be invoked (started=%v done=%v)", startedCalled, doneCalled)
	}
	if errEvents != 1 {
		t.Errorf("Expected 1 provider error event for failing snap mock, got %d", errEvents)
	}
	if len(batches) < 2 {
		t.Fatalf("Expected at least 2 progressive batches (pacman fast then flatpak slow), got %d", len(batches))
	}
	if len(batches[0].AddedApps) != 1 || len(batches[1].UpdatedApps) != 1 {
		t.Errorf("Expected batch 0 AddedApps=1 and batch 1 UpdatedApps=1 (incremental delta), got b0.added=%d b1.updated=%d",
			len(batches[0].AddedApps), len(batches[1].UpdatedApps))
	}
	for i := 1; i < len(sequences); i++ {
		if sequences[i] <= sequences[i-1] {
			t.Errorf("Expected strictly increasing sequence numbers, got %v", sequences)
			break
		}
	}
}

func TestProgressiveSearchCancellation(t *testing.T) {
	deduper := NewDeduplicator()
	recommender := NewRecommendationEngine(DefaultCatalogConfig())
	searchEngine := NewProgressiveSearchEngine(nil, nil, deduper, recommender, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cancelledCalled := false
	doneCalled := false
	searchEngine.StreamSearchWithCallbacks(ctx, "test-cancel-id", "firefox", ProgressiveSearchCallbacks{
		OnBatch: func(batch ProgressiveSearchBatchDTO) {
			t.Errorf("Unexpected batch emitted after cancellation")
		},
		OnCancelled: func(done ProgressiveSearchDoneDTO) {
			cancelledCalled = true
		},
		OnDone: func(done ProgressiveSearchDoneDTO) {
			doneCalled = true
			if !done.Cancelled {
				t.Errorf("Expected done.Cancelled=true")
			}
		},
	})

	if !cancelledCalled || !doneCalled {
		t.Errorf("Expected OnCancelled and OnDone to fire (cancelled=%v done=%v)", cancelledCalled, doneCalled)
	}
}

func TestTransactionLifecycleLockSafetyAndDualInstall(t *testing.T) {
	cfg := config.DefaultConfig()

	var pacCount, flatCount atomic.Int32
	commitEntered := make(chan struct{})
	allowFinish := make(chan struct{})

	pacMock := &mockProvider{
		name:      "pacman",
		supported: true,
		onInstall: func(ctx context.Context, pkg models.Package) (models.Result, error) {
			pacCount.Add(1)
			close(commitEntered)
			<-allowFinish
			return models.Result{Package: pkg.Name, Provider: "pacman", Success: true}, nil
		},
	}
	flatMock := &mockProvider{
		name:      "flatpak",
		supported: true,
		onInstall: func(ctx context.Context, pkg models.Package) (models.Result, error) {
			flatCount.Add(1)
			return models.Result{Package: pkg.Name, Provider: "flatpak", Success: true}, nil
		},
	}

	pm := providers.NewProviderManager(cfg, pacMock, flatMock)
	cm := NewCatalogManager(pm, nil, nil, DefaultCatalogConfig(), nil)

	// 1. Unknown OfferID must be rejected (no arbitrary execution authority)
	if _, err := cm.InstallOffer(context.Background(), "forged:offer:id", nil); err == nil {
		t.Errorf("Expected unknown offerID to be rejected")
	}

	// Register legitimate offers
	pacOffer := BuildOfferIdentity("pacman", "extra", "firefox", "x86_64", "", "")
	flatOffer := BuildOfferIdentity("flatpak", "flathub", "org.mozilla.firefox", "x86_64", "stable", "")

	cm.Transactions().RegisterOffers([]PackageOffer{
		{ID: pacOffer, AppID: "org.mozilla.firefox", PackageName: "firefox", Provider: "pacman", Repository: "extra", Version: "124.0.1-1", InstallScope: ScopeSystem, IsEligible: true},
		{ID: flatOffer, AppID: "org.mozilla.firefox", PackageName: "org.mozilla.firefox", Provider: "flatpak", Repository: "flathub", Version: "124.0.1", InstallScope: ScopeSandboxed, IsEligible: true},
	})

	// 2. Simulate active pacman db.lck
	cm.Transactions().SetLockChecker(func(prov string) (bool, string) {
		if prov == "pacman" {
			return true, "/var/lib/pacman/db.lck"
		}
		return false, ""
	})
	lockTx, err := cm.InstallOffer(context.Background(), pacOffer, nil)
	if err == nil || lockTx.ErrorCode != "lock_held" {
		t.Errorf("Expected lock_held error when pacman db.lck exists, got tx=%+v err=%v", lockTx, err)
	}

	// Clear lock
	cm.Transactions().SetLockChecker(func(prov string) (bool, string) { return false, "" })

	// 3. Verify unsafe mid-commit cancellation refuses to abort DB writes, and concurrent dual install works safely
	var states []TransactionState
	var txIDMu sync.Mutex
	var activeID string

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tx, err := cm.InstallOffer(context.Background(), pacOffer, func(u TransactionDTO) {
			txIDMu.Lock()
			activeID = u.ID
			states = append(states, u.State)
			txIDMu.Unlock()
		})
		if err != nil || tx.State != TxCompleted {
			t.Errorf("Expected pacman tx to complete safely even after deferred cancel, got state=%s err=%v", tx.State, err)
		}
	}()

	<-commitEntered
	txIDMu.Lock()
	cid := activeID
	txIDMu.Unlock()
	if err := cm.Transactions().CancelTransaction(cid); err == nil {
		t.Errorf("Expected CancelTransaction to refuse aborting mid-commit during TxInstalling")
	}
	close(allowFinish)
	wg.Wait()

	// 4. Simultaneous Flatpak offer installation on the same app leaves both installed independently
	flatTx, err := cm.InstallOffer(context.Background(), flatOffer, nil)
	if err != nil || flatTx.State != TxCompleted {
		t.Fatalf("Expected flatpak dual-install tx to succeed, got %+v err=%v", flatTx, err)
	}

	storedPac, _ := cm.Transactions().LookupOffer(pacOffer)
	storedFlat, _ := cm.Transactions().LookupOffer(flatOffer)
	if !storedPac.IsInstalled || !storedFlat.IsInstalled {
		t.Errorf("Both pacman and flatpak offers should be independently installed (pac=%v flat=%v)", storedPac.IsInstalled, storedFlat.IsInstalled)
	}
	if pacCount.Load() != 1 || flatCount.Load() != 1 {
		t.Errorf("Expected 1 pacman commit and 1 flatpak commit, got pac=%d flat=%d", pacCount.Load(), flatCount.Load())
	}
	if len(states) < 4 {
		t.Errorf("Expected multi-stage lifecycle transitions, got %v", states)
	}
}

func TestProviderCapabilityMatrix(t *testing.T) {
	matrix := providers.DefaultCapabilityMatrix()
	ordered := providers.GetCapabilitiesOrdered(matrix)
	if len(ordered) < 6 {
		t.Fatalf("Expected at least 6 tracked ecosystems in capability matrix, got %d", len(ordered))
	}
	if appimg, ok := matrix["appimage"]; ok {
		if okInstall, _ := appimg.CanPerform("install"); okInstall {
			t.Errorf("Planned AppImage provider must not claim install capability is ready")
		}
	} else {
		t.Errorf("Expected appimage entry in capability matrix")
	}
}

func TestDebianKaliProviderNotRelabeledAsPacman(t *testing.T) {
	d := NewDeduplicator()
	pkgs := []models.Package{
		{
			Name:        "nmap",
			Version:     "7.95+dfsg-2kali1",
			Arch:        "amd64",
			Description: "The Network Mapper",
			Provider:    "repository",
			Source:      "apt",
			Repo:        "kali-rolling",
		},
	}

	apps := d.GroupAndDeduplicate(pkgs, nil, map[string]string{})
	if len(apps) != 1 {
		t.Fatalf("Expected 1 unified app for apt package, got %d", len(apps))
	}
	app := apps[0]
	if len(app.Offers) != 1 {
		t.Fatalf("Expected 1 offer, got %d", len(app.Offers))
	}
	offer := app.Offers[0]
	if offer.Provider != "apt" {
		t.Fatalf("Expected offer provider 'apt' on Kali/Debian, got %q", offer.Provider)
	}
	if offer.CommandPreview != "sudo apt install nmap" {
		t.Errorf("Expected apt install command preview, got %q", offer.CommandPreview)
	}
	if offer.Trust.TrustLevel != TrustOfficial {
		t.Errorf("Expected kali-rolling apt repo to be TrustOfficial, got %+v", offer.Trust)
	}
	if !strings.Contains(app.Homepage, "packages.debian.org") {
		t.Errorf("Expected Debian/APT homepage URL rather than archlinux.org, got %q", app.Homepage)
	}
}

func FuzzCompareVersions(f *testing.F) {
	f.Add("pacman", "1:2.0.1-1", "1:2.0.1-2")
	f.Add("deb", "1.0~rc1-1", "1.0-1")
	f.Add("flatpak", "v3.1.2", "3.1.10")
	f.Add("rpm", "4.0^git1", "4.0")

	f.Fuzz(func(t *testing.T, prov, a, b string) {
		ab := CompareVersions(prov, a, b)
		ba := CompareVersions(prov, b, a)
		if ab > 0 && ba >= 0 {
			t.Errorf("Antisymmetry violated for (%q, %q, %q): ab=%d ba=%d", prov, a, b, ab, ba)
		}
		if ab == 0 && ba != 0 {
			t.Errorf("Equality symmetry violated for (%q, %q, %q): ab=%d ba=%d", prov, a, b, ab, ba)
		}
	})
}
