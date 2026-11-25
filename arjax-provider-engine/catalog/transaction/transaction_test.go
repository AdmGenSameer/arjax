package transaction

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/config"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
)

type noopProvider struct {
	name string
}

func (n *noopProvider) Name() string      { return n.name }
func (n *noopProvider) IsSupported() bool { return true }
func (n *noopProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	return nil, nil
}
func (n *noopProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	return models.Result{Package: pkg.Name, Provider: n.name, Success: true}, nil
}
func (n *noopProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	return models.Result{Package: pkg.Name, Provider: n.name, Success: true}, nil
}
func (n *noopProvider) Update(ctx context.Context) (models.Result, error) {
	return models.Result{Package: "system-update", Provider: n.name, Success: true}, nil
}
func (n *noopProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	return models.PackageInfo{Name: pkgName, Provider: n.name}, nil
}
func (n *noopProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	return nil, nil
}

func TestJournalRecoveryAfterSimulatedCrash(t *testing.T) {
	tmpDir := t.TempDir()
	journalPath := filepath.Join(tmpDir, "transactions.json")

	j := NewJournal(journalPath)
	j.Append(&TransactionDTO{
		ID:          "tx_crash_completed",
		OfferID:     "pacman:extra:firefox",
		PackageName: "firefox",
		Provider:    "pacman",
		Operation:   OpInstall,
		State:       TxInstalling,
		Progress:    82,
		StartedAt:   time.Now().Add(-3 * time.Minute),
	})
	j.Append(&TransactionDTO{
		ID:           "tx_crash_failed",
		OfferID:      "flatpak:flathub:org.blender.Blender:scope=user:branch=stable",
		PackageName:  "org.blender.Blender",
		Provider:     "flatpak",
		InstallScope: "user",
		Operation:    OpInstall,
		State:        TxDownloading,
		Progress:     50,
		StartedAt:    time.Now().Add(-2 * time.Minute),
	})
	j.Append(&TransactionDTO{
		ID:              "tx_crash_update_unknown",
		OfferID:         "flatpak:flathub:org.gimp.GIMP:scope=system:branch=stable",
		PackageName:     "org.gimp.GIMP",
		Provider:        "flatpak",
		InstallScope:    "system",
		PreviousVersion: "2.10.36",
		TargetVersion:   "2.10.38",
		Operation:       OpUpdate,
		State:           TxInstalling,
		Progress:        82,
		StartedAt:       time.Now().Add(-1 * time.Minute),
	})
	j.Append(&TransactionDTO{
		ID:          "tx_crash_locked",
		OfferID:     "pacman:extra:vim",
		PackageName: "vim",
		Provider:    "pacman",
		Operation:   OpInstall,
		State:       TxInstalling,
		Progress:    82,
		StartedAt:   time.Now().Add(-30 * time.Second),
	})

	pm := providers.NewProviderManager(config.DefaultConfig())
	mgr := NewManager(pm, journalPath)
	mgr.SetLockChecker(func(prov string) (bool, string) {
		// Simulate stale pacman db.lck remaining for tx_crash_locked check only
		return false, ""
	})

	// Track which flatpak scopes were probed cleanly
	var probedScopes []string
	reconciled := mgr.RecoverOnStartup(context.Background(), func(ctx context.Context, prov, pkg, scope string) (bool, string, error) {
		if prov == "flatpak" {
			probedScopes = append(probedScopes, pkg+":"+scope)
		}
		if prov == "pacman" && pkg == "firefox" {
			return true, "125.0.1-1", nil
		}
		if prov == "flatpak" && pkg == "org.gimp.GIMP" {
			// Package exists on host, but version doesn't match target or previous -> MUST be outcome_unknown!
			return true, "2.10.37", nil
		}
		return false, "", nil
	})

	if len(reconciled) != 4 {
		t.Fatalf("Expected 4 recovered transactions, got %d", len(reconciled))
	}

	txOK, _ := mgr.GetTransaction("tx_crash_completed")
	if txOK.State != TxCompleted || txOK.RecoveryStatus != RecoveryReconciledComplete {
		t.Errorf("Expected tx_crash_completed to reconcile COMPLETED, got %+v", txOK)
	}

	txFail, _ := mgr.GetTransaction("tx_crash_failed")
	if txFail.State != TxFailed || txFail.RecoveryStatus != RecoveryReconciledFailed {
		t.Errorf("Expected tx_crash_failed to reconcile FAILED, got %+v", txFail)
	}

	txUnk, _ := mgr.GetTransaction("tx_crash_update_unknown")
	if txUnk.RecoveryStatus != RecoveryOutcomeUnknown || txUnk.State != TxFailed {
		t.Errorf("Expected ambiguous interrupted update to report outcome_unknown, got %+v", txUnk)
	}

	// Verify active lock triggers recovery_required
	mgrLocked := NewManager(pm, filepath.Join(t.TempDir(), "lock.json"))
	mgrLocked.journal.Append(&TransactionDTO{ID: "tx_lock", PackageName: "vlc", Provider: "pacman", Operation: OpInstall, State: TxInstalling})
	mgrLocked.SetLockChecker(func(prov string) (bool, string) { return true, "/var/lib/pacman/db.lck" })
	recLock := mgrLocked.RecoverOnStartup(context.Background(), func(ctx context.Context, prov, pkg, scope string) (bool, string, error) {
		return false, "", nil
	})
	if len(recLock) != 1 || recLock[0].RecoveryStatus != RecoveryRequired {
		t.Errorf("Expected active lock after crash to trigger recovery_required, got %+v", recLock)
	}

	foundUserScope := false
	for _, s := range probedScopes {
		if s == "org.blender.Blender:user" {
			foundUserScope = true
		}
	}
	if !foundUserScope {
		t.Errorf("Expected user Flatpak scope passed to recovery probe, got %v", probedScopes)
	}
}

