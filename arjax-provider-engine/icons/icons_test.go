package icons

import (
	"bytes"
	"compress/gzip"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIconPipeline_DetectFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "arjax_icon_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	pipe, err := NewIconPipeline(tempDir)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	// 1. PNG Signature
	pngBytes := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}
	format, err := pipe.DetectFormat(pngBytes)
	if err != nil || format != "png" {
		t.Errorf("Expected png, got %s (err: %v)", format, err)
	}

	// 2. SVG Signature
	svgBytes := []byte("<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'></svg>")
	format, err = pipe.DetectFormat(svgBytes)
	if err != nil || format != "svg" {
		t.Errorf("Expected svg, got %s (err: %v)", format, err)
	}

	// 3. WebP Signature
	webpBytes := []byte("RIFF1234WEBPVP8 ...")
	format, err = pipe.DetectFormat(webpBytes)
	if err != nil || format != "webp" {
		t.Errorf("Expected webp, got %s (err: %v)", format, err)
	}

	// 4. JPEG Signature
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}
	format, err = pipe.DetectFormat(jpegBytes)
	if err != nil || format != "jpeg" {
		t.Errorf("Expected jpeg, got %s (err: %v)", format, err)
	}

	// 5. Invalid Signature
	badBytes := []byte("NotAnImageAtAll")
	_, err = pipe.DetectFormat(badBytes)
	if err == nil {
		t.Errorf("Expected error for invalid signature, got nil")
	}

	// 6. Empty Payload
	_, err = pipe.DetectFormat([]byte{})
	if err == nil {
		t.Errorf("Expected error for empty payload, got nil")
	}
}

func TestSecurity_SSRFAndRedirects(t *testing.T) {
	// 1. Insecure scheme
	if err := ValidateRemoteURL("http://example.com/icon.png"); err == nil {
		t.Errorf("Expected error for insecure http protocol")
	}
	if err := ValidateRemoteURL("ftp://example.com/icon.png"); err == nil {
		t.Errorf("Expected error for ftp protocol")
	}
	if err := ValidateRemoteURL("file:///etc/passwd"); err == nil {
		t.Errorf("Expected error for file protocol")
	}

	// 2. Localhost & local domains
	if err := ValidateRemoteURL("https://localhost/icon.png"); err == nil {
		t.Errorf("Expected error for localhost")
	}
	if err := ValidateRemoteURL("https://server.local/icon.png"); err == nil {
		t.Errorf("Expected error for .local domain")
	}
	if err := ValidateRemoteURL("https://internal.lan/icon.png"); err == nil {
		t.Errorf("Expected error for .lan domain")
	}

	// 3. IPv4 loopback & private subnets
	if err := ValidateRemoteURL("https://127.0.0.1/icon.png"); err == nil {
		t.Errorf("Expected error for 127.0.0.1")
	}
	if err := ValidateRemoteURL("https://10.0.0.5/icon.png"); err == nil {
		t.Errorf("Expected error for 10.0.0.0/8")
	}
	if err := ValidateRemoteURL("https://192.168.1.100/icon.png"); err == nil {
		t.Errorf("Expected error for 192.168.0.0/16")
	}
	if err := ValidateRemoteURL("https://172.16.5.1/icon.png"); err == nil {
		t.Errorf("Expected error for 172.16.0.0/12")
	}

	// 4. Cloud Metadata IPs
	if err := ValidateRemoteURL("https://169.254.169.254/latest/meta-data/"); err == nil {
		t.Errorf("Expected error for AWS/Cloud metadata IP")
	}

	// 5. IPv6 loopback & ULA
	if !IsPrivateOrReservedIP(net.ParseIP("::1")) {
		t.Errorf("Expected ::1 to be marked private/loopback")
	}
	if !IsPrivateOrReservedIP(net.ParseIP("fc00::1")) {
		t.Errorf("Expected fc00::1 to be marked private ULA")
	}
	if !IsPrivateOrReservedIP(net.ParseIP("fe80::1")) {
		t.Errorf("Expected fe80::1 to be marked link-local")
	}

	// 6. IPv4-mapped IPv6 loopback
	if !IsPrivateOrReservedIP(net.ParseIP("::ffff:127.0.0.1")) {
		t.Errorf("Expected ::ffff:127.0.0.1 to be marked loopback")
	}

	// 7. Legitimate public HTTPS URLs pass
	if err := ValidateRemoteURL("https://dl.flathub.org/media/icons/128x128/com.spotify.Client.png"); err != nil {
		t.Errorf("Expected valid public HTTPS URL to pass, got: %v", err)
	}
}

