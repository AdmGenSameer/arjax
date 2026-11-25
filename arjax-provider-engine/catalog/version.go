package catalog

import (
	"strconv"
	"strings"
	"unicode"
)

// CompareVersions performs provider-aware version comparison.
// Returns > 0 if a > b, < 0 if a < b, 0 if equal.
func CompareVersions(provider, a, b string) int {
	va := strings.TrimSpace(a)
	vb := strings.TrimSpace(b)
	if va == vb {
		return 0
	}
	if va == "" || strings.EqualFold(va, "latest") {
		if vb == "" || strings.EqualFold(vb, "latest") {
			return 0
		}
		return -1
	}
	if vb == "" || strings.EqualFold(vb, "latest") {
		return 1
	}

	prov := strings.ToLower(strings.TrimSpace(provider))
	switch prov {
	case "pacman", "aur", "paru", "repository":
		return compareALPM(va, vb)
	case "apt", "deb":
		return compareDebian(va, vb)
	case "dnf", "rpm", "zypper":
		return compareRPM(va, vb)
	default:
		return compareGenericSemver(va, vb)
	}
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
	if rela == "" && relb == "" {
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

// compareRPM implements RPM EVR comparison with '~' pre-release handling
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

	for i < na || j < nb {
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

	if i >= na && j >= nb {
		return 0
	}
	if i < na {
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