func TestOfferExpirationAndMetadataEpochValidation(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(nil, filepath.Join(tmpDir, "tx.json"))
	mgr.SetOfferTTL(2 * time.Millisecond)

	mgr.RegisterOfferDescriptor(OfferDescriptor{
		OfferID:     "pacman:extra:vim",
		AppID:       "vim",
		PackageName: "vim",
		Provider:    "pacman",
		IsEligible:  true,
	})

	time.Sleep(10 * time.Millisecond)

	tx, err := mgr.ExecuteOfferTransaction(context.Background(), "pacman:extra:vim", OpInstall, nil)
	if err == nil || tx.ErrorCode != "offer_expired" {
		t.Errorf("Expected offer_expired error after TTL elapses, got tx=%+v err=%v", tx, err)
	}

	mgr.SetOfferTTL(10 * time.Minute)
	mgr.RegisterOfferDescriptor(OfferDescriptor{
		OfferID:     "pacman:extra:nano",
		AppID:       "nano",
		PackageName: "nano",
		Provider:    "pacman",
		IsEligible:  true,
	})
	mgr.AdvanceMetadataEpoch()

	txStale, err := mgr.ExecuteOfferTransaction(context.Background(), "pacman:extra:nano", OpInstall, nil)
	if err == nil || txStale.ErrorCode != "offer_stale" || !strings.Contains(txStale.ErrorMessage, "superseded") {
		t.Errorf("Expected offer_stale rejection after epoch advance, got tx=%+v err=%v", txStale, err)
	}
}

// TestHelperProcessLockHold acts as a separate OS process (simulating TUI running concurrently with GUI)
func TestHelperProcessLockHold(t *testing.T) {
	if os.Getenv("ARJAX_TEST_LOCK_HELPER") != "1" {
		return
	}
	release, err := AcquireProcessCoordLock("pacman")
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper failed to acquire lock: %v\n", err)
		os.Exit(2)
	}
	fmt.Println("LOCKED")
	// Wait until parent closes stdin
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	release()
}

// TestRealCrossProcessLockCoordination spawns an actual OS subprocess holding the advisory lock,
// verifies a second process receives cross_process_lock busy error, then proceeds once released.
func TestRealCrossProcessLockCoordination(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcessLockHold$")
	cmd.Env = append(os.Environ(), "ARJAX_TEST_LOCK_HELPER=1", "XDG_RUNTIME_DIR="+runtimeDir)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start lock helper subprocess: %v", err)
	}

	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || !strings.Contains(line, "LOCKED") {
		_ = cmd.Process.Kill()
		t.Fatalf("helper subprocess did not signal LOCKED: line=%q err=%v", line, err)
	}

	// Second independent manager (simulating GUI while TUI holds lock) must receive busy rejection
	pm := providers.NewProviderManager(config.DefaultConfig(), &noopProvider{name: "pacman"})
	mgr := NewManager(pm, filepath.Join(t.TempDir(), "gui.json"))
	mgr.SetLockChecker(func(prov string) (bool, string) { return false, "" })
	mgr.RegisterOfferDescriptor(OfferDescriptor{
		OfferID:     "pacman:extra:htop",
		AppID:       "htop",
		PackageName: "htop",
		Provider:    "pacman",
		Scope:       "system",
		IsEligible:  true,
	})

	busyTx, err := mgr.ExecuteOfferTransaction(context.Background(), "pacman:extra:htop", OpInstall, nil)
	if err == nil || busyTx.ErrorCode != "cross_process_lock" {
		_ = stdin.Close()
		_ = cmd.Wait()
		t.Fatalf("Expected cross_process_lock busy rejection while subprocess holds lock, got tx=%+v err=%v", busyTx, err)
	}

	// Signal child process to release lock and exit cleanly
	_ = stdin.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("helper subprocess exited with error: %v", err)
	}

	// Now second process must proceed cleanly
	okTx, err := mgr.ExecuteOfferTransaction(context.Background(), "pacman:extra:htop", OpInstall, nil)
	if err != nil || okTx.State != TxCompleted {
		t.Fatalf("Expected transaction to succeed after subprocess released lock, got tx=%+v err=%v", okTx, err)
	}
}
