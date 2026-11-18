package icons

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	DefaultMaxCacheSize = 250 * 1024 * 1024 // 250 MB
	DefaultMaxEntries   = 5000
	DefaultIconTTL      = 30 * 24 * time.Hour // 30 days
	DefaultSoftTTL      = 7 * 24 * time.Hour  // 7 days (triggers background refresh)
	DefaultNegativeTTL  = 24 * time.Hour      // 24 hours (negative caching for missing icons)
)

// SingleFlight manages deduplication of in-flight icon resolution requests
type SingleFlight struct {
	mu    sync.Mutex
	calls map[string]*call
}

type call struct {
	wg  sync.WaitGroup
	val *IconResponseDTO
	err error
}

func NewSingleFlight() *SingleFlight {
	return &SingleFlight{calls: make(map[string]*call)}
}

func (g *SingleFlight) Do(key string, fn func() (*IconResponseDTO, error)) (*IconResponseDTO, error) {
	g.mu.Lock()
	if c, ok := g.calls[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.calls[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()

	return c.val, c.err
}

// CacheManager provides LRU caching, atomic writes, size limits, negative caching, and integrity verification
type CacheManager struct {
	mu           sync.RWMutex
	baseDir      string
	catalogPath  string
	maxSizeBytes int64
	maxEntries   int
	records      map[string]*IconRecord
	flight       *SingleFlight
}

func NewCacheManager(baseCacheDir string, maxSizeBytes int64) (*CacheManager, error) {
	if maxSizeBytes <= 0 {
		maxSizeBytes = DefaultMaxCacheSize
	}

	catalogPath := filepath.Join(baseCacheDir, "catalog.json")
	cm := &CacheManager{
		baseDir:      baseCacheDir,
		catalogPath:  catalogPath,
		maxSizeBytes: maxSizeBytes,
		maxEntries:   DefaultMaxEntries,
		records:      make(map[string]*IconRecord),
		flight:       NewSingleFlight(),
	}

	cm.loadCatalog()
	return cm, nil
}

// Get retrieves an icon record, updating its access time for LRU
func (cm *CacheManager) Get(appID string) (rec *IconRecord, exists bool, needsRefresh bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	r, found := cm.records[appID]
	if !found || r == nil {
		return nil, false, false
	}

	r.LastAccessed = time.Now()

	// Check if negative cache expired
	if r.IsNegative {
		if time.Since(r.LastChecked) > DefaultNegativeTTL {
			delete(cm.records, appID)
			return nil, false, false
		}
		return r, true, false
	}

	// Check if hard TTL expired
	if time.Since(r.LastChecked) > DefaultIconTTL {
		return r, true, true // Stale, needs refresh
	}

	// Check soft TTL for background refresh
	if time.Since(r.LastChecked) > DefaultSoftTTL {
		return r, true, true
	}

	return r, true, false
}

// Put records or updates an icon entry and enforces LRU limits
func (cm *CacheManager) Put(rec *IconRecord) error {
	if rec == nil || rec.AppID == "" {
		return fmt.Errorf("invalid icon record")
	}

	rec.LastChecked = time.Now()
	rec.LastAccessed = time.Now()

	cm.mu.Lock()
	cm.records[rec.AppID] = rec
	cm.saveCatalogLocked()
	cm.mu.Unlock()

	// Enforce LRU eviction asynchronously
	go cm.evictLRUIfNeeded()

	return nil
}

// PutNegative stores a negative cache entry to avoid querying non-existent icons repeatedly
func (cm *CacheManager) PutNegative(appID, source string) {
	rec := &IconRecord{
		AppID:         appID,
		PackageSource: source,
		Status:        "fallback",
		IsNegative:    true,
		LastChecked:   time.Now(),
		LastAccessed:  time.Now(),
		Variants:      make(map[int]string),
	}

	cm.mu.Lock()
	cm.records[appID] = rec
	cm.saveCatalogLocked()
	cm.mu.Unlock()
}

// Invalidate removes an app's icon from memory and disk
func (cm *CacheManager) Invalidate(appID string) {
	cm.mu.Lock()
	rec, found := cm.records[appID]
	if found && rec != nil {
		// Remove physical files
		if rec.OriginalPath != "" {
			_ = os.Remove(rec.OriginalPath)
		}
		for _, v := range rec.Variants {
			_ = os.Remove(v)
		}
		delete(cm.records, appID)
		cm.saveCatalogLocked()
	}
	cm.mu.Unlock()
}

// WriteAtomic writes data to a temporary file in the same directory and renames atomically
func (cm *CacheManager) WriteAtomic(targetPath string, data []byte) error {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName) // safe cleanup if rename fails
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, targetPath); err != nil {
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	return nil
}

// VerifyIntegrity checks if the file at path matches the recorded SHA-256 hash
func (cm *CacheManager) VerifyIntegrity(filePath, expectedHash string) (bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return false, err
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	return strings.EqualFold(actualHash, expectedHash), nil
}

// CleanOrphanedFiles deletes files in cache directories that are not referenced in the catalog
func (cm *CacheManager) CleanOrphanedFiles() (deleted int, err error) {
	cm.mu.RLock()
	referencedFiles := make(map[string]bool)
	for _, rec := range cm.records {
		if rec.OriginalPath != "" {
			referencedFiles[filepath.Clean(rec.OriginalPath)] = true
		}
		for _, v := range rec.Variants {
			if v != "" {
				referencedFiles[filepath.Clean(v)] = true
			}
		}
	}
	cm.mu.RUnlock()

	dirs := []string{"64", "128", "256", "original"}
	for _, sub := range dirs {
		subDir := filepath.Join(cm.baseDir, sub)
		entries, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fullPath := filepath.Clean(filepath.Join(subDir, e.Name()))
			if !referencedFiles[fullPath] {
				_ = os.Remove(fullPath)
				deleted++
			}
		}
	}

	return deleted, nil
}

func (cm *CacheManager) evictLRUIfNeeded() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.records) <= cm.maxEntries {
		return
	}

	// Sort records by LastAccessed ascending (oldest first)
	type sortItem struct {
		appID string
		rec   *IconRecord
	}
	items := make([]sortItem, 0, len(cm.records))
	for id, r := range cm.records {
		items = append(items, sortItem{id, r})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].rec.LastAccessed.Before(items[j].rec.LastAccessed)
	})

	// Evict oldest entries until under limit
	excess := len(items) - cm.maxEntries
	for i := 0; i < excess; i++ {
		evict := items[i]
		if evict.rec.OriginalPath != "" {
			_ = os.Remove(evict.rec.OriginalPath)
		}
		for _, v := range evict.rec.Variants {
			_ = os.Remove(v)
		}
		delete(cm.records, evict.appID)
	}

	cm.saveCatalogLocked()
}

func (cm *CacheManager) loadCatalog() {
	data, err := os.ReadFile(cm.catalogPath)
	if err != nil {
		return
	}
	cm.mu.Lock()
	defer cm.mu.Unlock()
	_ = json.Unmarshal(data, &cm.records)
}

func (cm *CacheManager) saveCatalogLocked() {
	data, err := json.MarshalIndent(cm.records, "", "  ")
	if err != nil {
		return
	}
	_ = cm.WriteAtomic(cm.catalogPath, data)
}
