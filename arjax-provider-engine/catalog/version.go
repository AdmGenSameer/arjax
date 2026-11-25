package catalog

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// ErrNotComparableVersion indicates that one or both strings are release tracks/channels
// (e.g. Flatpak "stable"/"beta", Snap "latest/edge") or span incompatible packaging epochs.
var ErrNotComparableVersion = errors.New("values represent release channels or incompatible versioning schemes")

// VersionComparator defines a strictly provider-isolated package version comparator.
type VersionComparator interface {
	Supports(provider string) bool
	Compare(a, b string) (int, error)
}

// IsReleaseTrack returns true when a string denotes a Flatpak branch or Snap channel
// rather than an ordinal numeric package version.
func IsReleaseTrack(v string) bool {
	low := strings.ToLower(strings.TrimSpace(v))
	if low == "" {
		return true
	}
	switch low {
	case "latest", "stable", "beta", "master", "edge", "candidate", "nightly", "canary":
		return true
	}
	if strings.Contains(low, "/") {
		// e.g. Snap "latest/stable", "latest/edge", "2.x/candidate"
		return true
	}
	return false
}

// ALPMComparator implements Arch Linux libalpm vercmp semantics ([epoch:]pkgver[-pkgrel]).
type ALPMComparator struct{}

func (ALPMComparator) Supports(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "pacman", "aur", "paru":
		return true
	default:
		return false
	}
}

func (ALPMComparator) Compare(a, b string) (int, error) {
	va, vb := strings.TrimSpace(a), strings.TrimSpace(b)
	if IsReleaseTrack(va) || IsReleaseTrack(vb) {
		return 0, ErrNotComparableVersion
	}
	return compareALPM(va, vb), nil
}

// DebianComparator implements Debian dpkg version semantics ([epoch:]upstream[-debian_revision]).
type DebianComparator struct{}

func (DebianComparator) Supports(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "apt", "deb":
		return true
	default:
		return false
	}
}

func (DebianComparator) Compare(a, b string) (int, error) {
	va, vb := strings.TrimSpace(a), strings.TrimSpace(b)
	if IsReleaseTrack(va) || IsReleaseTrack(vb) {
		return 0, ErrNotComparableVersion
	}
	return compareDebian(va, vb), nil
}

// RPMComparator implements RPM EVR semantics with tilde (~) and caret (^) awareness.
type RPMComparator struct{}

func (RPMComparator) Supports(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "dnf", "rpm", "zypper":
		return true
	default:
		return false
	}
}

func (RPMComparator) Compare(a, b string) (int, error) {
	va, vb := strings.TrimSpace(a), strings.TrimSpace(b)
	if IsReleaseTrack(va) || IsReleaseTrack(vb) {
		return 0, ErrNotComparableVersion
	}
	return compareRPM(va, vb), nil
}

// NumericReleaseComparator compares concrete numeric/semver strings for Flatpak, Snap, GitHub,
// while rejecting non-version channels/branches.
type NumericReleaseComparator struct{}

func (NumericReleaseComparator) Supports(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "flatpak", "snap", "github", "appimage":
		return true
	default:
		return false
	}
}

func (NumericReleaseComparator) Compare(a, b string) (int, error) {
	va, vb := strings.TrimSpace(a), strings.TrimSpace(b)
	if IsReleaseTrack(va) || IsReleaseTrack(vb) {
		return 0, ErrNotComparableVersion
	}
	return compareGenericSemver(va, vb), nil
}

var registeredComparators = []VersionComparator{
	ALPMComparator{},
	DebianComparator{},
	RPMComparator{},
	NumericReleaseComparator{},
}

func resolveVersionProvider(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "repository" || p == "" {
		detected := strings.ToLower(strings.TrimSpace(utils.DetectPackageManager()))
		if detected == "paru" || detected == "yay" || detected == "" {
			return "pacman"
		}
		return detected
	}
	return p
}

