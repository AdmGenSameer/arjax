package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
)

type CacheEntry struct {
	Timestamp time.Time        `json:"timestamp"`
	Packages  []models.Package `json:"packages"`
}

type ProviderCache struct {
	mu         sync.RWMutex
	searchData map[string]CacheEntry
	filePath   string
	ttl        time.Duration
}

func NewProviderCache(ttl time.Duration) *ProviderCache {
	cacheDir := os.ExpandEnv("$HOME/.cache/arjax")
	_ = os.MkdirAll(cacheDir, 0755)
	filePath := filepath.Join(cacheDir, "provider_cache.json")

	c := &ProviderCache{
		searchData: make(map[string]CacheEntry),
		filePath:   filePath,
		ttl:        ttl,
	}
	c.load()
	return c
}

func (c *ProviderCache) GetSearch(key string) ([]models.Package, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.searchData[key]
	if !ok {
		return nil, false
	}
	if time.Since(entry.Timestamp) > c.ttl {
		return nil, false
	}
	return entry.Packages, true
}

func (c *ProviderCache) SetSearch(key string, packages []models.Package) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.searchData[key] = CacheEntry{
		Timestamp: time.Now(),
		Packages:  packages,
	}
	_ = c.save()
}

func (c *ProviderCache) load() {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &c.searchData)
}

func (c *ProviderCache) save() error {
	data, err := json.MarshalIndent(c.searchData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.filePath, data, 0644)
}
