package transaction

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// InstalledStateProbe queries the real host package manager (honoring installation scope where applicable)
// to determine whether a package currently exists on the system and at which exact version.
type InstalledStateProbe func(ctx context.Context, provider, pkgName, scope string) (installed bool, version string, err error)

// DefaultHostInstalledProbe checks pacman/apt/dnf/zypper/flatpak/snap on the local Linux host.
func DefaultHostInstalledProbe(ctx context.Context, provider, pkgName, scope string) (bool, string, error) {
	prov := strings.ToLower(strings.TrimSpace(provider))
	if prov == "repository" || prov == "" {
		if detected := strings.ToLower(strings.TrimSpace(utils.DetectPackageManager())); detected != "" {
			prov = detected
		}
	}
	switch prov {
	case "pacman", "aur", "paru":
		if pkgName == "" || pkgName == "system-update" || pkgName == "aur-update" {
			return false, "", fmt.Errorf("system-wide upgrade transaction cannot be reconciled by single package name")
		}
		out, err := exec.CommandContext(ctx, "pacman", "-Q", pkgName).Output()
		if err != nil {
			return false, "", nil
		}
		fields := strings.Fields(string(out))
		if len(fields) >= 2 {
			return true, fields[1], nil
		}
		return true, "", nil

	case "apt", "deb":
		if pkgName == "" || pkgName == "system-update" {
			return false, "", fmt.Errorf("system-wide upgrade transaction cannot be reconciled by single package name")
		}
		out, err := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${Status}\t${Version}\n", pkgName).Output()
		if err != nil {
			return false, "", nil
		}
		parts := strings.SplitN(strings.TrimSpace(string(out)), "\t", 2)
		if len(parts) >= 1 && strings.Contains(parts[0], "installed") {
			ver := ""
			if len(parts) >= 2 {
				ver = strings.TrimSpace(parts[1])
			}
			return true, ver, nil
		}
		return false, "", nil

	case "dnf", "rpm", "zypper":
		if pkgName == "" || pkgName == "system-update" {
			return false, "", fmt.Errorf("system-wide upgrade transaction cannot be reconciled by single package name")
		}
		out, err := exec.CommandContext(ctx, "rpm", "-q", "--qf", "%{VERSION}-%{RELEASE}\n", pkgName).Output()
		if err != nil {
			return false, "", nil
		}
		return true, strings.TrimSpace(string(out)), nil

	case "flatpak":
		args := []string{"info"}
		if strings.EqualFold(scope, "user") {
			args = append(args, "--user")
		} else if strings.EqualFold(scope, "system") {
			args = append(args, "--system")
		}
		args = append(args, pkgName)
		out, err := exec.CommandContext(ctx, "flatpak", args...).Output()
		if err != nil {
			return false, "", nil
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "version:") {
				return true, strings.TrimSpace(strings.SplitN(line, ":", 2)[1]), nil
			}
		}
		return true, "", nil

	case "snap":
		out, err := exec.CommandContext(ctx, "snap", "list", pkgName).Output()
		if err != nil {
			return false, "", nil
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 2 {
				return true, fields[1], nil
			}
			return true, "", nil
		}
		return false, "", nil
	}
	return false, "", fmt.Errorf("no recovery probe available for provider %q", provider)
}

