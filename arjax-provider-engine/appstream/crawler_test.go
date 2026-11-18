package appstream

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestCrawler_GetAppDetails(t *testing.T) {
	c := NewCrawler()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test 1: Well-known app ID (e.g., Blender)
	details, err := c.GetAppstream(ctx, "org.blender.Blender")
	if err != nil {
		t.Logf("Notice: GetAppstream returned error (may be offline): %v", err)
	} else if details != nil {
		t.Logf("Found app: %s (%s)", details.Name, details.Summary)
		if len(details.Screenshots) == 0 {
			t.Logf("Warning: no screenshots returned for blender")
		}
	}

	// Test 2: Search
	results, err := c.SearchApps(ctx, "blender", 5)
	if err != nil {
		t.Logf("Notice: SearchApps returned error (may be offline): %v", err)
	} else {
		t.Logf("Search returned %d results", len(results))
		for _, r := range results {
			t.Logf(" - Result: %s (%s)", r.Name, r.ID)
		}
	}
}

func TestCrawler_LocalXML(t *testing.T) {
	xmlPath := "/var/lib/flatpak/appstream/flathub/x86_64/active/appstream.xml"
	if _, err := os.Stat(xmlPath); err != nil {
		t.Skip("Local flathub appstream XML not present, skipping XML test")
	}

	c := NewCrawler()
	ctx := context.Background()

	// Should fetch instantly from local XML or remote
	details, err := c.GetAppstream(ctx, "org.mozilla.firefox")
	if err != nil {
		t.Errorf("GetAppstream failed: %v", err)
	}
	if details != nil {
		t.Logf("Resolved Firefox: Name=%s, Developer=%s, Screenshots=%d", details.Name, details.Developer, len(details.Screenshots))
	}
}
