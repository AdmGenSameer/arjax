package appstream

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// AppstreamMetadata represents full application metadata with screenshots, developer, and homepage
type AppstreamMetadata struct {
	ID          string    `json:"id"`
	FlathubID   string    `json:"flathubId"`
	Name        string    `json:"name"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Developer   string    `json:"developer"`
	Homepage    string    `json:"homepage"`
	License     string    `json:"license"`
	Icon        string    `json:"icon"`
	Categories  []string  `json:"categories"`
	Screenshots []string  `json:"screenshots"`
	Releases    []string  `json:"releases"`
	CachedAt    time.Time `json:"cachedAt"`
}

// AppSummary represents compact search hit information
type AppSummary struct {
	ID        string `json:"id"`
	FlathubID string `json:"flathubId"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Icon      string `json:"icon"`
}

// Crawler manages dynamic querying, local caching, and offline XML parsing of 3,380+ apps
type Crawler struct {
	client    *http.Client
	mu        sync.RWMutex
	cache     map[string]*AppstreamMetadata
	cachePath string
	ttl       time.Duration
	localXML  string
}

// NewCrawler initializes an AppStream crawler instance with connection pooling and disk cache
func NewCrawler() *Crawler {
	cacheDir := os.ExpandEnv("$HOME/.cache/arjax")
	_ = os.MkdirAll(cacheDir, 0755)
	cacheFile := filepath.Join(cacheDir, "appstream_crawler_cache.json")

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// Detect local system appstream XML if available
	xmlCandidates := []string{
		"/var/lib/flatpak/appstream/flathub/x86_64/active/appstream.xml",
		os.ExpandEnv("$HOME/.local/share/flatpak/appstream/flathub/x86_64/active/appstream.xml"),
		"/usr/share/app-info/xmls/flathub.xml",
	}

	localXML := ""
	for _, p := range xmlCandidates {
		if _, err := os.Stat(p); err == nil {
			localXML = p
			break
		}
	}

	c := &Crawler{
		client:    client,
		cache:     make(map[string]*AppstreamMetadata),
		cachePath: cacheFile,
		ttl:       7 * 24 * time.Hour,
		localXML:  localXML,
	}

	c.loadDiskCache()
	return c
}

// NormalizeID cleans user queries and maps common package aliases to reverse-DNS Flathub IDs
func (c *Crawler) NormalizeID(input string) (string, string) {
	trimmed := strings.ToLower(strings.TrimSpace(input))

	aliasMap := map[string]string{
		"steam":                  "com.valvesoftware.Steam",
		"vlc":                    "org.videolan.VLC",
		"firefox":                "org.mozilla.firefox",
		"firefox-bin":            "org.mozilla.firefox",
		"chrome":                 "com.google.Chrome",
		"google-chrome":          "com.google.Chrome",
		"spotify":                "com.spotify.Client",
		"discord":                "com.discordapp.Discord",
		"blender":                "org.blender.Blender",
		"obs-studio":             "com.obsproject.Studio",
		"obs":                    "com.obsproject.Studio",
		"gimp":                   "org.gimp.GIMP",
		"kicad":                  "org.kicad.KiCad",
		"bottles":                "com.usebottles.bottles",
		"flatseal":               "com.github.tchx84.Flatseal",
		"telegram":               "org.telegram.desktop",
		"telegram-desktop":       "org.telegram.desktop",
		"darktable":              "org.darktable.Darktable",
		"audacity":               "org.audacityteam.Audacity",
		"inkscape":               "org.inkscape.Inkscape",
		"krita":                  "org.kde.krita",
		"kdenlive":               "org.kde.kdenlive",
		"obsidian":               "md.obsidian.Obsidian",
		"bitwarden":              "com.bitwarden.desktop",
		"brave":                  "com.brave.Browser",
		"brave-bin":              "com.brave.Browser",
		"code":                   "com.visualstudio.code",
		"visual-studio-code-bin": "com.visualstudio.code",
		"vscode":                 "com.visualstudio.code",
		"heroic":                 "com.heroicgameslauncher.hgl",
		"lutris":                 "net.lutris.Lutris",
		"retroarch":              "org.libretro.RetroArch",
		"libreoffice":            "org.libreoffice.LibreOffice",
		"onlyoffice":             "org.onlyoffice.desktopeditors",
		"onlyoffice-bin":         "org.onlyoffice.desktopeditors",
		"handbrake":              "fr.handbrake.ghb",
		"thunderbird":            "org.mozilla.Thunderbird",
		"sober":                  "org.vinegarhq.Sober",
		"shotcut":                "org.shotcut.Shotcut",
		"dbeaver":                "io.dbeaver.DBeaverCommunity",
		"postman":                "com.getpostman.Postman",
		"insomnia":               "rest.insomnia.Insomnia",
		"sublime-text":           "com.sublimetext.three",
		"alacritty":              "com.alacritty.Alacritty",
		"kitty":                  "io.github.kovidgoyal.kitty",
	}

	flathubID := trimmed
	if mapped, ok := aliasMap[trimmed]; ok {
		flathubID = mapped
	}

	shortID := trimmed
	if strings.Contains(flathubID, ".") {
		parts := strings.Split(flathubID, ".")
		shortID = strings.ToLower(parts[len(parts)-1])
	}

	return shortID, flathubID
}

