package icons

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxIconSizeBytes = 5 * 1024 * 1024 // 5 MB max
	DefaultIconSize  = 128
)

// Standard icon sizes for XDG caching
var StandardIconSizes = []int{64, 128, 256}

// IconPipeline handles validation, security checks, hashing, normalization, and caching
type IconPipeline struct {
	baseCacheDir string
	originalDir  string
	sizesDirs    map[int]string
	cacheMgr     *CacheManager
	sanitizer    *SVGSanitizer
}

// NewIconPipeline initializes the pipeline and sets up XDG cache directories
func NewIconPipeline(customCacheDir string) (*IconPipeline, error) {
	cacheBase := customCacheDir
	if cacheBase == "" {
		xdg := os.Getenv("XDG_CACHE_HOME")
		if xdg == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			xdg = filepath.Join(home, ".cache")
		}
		cacheBase = filepath.Join(xdg, "arjax", "icons")
	}

	origDir := filepath.Join(cacheBase, "original")
	sizesDirs := make(map[int]string)
	for _, sz := range StandardIconSizes {
		sizesDirs[sz] = filepath.Join(cacheBase, fmt.Sprintf("%d", sz))
	}

	// Ensure directories exist
	if err := os.MkdirAll(origDir, 0755); err != nil {
		return nil, err
	}
	for _, dir := range sizesDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	_ = os.MkdirAll(filepath.Join(cacheBase, "metadata"), 0755)

	cacheMgr, err := NewCacheManager(cacheBase, DefaultMaxCacheSize)
	if err != nil {
		return nil, err
	}

	return &IconPipeline{
		baseCacheDir: cacheBase,
		originalDir:  origDir,
		sizesDirs:    sizesDirs,
		cacheMgr:     cacheMgr,
		sanitizer:    NewSVGSanitizer(),
	}, nil
}

// ValidateURL inspects remote URLs to enforce HTTPS, block private IPs, and check DNS
func (p *IconPipeline) ValidateURL(remoteURL string) error {
	return ValidateRemoteURL(remoteURL)
}

// DetectFormat detects MIME type and validates file magic bytes
func (p *IconPipeline) DetectFormat(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty icon payload")
	}
	if len(data) > MaxIconSizeBytes {
		return "", fmt.Errorf("icon payload exceeds size limit of %d bytes", MaxIconSizeBytes)
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "png", nil
	}

	// WebP: RIFF .... WEBP
	if len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
		return "webp", nil
	}

	// JPEG: FF D8 FF
	if len(data) >= 3 && bytes.Equal(data[:3], []byte{0xFF, 0xD8, 0xFF}) {
		return "jpeg", nil
	}

	// SVG: strict text / tag check
	sample := strings.ToLower(string(data[:min(len(data), 512)]))
	if strings.Contains(sample, "<svg") || strings.Contains(sample, "<?xml") {
		return "svg", nil
	}

	return "", fmt.Errorf("unrecognized or unsupported image file signature")
}

// SanitizeSVG strips unsafe tags and attributes via maintained SVG parser
func (p *IconPipeline) SanitizeSVG(data []byte) []byte {
	sanitized, err := p.sanitizer.ValidateAndSanitize(data)
	if err == nil {
		return sanitized
	}
	fb, err := p.sanitizer.regexFallbackSanitize(data)
	if err == nil {
		return fb
	}
	return data
}

