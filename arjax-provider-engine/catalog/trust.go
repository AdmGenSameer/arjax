package catalog

import "strings"

// TrustLevel categorizes confidence in a package source separately from sandbox isolation and recommendation.
type TrustLevel string

const (
	TrustOfficial   TrustLevel = "official"    // Signed distribution repository (Arch core/extra/multilib, Debian/Fedora official)
	TrustVerified   TrustLevel = "verified"    // Verified upstream publisher (Flathub verified / Snap verified)
	TrustCommunity  TrustLevel = "community"   // Community-submitted recipe (AUR, Flathub unverified, PPAs)
	TrustThirdParty TrustLevel = "third-party" // External upstream release or vendor binary (GitHub, AppImage)
	TrustUnknown    TrustLevel = "unknown"
)

// SourceTrust explicitly describes provenance, verification, and build model of a package offer
// without collapsing trust into isolation scope or recommendation ranking.
type SourceTrust struct {
	TrustLevel       TrustLevel `json:"trustLevel"`       // official, verified, community, third-party, unknown
	OfficialRepo     bool       `json:"officialRepo"`     // True only for curated official distro repositories
	SignatureChecked bool       `json:"signatureChecked"` // True when keyring/GPG/OSTree signature verification applies
	SignatureStatus  string     `json:"signatureStatus"`  // "verified-keyring", "ostree-signed", "unverified-script", "release-checksum"
	Maintainer       string     `json:"maintainer"`       // Repository team, verified vendor, or AUR packager
	BuildRecipe      string     `json:"buildRecipe"`      // "binary-repo", "pkgbuild", "flatpak-manifest", "snap-store", "upstream-binary"
	Verified         bool       `json:"verified"`         // True when publisher or distro cryptographically vouches for package
	Summary          string     `json:"summary"`          // Human-readable provenance note
}

// EvaluateSourceTrust determines structured trust metadata for a package source independently of isolation/ranking.
func EvaluateSourceTrust(provider, repo, pkgName string) SourceTrust {
	prov := strings.ToLower(strings.TrimSpace(provider))
	r := strings.ToLower(strings.TrimSpace(repo))

	// If paru or another helper leaked an AUR package tagged under repository, classify accurately as AUR community!
	if r == "aur" || (prov == "repository" && r == "aur") {
		prov = "aur"
	}

	switch prov {
	case "pacman", "apt", "dnf", "zypper", "deb", "rpm", "repository":
		if r == "" || r == "core" || r == "extra" || r == "multilib" || r == "main" || r == "fedora" || r == "updates" || r == "repository" {
			return SourceTrust{
				TrustLevel:       TrustOfficial,
				OfficialRepo:     true,
				SignatureChecked: true,
				SignatureStatus:  "verified-keyring",
				Maintainer:       "Distribution Package Maintainers",
				BuildRecipe:      "binary-repo",
				Verified:         true,
				Summary:          "Cryptographically signed official distribution repository package.",
			}
		}
		return SourceTrust{
			TrustLevel:       TrustCommunity,
			OfficialRepo:     false,
			SignatureChecked: true,
			SignatureStatus:  "third-party-repo",
			Maintainer:       "Third-Party Repository",
			BuildRecipe:      "binary-repo",
			Verified:         false,
			Summary:          "Configured external/testing repository package.",
		}

	case "aur":
		recipeNote := "Community PKGBUILD recipe; review build script before installing."
		if strings.HasSuffix(strings.ToLower(pkgName), "-bin") {
			recipeNote = "Community PKGBUILD repackaging upstream binaries; verify maintainer trust."
		}
		return SourceTrust{
			TrustLevel:       TrustCommunity,
			OfficialRepo:     false,
			SignatureChecked: false,
			SignatureStatus:  "unverified-script",
			Maintainer:       "AUR Community Contributor",
			BuildRecipe:      "pkgbuild",
			Verified:         false,
			Summary:          recipeNote,
		}

	case "flatpak":
		if r == "flathub" || r == "" {
			return SourceTrust{
				TrustLevel:       TrustVerified,
				OfficialRepo:     false,
				SignatureChecked: true,
				SignatureStatus:  "ostree-signed",
				Maintainer:       "Upstream / Flathub Buildbot",
				BuildRecipe:      "flatpak-manifest",
				Verified:         true,
				Summary:          "OSTree-signed Flathub build with deterministic manifest.",
			}
		}
		return SourceTrust{
			TrustLevel:       TrustCommunity,
			OfficialRepo:     false,
			SignatureChecked: true,
			SignatureStatus:  "ostree-signed",
			Maintainer:       "Custom Flatpak Remote",
			BuildRecipe:      "flatpak-manifest",
			Verified:         false,
			Summary:          "Flatpak application from custom remote " + repo + ".",
		}

	case "snap":
		return SourceTrust{
			TrustLevel:       TrustVerified,
			OfficialRepo:     false,
			SignatureChecked: true,
			SignatureStatus:  "snap-assertion",
			Maintainer:       "Snapcraft Store Publisher",
			BuildRecipe:      "snap-store",
			Verified:         true,
			Summary:          "Signed SquashFS Snap image from Snapcraft.",
		}

	case "github", "appimage":
		return SourceTrust{
			TrustLevel:       TrustThirdParty,
			OfficialRepo:     false,
			SignatureChecked: false,
			SignatureStatus:  "upstream-asset",
			Maintainer:       "Upstream Release Asset",
			BuildRecipe:      "upstream-binary",
			Verified:         false,
			Summary:          "Direct upstream binary outside distro keyring.",
		}

	default:
		return SourceTrust{
			TrustLevel:       TrustUnknown,
			OfficialRepo:     false,
			SignatureChecked: false,
			SignatureStatus:  "unknown",
			Maintainer:       "Unknown Source",
			BuildRecipe:      "unknown",
			Verified:         false,
			Summary:          "Unclassified package source.",
		}
	}
}