// GetAppstream fetches rich metadata for an app with 3-tier lookup: memory -> disk cache -> network/XML
func (c *Crawler) GetAppstream(ctx context.Context, inputId string) (*AppstreamMetadata, error) {
	shortID, flathubID := c.NormalizeID(inputId)

	// 1. Check in-memory cache
	c.mu.RLock()
	if cached, ok := c.cache[shortID]; ok && time.Since(cached.CachedAt) < c.ttl {
		c.mu.RUnlock()
		return cached, nil
	}
	if cached, ok := c.cache[flathubID]; ok && time.Since(cached.CachedAt) < c.ttl {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	// 2. Check local system AppStream XML (fast, offline, 3,380+ apps)
	if c.localXML != "" {
		meta, err := c.parseLocalXML(flathubID, shortID)
		if err == nil && meta != nil {
			c.storeCache(shortID, flathubID, meta)
			return meta, nil
		}
	}

	// 3. Query Flathub REST API v2
	targetID := flathubID
	if !strings.Contains(targetID, ".") {
		// Discover full reverse-DNS ID via quick Flathub search
		searchHits, err := c.SearchApps(ctx, shortID, 1)
		if err == nil && len(searchHits) > 0 && searchHits[0].FlathubID != "" {
			targetID = searchHits[0].FlathubID
		}
	}

	if strings.Contains(targetID, ".") {
		meta, err := c.fetchFromFlathubAPI(ctx, targetID, shortID)
		if err == nil && meta != nil {
			c.storeCache(shortID, flathubID, meta)
			c.storeCache(shortID, targetID, meta)
			return meta, nil
		}
	}

	return nil, fmt.Errorf("appstream metadata not found for %s (%s)", inputId, flathubID)
}

func (c *Crawler) fetchFromFlathubAPI(ctx context.Context, flathubID, shortID string) (*AppstreamMetadata, error) {
	reqURL := fmt.Sprintf("https://flathub.org/api/v2/appstream/%s", flathubID)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Arjax-Store/2.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("flathub API returned status: %d", resp.StatusCode)
	}

	var data struct {
		Name           string `json:"name"`
		Summary        string `json:"summary"`
		DeveloperName  string `json:"developer_name"`
		ProjectLicense string `json:"project_license"`
		Description    string `json:"description"`
		Icon           string `json:"icon"`
		Urls           struct {
			Homepage string `json:"homepage"`
		} `json:"urls"`
		Categories  []string `json:"categories"`
		Screenshots []struct {
			Sizes []struct {
				Src   string `json:"src"`
				Width string `json:"width"`
			} `json:"sizes"`
		} `json:"screenshots"`
		Releases []struct {
			Version     string `json:"version"`
			Description string `json:"description"`
		} `json:"releases"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	meta := &AppstreamMetadata{
		ID:          shortID,
		FlathubID:   flathubID,
		Name:        data.Name,
		Summary:     data.Summary,
		Description: data.Description,
		Developer:   data.DeveloperName,
		Homepage:    data.Urls.Homepage,
		License:     data.ProjectLicense,
		Icon:        data.Icon,
		Categories:  data.Categories,
		CachedAt:    time.Now(),
		Screenshots: []string{},
		Releases:    []string{},
	}

	if meta.Homepage == "" {
		meta.Homepage = fmt.Sprintf("https://flathub.org/apps/%s", flathubID)
	}
	if meta.Developer == "" {
		meta.Developer = "Verified Linux Developer"
	}
	if meta.Icon == "" {
		meta.Icon = fmt.Sprintf("https://dl.flathub.org/media/icons/128x128/%s.png", flathubID)
	}

	for _, sc := range data.Screenshots {
		bestSrc := ""
		maxWidth := 0
		for _, sz := range sc.Sizes {
			var w int
			fmt.Sscanf(sz.Width, "%d", &w)
			if w > maxWidth {
				maxWidth = w
				bestSrc = sz.Src
			}
		}
		if bestSrc == "" && len(sc.Sizes) > 0 {
			bestSrc = sc.Sizes[0].Src
		}
		if bestSrc != "" {
			meta.Screenshots = append(meta.Screenshots, bestSrc)
		}
	}

	for i, rel := range data.Releases {
		if i >= 4 {
			break
		}
		if rel.Version != "" {
			meta.Releases = append(meta.Releases, fmt.Sprintf("v%s: %s", rel.Version, rel.Description))
		}
	}

	return meta, nil
}

// SearchApps queries Flathub's POST search endpoint to retrieve matching desktop applications
func (c *Crawler) SearchApps(ctx context.Context, query string, limit int) ([]AppSummary, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return []AppSummary{}, nil
	}

	payload, err := json.Marshal(map[string]interface{}{
		"query": trimmed,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://flathub.org/api/v2/search", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Arjax-Store/2.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status: %d", resp.StatusCode)
	}

	var searchResp struct {
		Hits []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Icon    string `json:"icon"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	if limit <= 0 || limit > len(searchResp.Hits) {
		limit = len(searchResp.Hits)
	}

	summaries := make([]AppSummary, 0, limit)
	for i := 0; i < limit; i++ {
		hit := searchResp.Hits[i]
		cleanID := strings.ReplaceAll(hit.ID, "_", ".")
		shortID, _ := c.NormalizeID(cleanID)
		summaries = append(summaries, AppSummary{
			ID:        shortID,
			FlathubID: cleanID,
			Name:      hit.Name,
			Summary:   hit.Summary,
			Icon:      hit.Icon,
		})
	}

	return summaries, nil
}

// parseLocalXML reads metadata from local system appstream.xml offline with full image & homepage accuracy
func (c *Crawler) parseLocalXML(flathubID, shortID string) (*AppstreamMetadata, error) {
	file, err := os.Open(c.localXML)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	type XMLURL struct {
		Type  string `xml:"type,attr"`
		Value string `xml:",chardata"`
	}
	type XMLImage struct {
		Type   string `xml:"type,attr"`
		Width  string `xml:"width,attr"`
		Height string `xml:"height,attr"`
		URL    string `xml:",chardata"`
	}
	type XMLScreenshot struct {
		Images []XMLImage `xml:"image"`
	}
	type XMLIcon struct {
		Type   string `xml:"type,attr"`
		Width  string `xml:"width,attr"`
		Height string `xml:"height,attr"`
		Value  string `xml:",chardata"`
	}
	type XMLText struct {
		Lang  string `xml:"lang,attr"`
		Value string `xml:",chardata"`
	}

	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}

		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "component" {
				var comp struct {
					ID          string          `xml:"id"`
					Names       []XMLText       `xml:"name"`
					Summaries   []XMLText       `xml:"summary"`
					License     string          `xml:"project_license"`
					Developer   string          `xml:"developer_name"`
					URLs        []XMLURL        `xml:"url"`
					Screenshots []XMLScreenshot `xml:"screenshots>screenshot"`
					Icons       []XMLIcon       `xml:"icon"`
				}

				if err := decoder.DecodeElement(&comp, &se); err == nil {
					cleanCompID := strings.TrimSuffix(comp.ID, ".desktop")
					isMatch := strings.EqualFold(comp.ID, flathubID) ||
						strings.EqualFold(cleanCompID, flathubID) ||
						strings.EqualFold(cleanCompID, shortID) ||
						strings.HasSuffix(strings.ToLower(cleanCompID), "."+shortID)

					if isMatch {
						// 1. Clean English Name
						name := ""
						for _, n := range comp.Names {
							if n.Lang == "" || n.Lang == "en" || n.Lang == "C" {
								name = strings.TrimSpace(n.Value)
								break
							}
						}
						if name == "" && len(comp.Names) > 0 {
							name = strings.TrimSpace(comp.Names[0].Value)
						}

						// 2. Clean English Summary
						summary := ""
						for _, s := range comp.Summaries {
							if s.Lang == "" || s.Lang == "en" || s.Lang == "C" {
								summary = strings.TrimSpace(s.Value)
								break
							}
						}
						if summary == "" && len(comp.Summaries) > 0 {
							summary = strings.TrimSpace(comp.Summaries[0].Value)
						}

						// 3. Homepage URL
						homepage := ""
						for _, u := range comp.URLs {
							val := strings.TrimSpace(u.Value)
							if u.Type == "homepage" && strings.HasPrefix(val, "http") {
								homepage = val
								break
							}
						}
						if homepage == "" {
							for _, u := range comp.URLs {
								val := strings.TrimSpace(u.Value)
								if strings.HasPrefix(val, "http") && !strings.Contains(u.Type, "bug") && !strings.Contains(u.Type, "translate") {
									homepage = val
									break
								}
							}
						}
						if homepage == "" {
							homepage = fmt.Sprintf("https://flathub.org/apps/%s", comp.ID)
						}

						// 4. Remote 128x128 Icon URL
						iconURL := ""
						for _, ic := range comp.Icons {
							val := strings.TrimSpace(ic.Value)
							if ic.Type == "remote" && strings.HasPrefix(val, "http") {
								if ic.Width == "128" {
									iconURL = val
									break
								} else if iconURL == "" {
									iconURL = val
								}
							}
						}
						if iconURL == "" {
							iconURL = fmt.Sprintf("https://dl.flathub.org/media/icons/128x128/%s.png", comp.ID)
						}

						// 5. High-Resolution Screenshots
						screenshotList := make([]string, 0, len(comp.Screenshots))
						for _, sc := range comp.Screenshots {
							bestImg := ""
							for _, img := range sc.Images {
								u := strings.TrimSpace(img.URL)
								if img.Type == "source" || strings.Contains(u, "_orig") {
									bestImg = u
									break
								}
							}
							if bestImg == "" {
								maxW := 0
								for _, img := range sc.Images {
									u := strings.TrimSpace(img.URL)
									var w int
									fmt.Sscanf(img.Width, "%d", &w)
									if w > maxW && strings.HasPrefix(u, "http") {
										maxW = w
										bestImg = u
									}
								}
							}
							if bestImg == "" && len(sc.Images) > 0 && strings.HasPrefix(strings.TrimSpace(sc.Images[0].URL), "http") {
								bestImg = strings.TrimSpace(sc.Images[0].URL)
							}
							if bestImg != "" {
								screenshotList = append(screenshotList, bestImg)
							}
						}

						return &AppstreamMetadata{
							ID:          shortID,
							FlathubID:   comp.ID,
							Name:        name,
							Summary:     summary,
							Developer:   comp.Developer,
							Homepage:    homepage,
							Icon:        iconURL,
							License:     comp.License,
							Screenshots: screenshotList,
							CachedAt:    time.Now(),
						}, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("not found in local XML")
}

func (c *Crawler) storeCache(shortID, flathubID string, meta *AppstreamMetadata) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[shortID] = meta
	c.cache[flathubID] = meta
	c.saveDiskCache()
}

func (c *Crawler) loadDiskCache() {
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	_ = json.Unmarshal(data, &c.cache)
}

func (c *Crawler) saveDiskCache() {
	data, err := json.MarshalIndent(c.cache, "", "  ")
	if err != nil {
		return
	}

	tmpFile := c.cachePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err == nil {
		_ = os.Rename(tmpFile, c.cachePath)
	}
}

// BackgroundCrawlPopular warms the cache with the top popular apps in the background
func (c *Crawler) BackgroundCrawlPopular(ctx context.Context, count int) {
	go func() {
		utils.LogInfo("Starting background AppStream crawler for top %d apps...", count)
		reqURL := fmt.Sprintf("https://flathub.org/api/v2/collection/popular?page=1&per_page=%d", count)
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return
		}
		req.Header.Set("User-Agent", "Arjax-Store/2.0")

		resp, err := c.client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			return
		}
		defer resp.Body.Close()

		var popResp struct {
			Hits []struct {
				ID string `json:"id"`
			} `json:"hits"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&popResp); err != nil {
			return
		}

		// Worker pool concurrency of 6 workers
		jobs := make(chan string, len(popResp.Hits))
		var wg sync.WaitGroup
		workers := 6

		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for id := range jobs {
					cleanID := strings.ReplaceAll(id, "_", ".")
					_, _ = c.GetAppstream(ctx, cleanID)
					time.Sleep(80 * time.Millisecond) // Respectful rate limit
				}
			}()
		}

		for _, hit := range popResp.Hits {
			jobs <- hit.ID
		}
		close(jobs)
		wg.Wait()
		utils.LogInfo("Completed background AppStream cache warming.")
	}()
}
