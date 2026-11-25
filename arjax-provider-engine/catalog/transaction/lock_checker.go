package transaction

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// LockChecker detects active host package-manager database locks before mutating system state.
type LockChecker func(provider string) (locked bool, lockPath string)

func isPosixFileLocked(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	lk := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: 0,
		Start:  0,
		Len:    0,
	}
	if err := syscall.FcntlFlock(f.Fd(), syscall.F_GETLK, &lk); err != nil {
		return false
	}
	return lk.Type != syscall.F_UNLCK
}

// DefaultHostLockChecker checks authoritative native package-manager lock files (db.lck, dpkg, dnf, zypper).
func DefaultHostLockChecker(provider string) (bool, string) {
	prov := strings.ToLower(strings.TrimSpace(provider))
	if prov == "repository" || prov == "" {
		detected := strings.ToLower(strings.TrimSpace(utils.DetectPackageManager()))
		if detected != "" {
			prov = detected
		}
	}
	switch prov {
	case "pacman", "aur", "paru":
		if _, err := os.Stat("/var/lib/pacman/db.lck"); err == nil {
			return true, "/var/lib/pacman/db.lck"
		}
	case "deb", "apt":
		for _, lockFile := range []string{"/var/lib/dpkg/lock-frontend", "/var/lib/dpkg/lock", "/var/lib/apt/lists/lock"} {
			if isPosixFileLocked(lockFile) {
				return true, lockFile
			}
		}
	case "rpm", "dnf":
		if _, err := os.Stat("/var/run/dnf.pid"); err == nil {
			return true, "/var/run/dnf.pid"
		}
	case "zypper":
		if _, err := os.Stat("/var/run/zypp.pid"); err == nil {
			return true, "/var/run/zypp.pid"
		}
	}
	return false, ""
}

// AcquireProcessCoordLock obtains an advisory cross-process Arjax lock file (between concurrent GUI & TUI
// processes) in XDG_RUNTIME_DIR, complementarily improving UX while NEVER replacing native PM locks.
func AcquireProcessCoordLock(provider string) (release func(), err error) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	prov := strings.ToLower(strings.TrimSpace(provider))
	if prov == "" {
		prov = "default"
	}
	lockPath := filepath.Join(runtimeDir, fmt.Sprintf("arjax-tx-%s.lock", prov))
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		// Non-fatal: native provider locks remain the authoritative guard
		return func() {}, nil
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another Arjax instance (GUI/TUI) is already modifying %s (%s)", prov, lockPath)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