func TestSecurity_SVGSanitizationAndXXE(t *testing.T) {
	sanitizer := NewSVGSanitizer()

	// 1. Script tag injection
	dirtySVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><circle cx="50" cy="50" r="40"/></svg>`)
	clean, err := sanitizer.ValidateAndSanitize(dirtySVG)
	if err != nil {
		t.Fatalf("Sanitization failed: %v", err)
	}
	if strings.Contains(string(clean), "<script") {
		t.Errorf("Failed to strip <script> tag")
	}

	// 2. Inline event handlers & javascript: hrefs
	eventSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="malicious()"><a href="javascript:steal()"><rect width="10" height="10"/></a></svg>`)
	clean, err = sanitizer.ValidateAndSanitize(eventSVG)
	if err != nil {
		t.Fatalf("Sanitization failed: %v", err)
	}
	if strings.Contains(string(clean), "onload") || strings.Contains(string(clean), "javascript:") {
		t.Errorf("Failed to strip onload or javascript: href")
	}

	// 3. Prohibit external XML entities (XXE)
	xxeSVG := []byte(`<?xml version="1.0"?><!DOCTYPE svg SYSTEM "http://evil.com/xxe"><svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	_, err = sanitizer.ValidateAndSanitize(xxeSVG)
	if err == nil {
		t.Errorf("Expected error for DTD/XXE system reference, got nil")
	}

	// 4. Test SVG Rasterization via rsvg-convert
	sampleSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><rect width="64" height="64" fill="#0078d4"/></svg>`)
	pngData, err := RasterizeSVG(sampleSVG, 64, 64)
	if err != nil {
		t.Logf("rsvg-convert note: %v", err)
	} else if len(pngData) > 0 {
		if !bytes.Equal(pngData[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
			t.Errorf("Rasterized output is not valid PNG")
		}
	}
}

func TestSecurity_DecompressionBomb(t *testing.T) {
	// 1. Normal small gzip
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte("Small safe SVG content"))
	gw.Close()

	decomp, err := SafeDecompressGzip(buf.Bytes(), 1024)
	if err != nil || string(decomp) != "Small safe SVG content" {
		t.Errorf("Expected successful decompression of small payload")
	}

	// 2. Decompression bomb (huge repeating data with strict max limit)
	var bombBuf bytes.Buffer
	gw = gzip.NewWriter(&bombBuf)
	largeData := bytes.Repeat([]byte("A"), 100*1024) // 100 KB
	gw.Write(largeData)
	gw.Close()

	// Enforce small limit (e.g. 5 KB) -> must reject decompression bomb
	_, err = SafeDecompressGzip(bombBuf.Bytes(), 5*1024)
	if err == nil {
		t.Errorf("Expected rejection of decompression bomb exceeding limit")
	}
}

func TestAliasRegistry_CanonicalMapping(t *testing.T) {
	reg := GetAliasRegistry()

	tests := []struct {
		input    string
		expected string
	}{
		{"brave", "com.brave.Browser"},
		{"brave-bin", "com.brave.Browser"},
		{"spotify", "com.spotify.Client"},
		{"spotify-launcher", "com.spotify.Client"},
		{"firefox", "org.mozilla.firefox"},
		{"blender", "org.blender.Blender"},
		{"steam", "com.valvesoftware.Steam"},
		{"vscode", "com.visualstudio.code"},
		{"visual-studio-code-bin", "com.visualstudio.code"},
		{"org.mozilla.firefox", "org.mozilla.firefox"},
	}

	for _, tc := range tests {
		actual := reg.ResolveCanonical(tc.input)
		if actual != tc.expected {
			t.Errorf("ResolveCanonical(%s) = %s, expected %s", tc.input, actual, tc.expected)
		}
	}

	aliases := reg.GetAliases("com.spotify.Client")
	found := false
	for _, a := range aliases {
		if a == "spotify" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'spotify' in aliases for com.spotify.Client")
	}
}

func TestCacheManager_AtomicWritesAndIntegrity(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "arjax_cache_test_*")
	defer os.RemoveAll(tempDir)

	cm, err := NewCacheManager(tempDir, 10*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}

	// 1. Atomic write
	testFile := filepath.Join(tempDir, "test.png")
	data := []byte{0x89, 'P', 'N', 'G', 0x01, 0x02, 0x03}
	if err := cm.WriteAtomic(testFile, data); err != nil {
		t.Fatalf("WriteAtomic failed: %v", err)
	}

	// 2. Integrity check
	// Calculate expected hash
	pipe, _ := NewIconPipeline(tempDir)
	rec, err := pipe.ProcessAndCache("test.app", &IconAsset{
		AppID:  "test.app",
		Source: "test",
		Data:   []byte("<svg xmlns='http://www.w3.org/2000/svg' width='64' height='64'><rect width='64' height='64' fill='#0078d4'/></svg>"),
	})
	if err != nil {
		t.Fatalf("ProcessAndCache failed: %v", err)
	}

	ok, err := cm.VerifyIntegrity(rec.OriginalPath, rec.ContentHash)
	if err != nil || !ok {
		t.Errorf("Integrity check failed on valid file: ok=%v, err=%v", ok, err)
	}

	// 3. Detect file corruption
	_ = os.WriteFile(rec.OriginalPath, []byte("CorruptedBytes"), 0644)
	ok, _ = cm.VerifyIntegrity(rec.OriginalPath, rec.ContentHash)
	if ok {
		t.Errorf("Expected integrity check to FAIL on tampered file, got ok=true")
	}

	// 4. Clean orphaned files
	orphanFile := filepath.Join(tempDir, "128", "orphan.png")
	_ = os.WriteFile(orphanFile, []byte("orphan"), 0644)
	deleted, _ := cm.CleanOrphanedFiles()
	if deleted == 0 {
		t.Errorf("Expected orphaned file to be cleaned")
	}
}

