package catalog

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestALPMComparatorMatchesSystemVercmp(t *testing.T) {
	vercmpPath, err := exec.LookPath("vercmp")
	if err != nil {
		t.Skip("vercmp binary not available on host; skipping authoritative ALPM cross-check")
	}

	cases := [][2]string{
		{"1.0-1", "1.0-1"},
		{"1.0-2", "1.0-1"},
		{"1:1.0-1", "2.0-1"},
		{"2:1.0-1", "1:99.0-1"},
		{"124.0.2-1", "124.0.10-1"},
		{"1.0.a-1", "1.0.1-1"},
		{"1.0a-1", "1.0b-1"},
		{"20240501-1", "20240430-1"},
		{"1.001-1", "1.1-1"},
		{"4.2.2", "4.2.2-1"},
		{"1.5", "1.5a"},
		{"1.5", "1.5.1"},
		{"0.9.9-1", "0.10.0-1"},
	}

	for _, c := range cases {
		out, err := exec.Command(vercmpPath, c[0], c[1]).Output()
		if err != nil {
			t.Fatalf("vercmp(%q, %q) failed: %v", c[0], c[1], err)
		}
		sysVal, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err != nil {
			t.Fatalf("invalid vercmp output %q: %v", string(out), err)
		}

		goVal, err := CompareVersionsSafe("pacman", c[0], c[1])
		if err != nil {
			t.Fatalf("CompareVersionsSafe failed for (%q, %q): %v", c[0], c[1], err)
		}

		if (sysVal > 0 && goVal <= 0) || (sysVal < 0 && goVal >= 0) || (sysVal == 0 && goVal != 0) {
			t.Errorf("ALPM mismatch vs /usr/bin/vercmp for (%q, %q): vercmp=%d, go=%d", c[0], c[1], sysVal, goVal)
		}
	}
}

func TestVersionComparatorRejectsReleaseChannelsAndCrossEpochs(t *testing.T) {
	if _, err := CompareVersionsSafe("flatpak", "stable", "beta"); !errors.Is(err, ErrNotComparableVersion) {
		t.Errorf("Expected ErrNotComparableVersion comparing Flatpak branches, got %v", err)
	}
	if _, err := CompareVersionsSafe("snap", "latest/stable", "latest/edge"); !errors.Is(err, ErrNotComparableVersion) {
		t.Errorf("Expected ErrNotComparableVersion comparing Snap channels, got %v", err)
	}
	if _, err := CompareCrossEcosystem("pacman", "1:124.0-1", "flatpak", "125.0"); !errors.Is(err, ErrNotComparableVersion) {
		t.Errorf("Expected ErrNotComparableVersion across differing distro epochs, got %v", err)
	}
	if cmp, err := CompareCrossEcosystem("pacman", "4.2.2-1", "flatpak", "4.2.1"); err != nil || cmp <= 0 {
		t.Errorf("Expected 4.2.2-1 > 4.2.1 across compatible tracks, got cmp=%d err=%v", cmp, err)
	}
}

func TestPackageProvenanceDetails(t *testing.T) {
	pacTrust := EvaluateSourceTrust("pacman", "extra", "firefox")
	if !pacTrust.OfficialRepo || pacTrust.Provenance.Origin == "" || pacTrust.Provenance.VerifiedAt.IsZero() {
		t.Errorf("Expected populated official Pacman provenance, got %+v", pacTrust)
	}
	aurTrust := EvaluateSourceTrust("aur", "aur", "firefox-bin")
	if aurTrust.OfficialRepo || aurTrust.SignatureChecked || !strings.Contains(aurTrust.Provenance.Verification, "PKGBUILD") {
		t.Errorf("Expected AUR provenance to distinguish unverified PKGBUILD from checksums, got %+v", aurTrust)
	}
}