// ProcessAndCache normalizes the icon asset, calculates content hash, creates size variants, and saves to cache
func (p *IconPipeline) ProcessAndCache(appID string, asset *IconAsset) (*IconRecord, error) {
	format, err := p.DetectFormat(asset.Data)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// SVG security & sanitization
	if format == "svg" {
		sanitized, err := p.sanitizer.ValidateAndSanitize(asset.Data)
		if err != nil {
			return nil, fmt.Errorf("svg security validation failed: %w", err)
		}
		asset.Data = sanitized
	}

	// Calculate content hash (SHA-256)
	hasher := sha256.New()
	hasher.Write(asset.Data)
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	origExt := format
	if origExt == "jpeg" {
		origExt = "jpg"
	}
	origFilename := fmt.Sprintf("sha256-%s.%s", contentHash, origExt)
	origPath := filepath.Join(p.originalDir, origFilename)

	// Save original asset atomically
	if _, err := os.Stat(origPath); os.IsNotExist(err) {
		_ = p.cacheMgr.WriteAtomic(origPath, asset.Data)
	}

	variants := make(map[int]string)

	if format == "svg" {
		// Attempt to rasterize SVG into crisp, square PNGs before frontend display
		rasterizedSuccess := false
		for _, sz := range StandardIconSizes {
			pngData, err := RasterizeSVG(asset.Data, sz, sz)
			if err == nil && len(pngData) > 0 {
				targetPath := filepath.Join(p.sizesDirs[sz], fmt.Sprintf("%s.png", appID))
				if err := p.cacheMgr.WriteAtomic(targetPath, pngData); err == nil {
					variants[sz] = targetPath
					rasterizedSuccess = true
				}
			}
		}

		// Fallback: If rsvg-convert is unavailable or failed, store sanitized SVG
		if !rasterizedSuccess {
			for _, sz := range StandardIconSizes {
				targetPath := filepath.Join(p.sizesDirs[sz], fmt.Sprintf("%s.svg", appID))
				_ = p.cacheMgr.WriteAtomic(targetPath, asset.Data)
				variants[sz] = targetPath
			}
		}
		asset.Width = 128
		asset.Height = 128
	} else {
		// Raster: Decode and generate square, alpha-preserved resized PNGs
		srcImg, _, err := image.Decode(bytes.NewReader(asset.Data))
		if err != nil {
			return nil, fmt.Errorf("failed to decode raster image: %w", err)
		}

		b := srcImg.Bounds()
		asset.Width = b.Dx()
		asset.Height = b.Dy()

		for _, sz := range StandardIconSizes {
			resized := p.resizeSquare(srcImg, sz)
			targetPath := filepath.Join(p.sizesDirs[sz], fmt.Sprintf("%s.png", appID))

			var pngBuf bytes.Buffer
			if err := png.Encode(&pngBuf, resized); err == nil {
				_ = p.cacheMgr.WriteAtomic(targetPath, pngBuf.Bytes())
				variants[sz] = targetPath
			}
		}
	}

	prov := IconProvenance{
		Source:           asset.Source,
		Repository:       asset.Repository,
		OriginalURL:      asset.OriginalURL,
		LocalOriginPath:  asset.LocalPath,
		ContentHash:      contentHash,
		FetchedAt:        time.Now(),
		ValidatedAt:      time.Now(),
		ValidationStatus: "validated",
	}

	rec := &IconRecord{
		AppID:          appID,
		PackageSource:  asset.Source,
		Repository:     asset.Repository,
		OriginalURL:    asset.OriginalURL,
		OriginalFormat: format,
		ContentHash:    contentHash,
		OriginalPath:   origPath,
		Variants:       variants,
		Width:          asset.Width,
		Height:         asset.Height,
		Status:         "cached",
		LastChecked:    time.Now(),
		LastAccessed:   time.Now(),
		Provenance:     prov,
	}

	_ = p.cacheMgr.Put(rec)
	return rec, nil
}

// resizeSquare fits the image into a targetSize x targetSize square without distortion, preserving transparency
func (p *IconPipeline) resizeSquare(src image.Image, targetSize int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, targetSize, targetSize))
	// Clear to fully transparent
	draw.Draw(dst, dst.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if srcW == 0 || srcH == 0 {
		return dst
	}

	// Determine scaling factor to maintain aspect ratio
	scale := float64(targetSize) / float64(max(srcW, srcH))
	newW := int(float64(srcW) * scale)
	newH := int(float64(srcH) * scale)

	// Center in target square
	offsetX := (targetSize - newW) / 2
	offsetY := (targetSize - newH) / 2

	for y := 0; y < newH; y++ {
		srcY := int(float64(y) / scale)
		for x := 0; x < newW; x++ {
			srcX := int(float64(x) / scale)
			dst.Set(x+offsetX, y+offsetY, src.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY))
		}
	}

	return dst
}

// ToDataURI converts raw bytes or a cached file to a base64 Data URI for instant frontend display
func (p *IconPipeline) ToDataURI(data []byte, format string) string {
	mime := "image/png"
	switch format {
	case "svg":
		mime = "image/svg+xml"
	case "webp":
		mime = "image/webp"
	case "jpeg", "jpg":
		mime = "image/jpeg"
	}
	return fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(data))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
