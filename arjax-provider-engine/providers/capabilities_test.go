package providers

import (
	"context"
	"testing"
)

func TestProviderStatusSeparatesSupportFromRuntimeHealth(t *testing.T) {
	matrix := map[string]ProviderCapability{
		"pacman":   DefaultCapabilityMatrix()["pacman"],
		"flatpak":  {Provider: "flatpak", DisplayName: "Flatpak", Search: SupportFull, Install: SupportFull},
		"snap":     DefaultCapabilityMatrix()["snap"],
		"appimage": DefaultCapabilityMatrix()["appimage"],
	}

	statuses := ProbeProviderStatuses(context.Background(), matrix)
	var foundAppImage, foundFlatpak bool
	for _, st := range statuses {
		if st.Provider == "appimage" {
			foundAppImage = true
			if st.Supported {
				t.Errorf("Planned AppImage must have Supported=false until implemented")
			}
		}
		if st.Provider == "flatpak" {
			foundFlatpak = true
			if !st.Supported {
				t.Errorf("Flatpak integration is implemented so Supported must be true regardless of host binary presence")
			}
			if !st.Installed && st.StatusMessage == "" {
				t.Errorf("Uninstalled supported provider must provide an actionable UI message")
			}
		}
	}
	if !foundAppImage || !foundFlatpak {
		t.Errorf("Expected appimage and flatpak in ProbeProviderStatuses")
	}
}

type fakeAptRepoProvider struct {
	*RepositoryProvider
}

func (f *fakeAptRepoProvider) IsSupported() bool { return true }

func TestAptRepositorySearchParsingAndCapabilityMapping(t *testing.T) {
	rp := NewRepositoryProvider("apt")
	sample := `Sorting...
Full Text Search...
nmap/kali-rolling,now 7.95+dfsg-2kali1 amd64 [installed]
  The Network Mapper

curl-minimal/kali-rolling 8.10.1-1 amd64
  command line tool for transferring data
`
	pkgs := rp.parseSearchResults(sample, "nmap")
	if len(pkgs) != 2 {
		t.Fatalf("Expected 2 parsed apt packages, got %d (%+v)", len(pkgs), pkgs)
	}
	if pkgs[0].Name != "nmap" || pkgs[0].Source != "apt" || pkgs[0].Repo != "kali-rolling" || pkgs[0].Version != "7.95+dfsg-2kali1" || pkgs[0].Arch != "amd64" {
		t.Fatalf("Unexpected apt parse result[0]: %+v", pkgs[0])
	}
	if pkgs[1].Name != "curl-minimal" || pkgs[1].Repo != "kali-rolling" {
		t.Fatalf("Hyphenated apt package name was mangled: %+v", pkgs[1])
	}

	// Registering RepositoryProvider("apt") must NOT flip pacman capability AvailableOnHost to true
	pm := &ProviderManager{providers: []Provider{&fakeAptRepoProvider{rp}}}
	matrix := pm.GetCapabilityMatrix()
	if matrix["deb"].Provider != "deb" || !matrix["deb"].AvailableOnHost {
		t.Errorf("Expected deb capability to be marked available for apt RepositoryProvider, got %+v", matrix["deb"])
	}
}