// GetVersionComparator selects the isolated comparator for a package ecosystem.
func GetVersionComparator(provider string) VersionComparator {
	resolved := resolveVersionProvider(provider)
	for _, c := range registeredComparators {
		if c.Supports(resolved) {
			return c
		}
	}
	return NumericReleaseComparator{}
}

// CompareVersionsSafe compares two versions using provider-specific logic and returns
// ErrNotComparableVersion if either string is a channel/branch or ecosystems are incomparable.
func CompareVersionsSafe(provider, a, b string) (int, error) {
	comp := GetVersionComparator(provider)
	return comp.Compare(a, b)
}

// CompareCrossEcosystem compares two offers only when both have comparable numeric versions
// without incompatible epoch schemes (epochs remain strictly distro-scoped!).
func CompareCrossEcosystem(provA, verA, provB, verB string) (int, error) {
	if IsReleaseTrack(verA) || IsReleaseTrack(verB) {
		return 0, ErrNotComparableVersion
	}
	pa := strings.ToLower(strings.TrimSpace(provA))
	pb := strings.ToLower(strings.TrimSpace(provB))
	if pa == pb {
		return CompareVersionsSafe(pa, verA, verB)
	}
	// Epochs (1:2.0 vs 2.0) are distro packaging metadata, not cross-ecosystem comparable
	ea, ra := splitEpoch(strings.TrimSpace(verA))
	eb, rb := splitEpoch(strings.TrimSpace(verB))
	if ea != eb {
		return 0, fmt.Errorf("%w: distro epochs differ (%d vs %d)", ErrNotComparableVersion, ea, eb)
	}
	upA, _ := splitLastRelease(strings.TrimPrefix(strings.TrimPrefix(ra, "v"), "V"))
	upB, _ := splitLastRelease(strings.TrimPrefix(strings.TrimPrefix(rb, "v"), "V"))
	return rpmvercmp(upA, upB, true), nil
}

// CompareVersions provides backwards-compatible int comparison inside single-provider contexts.
func CompareVersions(provider, a, b string) int {
	va := strings.TrimSpace(a)
	vb := strings.TrimSpace(b)
	if va == vb {
		return 0
	}
	if IsReleaseTrack(va) {
		if IsReleaseTrack(vb) {
			return 0
		}
		return -1
	}
	if IsReleaseTrack(vb) {
		return 1
	}
	cmp, err := CompareVersionsSafe(provider, va, vb)
	if err != nil {
		return strings.Compare(va, vb)
	}
	return cmp
}

func splitEpoch(v string) (int, string) {
	idx := strings.IndexByte(v, ':')
	if idx > 0 {
		if epoch, err := strconv.Atoi(v[:idx]); err == nil {
			return epoch, v[idx+1:]
		}
	}
	return 0, v
}

// compareALPM implements Arch libalpm vercmp semantics: [epoch:]pkgver[-pkgrel]
func compareALPM(a, b string) int {
	if a == b {
		return 0
	}
	ea, ra := splitEpoch(a)
	eb, rb := splitEpoch(b)
	if ea != eb {
		if ea > eb {
			return 1
		}
		return -1
	}

	va, rela := splitLastRelease(ra)
	vb, relb := splitLastRelease(rb)

	if c := rpmvercmp(va, vb, false); c != 0 {
		return c
	}
	// libalpm rule: pkgrel is only compared when both strings specify a release!
	if rela == "" || relb == "" {
		return 0
	}
	return rpmvercmp(rela, relb, false)
}

// compareDebian implements Debian version comparison with '~' pre-release handling
func compareDebian(a, b string) int {
	ea, ra := splitEpoch(a)
	eb, rb := splitEpoch(b)
	if ea != eb {
		if ea > eb {
			return 1
		}
		return -1
	}
	va, rela := splitLastRelease(ra)
	vb, relb := splitLastRelease(rb)
	if c := rpmvercmp(va, vb, true); c != 0 {
		return c
	}
	return rpmvercmp(rela, relb, true)
}

