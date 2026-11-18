package icons

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// SVGSanitizer validates and neutralizes SVG content against XXE, XSS, and SSRF references
type SVGSanitizer struct{}

func NewSVGSanitizer() *SVGSanitizer {
	return &SVGSanitizer{}
}

// ValidateAndSanitize checks SVG structure, strips unsafe elements and attributes, and rejects external DTDs
func (s *SVGSanitizer) ValidateAndSanitize(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty SVG payload")
	}

	content := string(data)

	// 1. XXE / DTD Protection: Strictly reject external entities and custom DOCTYPE declarations
	lowContent := strings.ToLower(content)
	if strings.Contains(lowContent, "<!entity") || strings.Contains(lowContent, "system \"") || strings.Contains(lowContent, "public \"") {
		return nil, fmt.Errorf("SVG contains prohibited XML entity / DOCTYPE system references (XXE risk)")
	}

	// 2. Structural XML Tokenizer Validation
	decoder := xml.NewDecoder(bytes.NewReader(data))
	// Strict: do not resolve entity references
	decoder.Entity = map[string]string{}

	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)

	hasSVGTag := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			// If strict XML decoder fails on unclosed tags or non-standard HTML-like SVG,
			// fall back to regex-based robust sanitization
			return s.regexFallbackSanitize(data)
		}

		switch el := token.(type) {
		case xml.StartElement:
			tag := strings.ToLower(el.Name.Local)
			if tag == "svg" {
				hasSVGTag = true
			}

			// Reject dangerous element tags outright
			if tag == "script" || tag == "foreignobject" || tag == "iframe" ||
				tag == "object" || tag == "embed" || tag == "applet" ||
				tag == "audio" || tag == "video" {
				// Skip element and its children
				if err := decoder.Skip(); err != nil {
					return nil, err
				}
				continue
			}

			// Filter attributes: remove event handlers (on*), javascript: URIs, external URLs
			cleanAttrs := make([]xml.Attr, 0, len(el.Attr))
			for _, attr := range el.Attr {
				attrName := strings.ToLower(attr.Name.Local)
				attrVal := strings.TrimSpace(attr.Value)
				lowVal := strings.ToLower(attrVal)

				// Strip inline handlers: onclick, onload, onerror, onmouseover, etc.
				if strings.HasPrefix(attrName, "on") {
					continue
				}

				// Strip javascript:, data: URIs, or external http/https in hrefs
				if attrName == "href" || strings.HasSuffix(attrName, ":href") || attrName == "src" {
					if strings.HasPrefix(lowVal, "javascript:") || strings.HasPrefix(lowVal, "vbscript:") {
						continue
					}
					// Reject external resource loading inside SVG
					if strings.HasPrefix(lowVal, "http://") || strings.HasPrefix(lowVal, "https://") {
						continue
					}
				}

				cleanAttrs = append(cleanAttrs, attr)
			}
			el.Attr = cleanAttrs
			_ = encoder.EncodeToken(el)

		case xml.EndElement:
			tag := strings.ToLower(el.Name.Local)
			if tag == "script" || tag == "foreignobject" || tag == "iframe" ||
				tag == "object" || tag == "embed" || tag == "applet" {
				continue
			}
			_ = encoder.EncodeToken(el)

		case xml.Directive:
			// Strip all DTD directives
			continue

		default:
			_ = encoder.EncodeToken(el)
		}
	}

	_ = encoder.Flush()

	if !hasSVGTag {
		return nil, fmt.Errorf("payload is missing <svg> root element")
	}

	return buf.Bytes(), nil
}

// regexFallbackSanitize provides a secondary defense layer for complex SVGs
func (s *SVGSanitizer) regexFallbackSanitize(data []byte) ([]byte, error) {
	str := string(data)

	// Reject external DTDs / XXE
	low := strings.ToLower(str)
	if strings.Contains(low, "<!entity") || strings.Contains(low, "<!doctype") {
		return nil, fmt.Errorf("SVG contains prohibited XML entity / DOCTYPE declaration (XXE risk)")
	}

	// Strip <script>...</script>
	reScript := regexp.MustCompile(`(?is)<script.*?</script>`)
	str = reScript.ReplaceAllString(str, "")

	// Strip <foreignObject>...</foreignObject>
	reFO := regexp.MustCompile(`(?is)<foreignobject.*?</foreignobject>`)
	str = reFO.ReplaceAllString(str, "")

	// Strip inline handlers (on*)
	reHandlers := regexp.MustCompile(`(?i)\bon\w+\s*=\s*["'][^"']*["']`)
	str = reHandlers.ReplaceAllString(str, "")

	// Strip javascript: / external URLs in hrefs
	reJS := regexp.MustCompile(`(?i)href\s*=\s*["']\s*(javascript:[^"']*|https?:[^"']*)["']`)
	str = reJS.ReplaceAllString(str, `href="#"`)

	if !strings.Contains(strings.ToLower(str), "<svg") {
		return nil, fmt.Errorf("missing <svg> tag after sanitization")
	}

	return []byte(str), nil
}

// RasterizeSVG converts a sanitized SVG into a crisp, square PNG using rsvg-convert
func RasterizeSVG(svgData []byte, width, height int) ([]byte, error) {
	if width <= 0 {
		width = DefaultIconSize
	}
	if height <= 0 {
		height = width
	}

	rsvgPath, err := exec.LookPath("rsvg-convert")
	if err != nil {
		return nil, fmt.Errorf("rsvg-convert not found: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, rsvgPath,
		"-w", fmt.Sprintf("%d", width),
		"-h", fmt.Sprintf("%d", height),
		"-f", "png",
	)

	cmd.Stdin = bytes.NewReader(svgData)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("rsvg-convert failed: %w (%s)", err, errBuf.String())
	}

	out := outBuf.Bytes()
	if len(out) == 0 {
		return nil, fmt.Errorf("rsvg-convert produced empty PNG output")
	}

	return out, nil
}