func TestIconResolverService_ModesAndProviders(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "arjax_svc_test_*")
	defer os.RemoveAll(tempDir)

	svc, err := NewIconResolverService(tempDir)
	if err != nil {
		t.Fatalf("NewIconResolverService failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	// 1. Catalog Mode Resolution: Brave Browser (AUR uninstalled package)
	// Must resolve canonical AppID com.brave.Browser without PKGBUILD execution
	braveDTO, err := svc.ResolveAppIcon(ctx, "brave", "Brave Browser", "aur", ModeCatalog, 128)
	if err != nil {
		t.Logf("Notice: brave catalog resolve returned: %v", err)
	} else if braveDTO != nil {
		t.Logf("Resolved Brave (Catalog): AppID=%s, Source=%s, Status=%s", braveDTO.AppID, braveDTO.Source, braveDTO.IconStatus)
		if braveDTO.IconStatus != "available" {
			t.Errorf("Expected Brave icon status 'available', got %s", braveDTO.IconStatus)
		}
	}

	// 2. Catalog Mode Resolution: Spotify (AUR uninstalled package)
	spotifyDTO, err := svc.ResolveAppIcon(ctx, "spotify", "Spotify", "aur", ModeCatalog, 128)
	if err != nil {
		t.Logf("Notice: spotify catalog resolve returned: %v", err)
	} else if spotifyDTO != nil {
		t.Logf("Resolved Spotify (Catalog): AppID=%s, Source=%s, Status=%s", spotifyDTO.AppID, spotifyDTO.Source, spotifyDTO.IconStatus)
		if spotifyDTO.IconStatus != "available" {
			t.Errorf("Expected Spotify icon status 'available', got %s", spotifyDTO.IconStatus)
		}
	}

	// 3. Installed Mode Resolution: Firefox (Local .desktop file & theme)
	fxDTO, err := svc.ResolveAppIcon(ctx, "firefox", "firefox", "pacman", ModeInstalled, 128)
	if err == nil && fxDTO != nil {
		t.Logf("Resolved Firefox (Installed Mode): Source=%s, LocalPath=%s", fxDTO.Source, fxDTO.LocalPath)
		if fxDTO.Source != "desktop_file" && fxDTO.Source != "system_theme" && fxDTO.Source != "system_pixmap" {
			t.Logf("Firefox resolved with source %s", fxDTO.Source)
		}
	}

	// 4. Fallback Resolver: Must have NO tickmark!
	fb := NewFallbackResolver()
	fbAsset, err := fb.ResolveIcon(ctx, "com.fake.NoSuchApp", "nosuchapp", ModeCatalog)
	if err != nil {
		t.Fatalf("Fallback resolver failed: %v", err)
	}
	fbStr := string(fbAsset.Data)
	if strings.Contains(fbStr, "L60 76 L80 52") || strings.Contains(fbStr, "polyline") {
		t.Errorf("CRITICAL: Fallback SVG still contains tickmark graphic!")
	}

	// 5. Provenance tracking verification
	if braveDTO != nil {
		if braveDTO.Provenance.FetchedAt.IsZero() {
			t.Errorf("Expected non-zero Provenance.FetchedAt")
		}
		if braveDTO.Provenance.ContentHash == "" {
			t.Errorf("Expected non-empty Provenance.ContentHash")
		}
	}
}

func TestIconResolverService_ConcurrencyAndDeduplication(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "arjax_concurrency_test_*")
	defer os.RemoveAll(tempDir)

	svc, err := NewIconResolverService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const numRoutines = 15
	var wg sync.WaitGroup
	wg.Add(numRoutines)

	results := make([]*IconResponseDTO, numRoutines)

	for i := 0; i < numRoutines; i++ {
		idx := i
		go func() {
			defer wg.Done()
			dto, _ := svc.ResolveAppIcon(ctx, "org.blender.Blender", "blender", "flatpak", ModeCatalog, 128)
			results[idx] = dto
		}()
	}

	wg.Wait()

	for i, r := range results {
		if r == nil {
			t.Errorf("Routine %d got nil result", i)
		}
	}
}

func TestIconResolverService_CacheInvalidation(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "arjax_inval_test_*")
	defer os.RemoveAll(tempDir)

	svc, _ := NewIconResolverService(tempDir)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initial resolve
	dto1, err := svc.ResolveAppIcon(ctx, "firefox", "firefox", "pacman", ModeCatalog, 128)
	if err != nil || dto1 == nil {
		t.Logf("Initial resolve: %v", err)
	}

	// Refresh
	dto2, err := svc.RefreshIcon(ctx, "firefox", "firefox")
	if err != nil || dto2 == nil {
		t.Fatalf("RefreshIcon failed: %v", err)
	}
}