// compareRPM implements RPM EVR comparison with '~' and '^' handling
func compareRPM(a, b string) int {
	ea, ra := splitEpoch(a)
	eb, rb := splitEpoch(b)
	if ea != eb {
		if ea > eb {
			return 1
		}
		return -1
	}
	va, rela := splitLastRelease(ra)
	vb, relb := splitLastRelease(rb)
	if c := rpmvercmp(va, vb, true); c != 0 {
		return c
	}
	return rpmvercmp(rela, relb, true)
}

func compareGenericSemver(a, b string) int {
	va := strings.TrimPrefix(strings.TrimPrefix(a, "v"), "V")
	vb := strings.TrimPrefix(strings.TrimPrefix(b, "v"), "V")
	return rpmvercmp(va, vb, true)
}

func splitLastRelease(v string) (string, string) {
	idx := strings.LastIndexByte(v, '-')
	if idx < 0 {
		return v, ""
	}
	return v[:idx], v[idx+1:]
}

func rpmvercmp(a, b string, handleTilde bool) int {
	if a == b {
		return 0
	}
	i, j := 0, 0
	na, nb := len(a), len(b)

	for i < na && j < nb {
		for i < na && !isAlnum(a[i]) && (!handleTilde || a[i] != '~') {
			i++
		}
		for j < nb && !isAlnum(b[j]) && (!handleTilde || b[j] != '~') {
			j++
		}

		if handleTilde {
			aTilde := i < na && a[i] == '~'
			bTilde := j < nb && b[j] == '~'
			if aTilde || bTilde {
				if aTilde && bTilde {
					i++
					j++
					continue
				}
				if aTilde {
					return -1
				}
				return 1
			}
		}

		if i >= na || j >= nb {
			break
		}

		startA := i
		startB := j
		isNumA := isDigit(a[i])
		isNumB := isDigit(b[j])

		if isNumA != isNumB {
			if isNumA {
				return 1
			}
			return -1
		}

		if isNumA {
			for i < na && isDigit(a[i]) {
				i++
			}
			for j < nb && isDigit(b[j]) {
				j++
			}
			segA := strings.TrimLeft(a[startA:i], "0")
			segB := strings.TrimLeft(b[startB:j], "0")
			if len(segA) != len(segB) {
				if len(segA) > len(segB) {
					return 1
				}
				return -1
			}
			if segA != segB {
				if segA > segB {
					return 1
				}
				return -1
			}
		} else {
			for i < na && isAlpha(a[i]) {
				i++
			}
			for j < nb && isAlpha(b[j]) {
				j++
			}
			segA := a[startA:i]
			segB := b[startB:j]
			if segA != segB {
				if segA > segB {
					return 1
				}
				return -1
			}
		}
	}

	for i < na && !isAlnum(a[i]) && (!handleTilde || a[i] != '~') {
		i++
	}
	for j < nb && !isAlnum(b[j]) && (!handleTilde || b[j] != '~') {
		j++
	}
	if i >= na && j >= nb {
		return 0
	}
	// Trailing separator check & libalpm rule: remaining alphabetic suffix (e.g. 1.5a) sorts older than 1.5,
	// whereas remaining numeric segment (e.g. 1.5.1) sorts newer!
	if handleTilde && i < na && a[i] == '~' {
		return -1
	}
	if handleTilde && j < nb && b[j] == '~' {
		return 1
	}
	if i < na {
		if isAlpha(a[i]) {
			return -1
		}
		return 1
	}
	if isAlpha(b[j]) {
		return 1
	}
	return -1
}

func isDigit(b byte) bool {
	return unicode.IsDigit(rune(b))
}

func isAlpha(b byte) bool {
	return unicode.IsLetter(rune(b))
}

func isAlnum(b byte) bool {
	return isDigit(b) || isAlpha(b)
}
