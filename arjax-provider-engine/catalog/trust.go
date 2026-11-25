package catalog

import (
	"fmt"
	"strings"
	"time"
)

// TrustLevel categorizes confidence in a package source separately from sandbox isolation and recommendation.
type TrustLevel string

const (
	TrustOfficial   TrustLevel = "official"    // Signed distribution repository (Arch core/extra/multilib, Debian/Fedora official)
	TrustVerified   TrustLevel = "verified"    // Verified upstream publisher (Flathub verified / Snap publisher)
	TrustCommunity  TrustLevel = "community"   // Community-submitted recipe (AUR, Flathub unverified, PPAs)
	TrustThirdParty TrustLevel = "third-party" // External upstream release or vendor binary (GitHub, AppImage)
	TrustUnknown    TrustLevel = "unknown"
)

// PackageProvenance captures cryptographic & origin evidence for an offer,
// explicitly noting what is verified vs unverified (e.g., AUR PKGBUILD recipe vs artifact hash).
type PackageProvenance struct {
	Origin         string    `json:"origin"`         // Repository/remote URI or ecosystem origin
	Publisher      string    `json:"publisher"`      // Organization, repo maintainer, or AUR submitter
	SigningKey     string    `json:"signingKey"`     // Keyring / GPG / OSTree / Assertion signer
	ArtifactDigest string    `json:"artifactDigest"` // Verification mechanism ("pacman-repo-sha256", "ostree-commit-sha256", "pkgbuild-srcinfo-sums")
	Verification   string    `json:"verification"`   // Clear statement of provenance scope (never claims runtime safety!)
	VerifiedAt     time.Time `json:"verifiedAt"`
}

// SourceTrust explicitly describes provenance, verification, and build model of a package offer
// without collapsing trust into isolation scope or recommendation ranking.
type SourceTrust struct {
	TrustLevel       TrustLevel        `json:"trustLevel"`       // official, verified, community, third-party, unknown
	OfficialRepo     bool              `json:"officialRepo"`     // True only for curated official distro repositories
	SignatureChecked bool              `json:"signatureChecked"` // True when keyring/GPG/OSTree signature verification applies
	SignatureStatus  string            `json:"signatureStatus"`  // "verified-keyring", "ostree-signed", "unverified-script", "release-checksum"
	Maintainer       string            `json:"maintainer"`       // Repository team, verified vendor, or AUR packager
	BuildRecipe      string            `json:"buildRecipe"`      // "binary-repo", "pkgbuild", "flatpak-manifest", "snap-store", "upstream-binary"
	Verified         bool              `json:"verified"`         // True when publisher or distro cryptographically vouches for package origin
	Summary          string            `json:"summary"`          // Human-readable provenance note
	Provenance       PackageProvenance `json:"provenance"`       // Structured origin & cryptographic provenance details
}

