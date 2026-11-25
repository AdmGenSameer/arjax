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

