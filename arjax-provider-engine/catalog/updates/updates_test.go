package updates

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/transaction"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/config"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

type mockUpdateProvider struct {
	name      string
	updates   []AvailableUpdate
	updatesFn func(call int) ([]AvailableUpdate, error)
	err       error
	failIDs   map[string]error
	mu        sync.Mutex
	calls     int
}

func (m *mockUpdateProvider) Name() string { return m.name }
func (m *mockUpdateProvider) CheckUpdates(ctx context.Context) ([]AvailableUpdate, error) {
	m.mu.Lock()
	m.calls++
	c := m.calls
	m.mu.Unlock()
	if m.updatesFn != nil {
		return m.updatesFn(c)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.updates, nil
}
func (m *mockUpdateProvider) Update(ctx context.Context, offerID string) error {
	if err, bad := m.failIDs[offerID]; bad {
		return err
	}
	return nil
}

type recordingProvider struct {
	name          string
	mu            sync.Mutex
	fullUpdates   int
	failUpdate    bool
	scopedUpdates []models.Package
	seqLog        *[]string
}

func (r *recordingProvider) Name() string      { return r.name }
func (r *recordingProvider) IsSupported() bool { return true }
func (r *recordingProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	return nil, nil
}
func (r *recordingProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	return models.Result{Package: pkg.Name, Provider: r.name, Success: true}, nil
}
func (r *recordingProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	return models.Result{Package: pkg.Name, Provider: r.name, Success: true}, nil
}
func (r *recordingProvider) Update(ctx context.Context) (models.Result, error) {
	r.mu.Lock()
	r.fullUpdates++
	if r.seqLog != nil {
		*r.seqLog = append(*r.seqLog, r.name+":update")
	}
	shouldFail := r.failUpdate
	r.mu.Unlock()
	if shouldFail {
		return models.Result{Package: r.name + "-update", Provider: r.name, Success: false, Error: "pacman conflict"}, errors.New("pacman conflict")
	}
	return models.Result{Package: r.name + "-update", Provider: r.name, Success: true}, nil
}
func (r *recordingProvider) UpdatePackage(ctx context.Context, pkg models.Package) (models.Result, error) {
	r.mu.Lock()
	r.scopedUpdates = append(r.scopedUpdates, pkg)
	if r.seqLog != nil {
		*r.seqLog = append(*r.seqLog, r.name+":scoped:"+pkg.Name)
	}
	r.mu.Unlock()
	return models.Result{Package: pkg.Name, Provider: r.name, Success: true}, nil
}
func (r *recordingProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	return models.PackageInfo{Name: pkgName, Provider: r.name}, nil
}
func (r *recordingProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	return nil, nil
}

func TestPacmanCheckupdatesExitCodes(t *testing.T) {
	// Exit code 2 -> healthy zero updates, NO error
	pacZero := NewPacmanUpdateProviderWithRunner(nil, func(ctx context.Context, name string, args ...string) utils.ExecResult {
		return utils.ExecResult{ExitCode: 2}
	})
	upds, err := pacZero.CheckUpdates(context.Background())
	if err != nil || len(upds) != 0 {
		t.Errorf("checkupdates exit code 2 must return empty slice and nil error; got len=%d err=%v", len(upds), err)
	}

	// Exit code 1 -> unknown failure, MUST return error
	pacErr := NewPacmanUpdateProviderWithRunner(nil, func(ctx context.Context, name string, args ...string) utils.ExecResult {
		return utils.ExecResult{ExitCode: 1, Stderr: "fakeroot lock failed"}
	})
	if _, err := pacErr.CheckUpdates(context.Background()); err == nil {
		t.Errorf("checkupdates exit code 1 must return explicit error")
	}

	// Exit code 0 -> parsed updates
	pacFound := NewPacmanUpdateProviderWithRunner(nil, func(ctx context.Context, name string, args ...string) utils.ExecResult {
		return utils.ExecResult{ExitCode: 0, Stdout: "firefox 124.0-1 -> 125.0-1\n"}
	})
	updsFound, err := pacFound.CheckUpdates(context.Background())
	if err != nil || len(updsFound) != 1 || updsFound[0].PackageName != "firefox" {
		t.Errorf("checkupdates exit code 0 expected 1 parsed update; got %+v err=%v", updsFound, err)
	}
}

func TestUpdateCenterParsersAndFlatpakScopes(t *testing.T) {
	pac := ParseCheckupdatesOutput("firefox 124.0-1 -> 125.0-1\nopenssl 3.2.1-1 -> 3.3.0-1\n")
	if len(pac) != 2 || !pac[0].IsSecurity || !pac[1].IsSecurity || pac[0].PartialUpgradeSafe {
		t.Errorf("Expected 2 security-tagged pacman updates with PartialUpgradeSafe=false, got %+v", pac)
	}

	userFlat := ParseFlatpakUpdatesOutputScoped("Blender\torg.blender.Blender\t4.2.2\tstable\tflathub\n", "user")
	sysFlat := ParseFlatpakUpdatesOutputScoped("Blender\torg.blender.Blender\t4.2.2\tstable\tflathub\n", "system")
	if userFlat[0].OfferID == sysFlat[0].OfferID || userFlat[0].InstallScope != "user" || sysFlat[0].InstallScope != "system" {
		t.Errorf("Expected separate user vs system Flatpak OfferIDs and scopes")
	}
}

func TestAURBlockedWhenOfficialUpdatesPendingAndSequencedOrder(t *testing.T) {
	var seq []string
	pacRec := &recordingProvider{name: "pacman", seqLog: &seq}
	aurRec := &recordingProvider{name: "aur", seqLog: &seq}

	pm := providers.NewProviderManager(config.DefaultConfig(), pacRec, aurRec)
	txMgr := transaction.NewManager(pm, filepath.Join(t.TempDir(), "tx.json"))
	txMgr.SetLockChecker(func(prov string) (bool, string) { return false, "" })

	pacUpd := &mockUpdateProvider{
		name: "pacman",
		updates: []AvailableUpdate{
			{OfferID: "pacman:extra:firefox", PackageName: "firefox", Provider: "pacman", Repository: "extra", PartialUpgradeSafe: false},
		},
	}
	aurUpd := &mockUpdateProvider{
		name: "aur",
		updates: []AvailableUpdate{
			{OfferID: "aur:aur:visual-studio-code-bin", PackageName: "visual-studio-code-bin", Provider: "aur", Repository: "aur"},
		},
	}

	mgr := NewManager(txMgr, pacUpd, aurUpd)
	_ = mgr.CheckAllUpdates(context.Background())

	// 1. Selecting ONLY AUR while Pacman updates are pending MUST block AUR execution!
	blockedBatch := mgr.UpdateSelected(context.Background(), []string{"aur:aur:visual-studio-code-bin"}, nil)
	if blockedBatch.Failed == 0 || len(blockedBatch.Results) != 1 || blockedBatch.Results[0].ErrorCode != "official_updates_pending" {
		t.Fatalf("Expected AUR upgrade blocked with official_updates_pending when Pacman pending, got %+v", blockedBatch)
	}
	if aurRec.fullUpdates != 0 {
		t.Fatalf("AUR update must NOT execute while official Pacman updates remain pending!")
	}

	// 2. Preview when ONLY AUR selected with Pacman pending must set RequiresOfficialFirst=true
	prev := mgr.PreviewUpdates(context.Background(), []string{"aur:aur:visual-studio-code-bin"})
	if !prev.RequiresOfficialFirst {
		t.Errorf("Expected TransactionPreview.RequiresOfficialFirst=true")
	}

	// 3. Declining approval must refuse execution
	unapproved := mgr.ExecuteApprovedUpdates(context.Background(), prev.PreviewID, false, nil)
	if unapproved.Succeeded != 0 || pacRec.fullUpdates != 0 || aurRec.fullUpdates != 0 {
		t.Errorf("Unapproved ExecuteApprovedUpdates must not execute any provider updates")
	}

	// 4. Selecting BOTH Pacman & AUR executes Pacman -Syu FIRST, re-discovers AUR, then executes paru -Sua!
	aurCallsBefore := aurUpd.calls
	okBatch := mgr.UpdateAll(context.Background(), nil)
	if okBatch.Failed != 0 || okBatch.Succeeded != 2 {
		t.Fatalf("Expected sequenced Pacman+AUR batch to succeed, got %+v", okBatch)
	}
	if len(seq) != 2 || seq[0] != "pacman:update" || seq[1] != "aur:update" {
		t.Errorf("Expected strict Pacman-before-AUR execution sequence, got %v", seq)
	}
	if aurUpd.calls <= aurCallsBefore {
		t.Errorf("Expected AUR updates to be re-discovered after Pacman -Syu before running paru -Sua")
	}

	// 5. If Pacman fails, AUR MUST be blocked rather than running against broken official state
	seq = nil
	pacRec.failUpdate = true
	failBatch := mgr.UpdateAll(context.Background(), nil)
	if len(seq) != 1 || seq[0] != "pacman:update" {
		t.Errorf("Expected AUR aborted when Pacman failed, got seq=%v", seq)
	}
	if failBatch.Results[1].ErrorCode != "official_upgrade_failed" {
		t.Errorf("Expected AUR blocked with official_upgrade_failed, got %+v", failBatch.Results[1])
	}
}

func TestSnapRevalidationWhenStateChangesBetweenPreviewAndExecution(t *testing.T) {
	snapRec := &recordingProvider{name: "snap"}
	pm := providers.NewProviderManager(config.DefaultConfig(), snapRec)
	txMgr := transaction.NewManager(pm, filepath.Join(t.TempDir(), "tx.json"))
	txMgr.SetLockChecker(func(prov string) (bool, string) { return false, "" })

	// First discovery reports "spotify" pending; subsequent pre-execution revalidation returns empty (auto-refreshed!)
	snapUpd := &mockUpdateProvider{
		name: "snap",
		updatesFn: func(call int) ([]AvailableUpdate, error) {
			if call == 1 {
				return []AvailableUpdate{{OfferID: "snap:snapcraft:spotify", PackageName: "spotify", Provider: "snap", Repository: "snapcraft"}}, nil
			}
			return []AvailableUpdate{}, nil
		},
	}

	mgr := NewManager(txMgr, snapUpd)
	summary := mgr.CheckAllUpdates(context.Background())
	if summary.TotalCount != 1 {
		t.Fatalf("Expected 1 pending snap during initial discovery")
	}

	batch := mgr.UpdateSelected(context.Background(), []string{"snap:snapcraft:spotify"}, nil)
	if batch.Failed != 0 || batch.Succeeded != 1 || !batch.Results[0].SkippedUpToDate {
		t.Errorf("Expected snap no longer pending at execution time to skip cleanly via structured revalidation, got %+v", batch)
	}
	if len(snapRec.scopedUpdates) != 0 {
		t.Errorf("Expected snap provider not called redundantly once revalidation confirmed already up-to-date")
	}
}