// EvaluateSourceTrust determines structured trust + provenance metadata independently of isolation/ranking.
func EvaluateSourceTrust(provider, repo, pkgName string) SourceTrust {
	prov := strings.ToLower(strings.TrimSpace(provider))
	r := strings.ToLower(strings.TrimSpace(repo))
	now := time.Now().UTC()

	if r == "aur" || (prov == "repository" && r == "aur") {
		prov = "aur"
	}

	switch prov {
	case "pacman", "apt", "dnf", "zypper", "deb", "rpm", "repository":
		repoLabel := r
		if repoLabel == "" {
			if prov == "apt" || prov == "deb" {
				repoLabel = "main"
			} else {
				repoLabel = "extra"
			}
		}
		signingKey := "Local pacman-key / distro master keyring"
		if prov == "apt" || prov == "deb" {
			signingKey = "Local apt-key / distribution GPG keyring"
		} else if prov == "dnf" || prov == "rpm" {
			signingKey = "Local RPM / distribution GPG keyring"
		} else if prov == "zypper" {
			signingKey = "Local Zypper / RPM GPG keyring"
		}
		isOfficialRepo := r == "" || r == prov || r == "core" || r == "extra" || r == "multilib" ||
			r == "main" || r == "contrib" || r == "non-free" || r == "non-free-firmware" ||
			r == "kali-rolling" || r == "stable" || r == "testing" || r == "unstable" ||
			r == "fedora" || r == "updates" || r == "oss" || r == "repository"
		if isOfficialRepo {
			return SourceTrust{
				TrustLevel:       TrustOfficial,
				OfficialRepo:     true,
				SignatureChecked: true,
				SignatureStatus:  "verified-keyring",
				Maintainer:       "Distribution Package Maintainers",
				BuildRecipe:      "binary-repo",
				Verified:         true,
				Summary:          "Cryptographically signed official distribution repository package.",
				Provenance: PackageProvenance{
					Origin:         fmt.Sprintf("distro-repo/%s", repoLabel),
					Publisher:      "Official Distro Packaging Team",
					SigningKey:     signingKey,
					ArtifactDigest: "repo-sync-db-sha256",
					Verification:   "Package archive and repo database signed by trusted distro packagers.",
					VerifiedAt:     now,
				},
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
			Provenance: PackageProvenance{
				Origin:         fmt.Sprintf("custom-repo/%s", repoLabel),
				Publisher:      "Configured Third-Party Repo Maintainer",
				SigningKey:     "Custom Keyring Entry",
				ArtifactDigest: "repo-sync-db-sha256",
				Verification:   "Signed by custom repository key; outside official distro audit scope.",
				VerifiedAt:     now,
			},
		}

	case "aur":
		recipeNote := "Community PKGBUILD recipe; recipe itself is unverified and requires inspection before building."
		if strings.HasSuffix(strings.ToLower(pkgName), "-bin") {
			recipeNote = "Community PKGBUILD repackaging upstream binaries; verify PKGBUILD script and upstream URL."
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
			Provenance: PackageProvenance{
				Origin:         fmt.Sprintf("https://aur.archlinux.org/packages/%s", pkgName),
				Publisher:      "AUR Package Submitter (Community)",
				SigningKey:     "None (unverified PKGBUILD script; artifact hashes declared in .SRCINFO)",
				ArtifactDigest: "pkgbuild-sha256sums-local-makepkg",
				Verification:   "Distinction: PKGBUILD script is community-authored without repo signature; makepkg verifies downloaded source archives against PKGBUILD checksums at build time.",
				VerifiedAt:     now,
			},
		}

	case "flatpak":
		remote := r
		if remote == "" {
			remote = "flathub"
		}
		isFlathub := remote == "flathub"
		level := TrustVerified
		if !isFlathub {
			level = TrustCommunity
		}
		return SourceTrust{
			TrustLevel:       level,
			OfficialRepo:     false,
			SignatureChecked: true,
			SignatureStatus:  "ostree-signed",
			Maintainer:       "Upstream / Flathub Buildbot",
			BuildRecipe:      "flatpak-manifest",
			Verified:         isFlathub,
			Summary:          "OSTree-signed Flatpak container; signature proves remote integrity rather than source safety.",
			Provenance: PackageProvenance{
				Origin:         fmt.Sprintf("flatpak:%s/%s", remote, pkgName),
				Publisher:      "Flathub Remote / App Maintainer",
				SigningKey:     "Flathub OSTree GPG Signing Key",
				ArtifactDigest: "ostree-commit-sha256",
				Verification:   "OSTree commit signature verified; check AppStream verified status & sandbox permissions independently.",
				VerifiedAt:     now,
			},
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
			Summary:          "Signed SquashFS Snap image asserted by Snapcraft Store.",
			Provenance: PackageProvenance{
				Origin:         fmt.Sprintf("snapcraft/%s", pkgName),
				Publisher:      "Snapcraft Publisher Assertion",
				SigningKey:     "Snapcraft Store Assertion Key",
				ArtifactDigest: "snap-squashfs-sha3-384",
				Verification:   "Snap assertion & SquashFS digest validated by snapd.",
				VerifiedAt:     now,
			},
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
			Provenance: PackageProvenance{
				Origin:         fmt.Sprintf("upstream/%s", pkgName),
				Publisher:      "Upstream Project Release",
				SigningKey:     "None enforced automatically",
				ArtifactDigest: "tls-transport-only",
				Verification:   "Fetched over HTTPS; distro keyring verification does not apply.",
				VerifiedAt:     now,
			},
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
			Provenance: PackageProvenance{
				Origin:       "unknown",
				Verification: "No provenance guarantees available.",
				VerifiedAt:   now,
			},
		}
	}
}