// ReconcileIncomplete inspects any journal transactions that were non-terminal when Arjax restarted,
// reconciling using installed versions, active locks, and operation intent.
// Crucially, an interrupted update where target version cannot be confirmed is marked outcome_unknown or recovery_required,
// never falsely assumed completed just because the package exists on disk.
func ReconcileIncomplete(ctx context.Context, journal *Journal, probe InstalledStateProbe, lockChecker LockChecker) []TransactionDTO {
	if journal == nil {
		return nil
	}
	if probe == nil {
		probe = DefaultHostInstalledProbe
	}

	incomplete := journal.Incomplete()
	if len(incomplete) == 0 {
		return nil
	}

	var reconciled []TransactionDTO
	now := time.Now()

	for _, tx := range incomplete {
		subCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		installed, ver, err := probe(subCtx, tx.Provider, tx.PackageName, tx.InstallScope)
		cancel()

		locked := false
		lockPath := ""
		if lockChecker != nil {
			locked, lockPath = lockChecker(tx.Provider)
		}

		tx.UpdatedAt = now
		tx.CompletedAt = &now
		tx.CanCancelSafely = false

		switch {
		case locked:
			// Host DB lock still exists after restart -> administrator/user must inspect lock & transaction state
			tx.State = TxFailed
			tx.RecoveryStatus = RecoveryRequired
			tx.ErrorCode = "recovery_required"
			tx.ErrorMessage = fmt.Sprintf("Previous transaction was interrupted and lock (%s) remains active; verify package database integrity before retrying.", lockPath)
			tx.Message = tx.ErrorMessage
			tx.Logs = append(tx.Logs, "[RECOVERY] Database lock still present -> recovery_required")

		case err != nil:
			tx.State = TxFailed
			tx.RecoveryStatus = RecoveryOutcomeUnknown
			tx.ErrorCode = "outcome_unknown"
			tx.ErrorMessage = fmt.Sprintf("Previous %s was interrupted and host state could not be confirmed (%v).", tx.Operation, err)
			tx.Message = tx.ErrorMessage
			tx.Logs = append(tx.Logs, "[RECOVERY] Host probe inconclusive -> outcome_unknown")

		case tx.Operation == OpInstall:
			if installed {
				tx.State = TxCompleted
				tx.Progress = 100
				tx.RecoveryStatus = RecoveryReconciledComplete
				tx.Message = fmt.Sprintf("Recovered after restart: %s (%s) completed installation on host.", tx.PackageName, ver)
				tx.Logs = append(tx.Logs, fmt.Sprintf("[RECOVERY] Verified %s installed (%s) -> reconciled_completed", tx.PackageName, ver))
			} else {
				tx.State = TxFailed
				tx.Progress = 100
				tx.RecoveryStatus = RecoveryReconciledFailed
				tx.ErrorCode = "interrupted_crash"
				tx.ErrorMessage = "Transaction was interrupted before installation completed."
				tx.Message = tx.ErrorMessage
				tx.Logs = append(tx.Logs, "[RECOVERY] Verified package absent on host -> reconciled_failed")
			}

		case tx.Operation == OpRemove:
			if !installed {
				tx.State = TxCompleted
				tx.Progress = 100
				tx.RecoveryStatus = RecoveryReconciledComplete
				tx.Message = fmt.Sprintf("Recovered after restart: %s removal succeeded on host.", tx.PackageName)
				tx.Logs = append(tx.Logs, "[RECOVERY] Verified package absent on host -> reconciled_completed")
			} else {
				tx.State = TxFailed
				tx.Progress = 100
				tx.RecoveryStatus = RecoveryReconciledFailed
				tx.ErrorCode = "interrupted_crash"
				tx.ErrorMessage = fmt.Sprintf("Removal of %s was interrupted; package (%s) remains installed.", tx.PackageName, ver)
				tx.Message = tx.ErrorMessage
				tx.Logs = append(tx.Logs, "[RECOVERY] Package still present -> reconciled_failed")
			}

		case tx.Operation == OpUpdate:
			// Presence alone NEVER establishes update completion! Compare TargetVersion vs PreviousVersion.
			if installed && tx.TargetVersion != "" && ver == tx.TargetVersion {
				tx.State = TxCompleted
				tx.Progress = 100
				tx.RecoveryStatus = RecoveryReconciledComplete
				tx.Message = fmt.Sprintf("Recovered after restart: %s reached target version %s.", tx.PackageName, ver)
				tx.Logs = append(tx.Logs, fmt.Sprintf("[RECOVERY] Version %s matches target %s -> reconciled_completed", ver, tx.TargetVersion))
			} else if installed && tx.PreviousVersion != "" && ver == tx.PreviousVersion {
				tx.State = TxFailed
				tx.Progress = 100
				tx.RecoveryStatus = RecoveryReconciledFailed
				tx.ErrorCode = "interrupted_crash"
				tx.ErrorMessage = fmt.Sprintf("Update was interrupted; %s remains at previous version %s.", tx.PackageName, ver)
				tx.Message = tx.ErrorMessage
				tx.Logs = append(tx.Logs, "[RECOVERY] Version unchanged from previous -> reconciled_failed")
			} else {
				tx.State = TxFailed
				tx.Progress = 100
				tx.RecoveryStatus = RecoveryOutcomeUnknown
				tx.ErrorCode = "outcome_unknown"
				tx.ErrorMessage = fmt.Sprintf("Update of %s was interrupted (host reports version %q, target %q); run full update check to verify consistency.", tx.PackageName, ver, tx.TargetVersion)
				tx.Message = tx.ErrorMessage
				tx.Logs = append(tx.Logs, "[RECOVERY] Interrupted update outcome ambiguous -> outcome_unknown")
			}
		}

		journal.Update(tx)
		cp := *tx
		cp.Logs = append([]string(nil), tx.Logs...)
		reconciled = append(reconciled, cp)
	}

	return reconciled
}
