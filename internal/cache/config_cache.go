package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ConfigCache struct {
	mu       sync.RWMutex
	cache    map[string]*CacheEntry
	maxSize  int
	maxAge   time.Duration
	basePath string
}

type CacheEntry struct {
	Config     interface{}
	Hash       string
	ModTime    time.Time
	CachedAt   time.Time
	AccessedAt time.Time
	Services   []string
	Networks   []string
	Volumes    []string
}

type ComposeConfig struct {
	Services map[string]interface{} `json:"services,omitempty"`
	Networks map[string]interface{} `json:"networks,omitempty"`
	Volumes  map[string]interface{} `json:"volumes,omitempty"`
	Version  string                 `json:"version,omitempty"`
}

func NewConfigCache(basePath string, maxSize int, maxAge time.Duration) *ConfigCache {
	cache := &ConfigCache{
		cache:    make(map[string]*CacheEntry),
		maxSize:  maxSize,
		maxAge:   maxAge,
		basePath: basePath,
	}
	
	// Start cleanup goroutine
	go cache.cleanupLoop()
	
	return cache
}

func (c *ConfigCache) Get(composeFile string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.cache[composeFile]
	if !exists {
		return nil, false
	}
	
	// Check if file has been modified
	fileInfo, err := os.Stat(composeFile)
	if err != nil {
		delete(c.cache, composeFile)
		return nil, false
	}
	
	// Check if entry is stale
	if fileInfo.ModTime().After(entry.ModTime) || 
	   time.Since(entry.CachedAt) > c.maxAge {
		delete(c.cache, composeFile)
		return nil, false
	}
	
	// Update access time
	entry.AccessedAt = time.Now()
	
	return entry, true
}

func (c *ConfigCache) Set(composeFile string, config interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	fileInfo, err := os.Stat(composeFile)
	if err != nil {
		return fmt.Errorf("failed to stat compose file: %w", err)
	}
	
	// Calculate file hash for integrity check
	hash, err := c.calculateFileHash(composeFile)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}
	
	// Extract service, network, and volume names
	services, networks, volumes := c.extractNames(config)
	
	entry := &CacheEntry{
		Config:     config,
		Hash:       hash,
		ModTime:    fileInfo.ModTime(),
		CachedAt:   time.Now(),
		AccessedAt: time.Now(),
		Services:   services,
		Networks:   networks,
		Volumes:    volumes,
	}
	
	// Evict oldest entries if cache is full
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}
	
	c.cache[composeFile] = entry
	return nil
}

func (c *ConfigCache) GetServices(composeFile string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if entry, exists := c.cache[composeFile]; exists {
		return append([]string(nil), entry.Services...)
	}
	return nil
}

func (c *ConfigCache) GetNetworks(composeFile string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if entry, exists := c.cache[composeFile]; exists {
		return append([]string(nil), entry.Networks...)
	}
	return nil
}

func (c *ConfigCache) GetVolumes(composeFile string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if entry, exists := c.cache[composeFile]; exists {
		return append([]string(nil), entry.Volumes...)
	}
	return nil
}

func (c *ConfigCache) Invalidate(composeFile string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, composeFile)
}

func (c *ConfigCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*CacheEntry)
}

func (c *ConfigCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var totalSize int
	for _, entry := range c.cache {
		// Estimate memory usage
		data, _ := json.Marshal(entry.Config)
		totalSize += len(data)
	}
	
	return CacheStats{
		Entries:     len(c.cache),
		MaxEntries:  c.maxSize,
		TotalSizeKB: totalSize / 1024,
		MaxAge:      c.maxAge,
	}
}

type CacheStats struct {
	Entries     int           `json:"entries"`
	MaxEntries  int           `json:"maxEntries"`
	TotalSizeKB int           `json:"totalSizeKB"`
	MaxAge      time.Duration `json:"maxAge"`
}

func (c *ConfigCache) calculateFileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash), nil
}

func (c *ConfigCache) extractNames(config interface{}) ([]string, []string, []string) {
	var services, networks, volumes []string
	
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return services, networks, volumes
	}
	
	if servicesMap, ok := configMap["services"].(map[string]interface{}); ok {
		for name := range servicesMap {
			services = append(services, name)
		}
	}
	
	if networksMap, ok := configMap["networks"].(map[string]interface{}); ok {
		for name := range networksMap {
			networks = append(networks, name)
		}
	}
	
	if volumesMap, ok := configMap["volumes"].(map[string]interface{}); ok {
		for name := range volumesMap {
			volumes = append(volumes, name)
		}
	}
	
	return services, networks, volumes
}

func (c *ConfigCache) evictOldest() {
	var oldestFile string
	var oldestTime time.Time
	
	for file, entry := range c.cache {
		if oldestFile == "" || entry.AccessedAt.Before(oldestTime) {
			oldestFile = file
			oldestTime = entry.AccessedAt
		}
	}
	
	if oldestFile != "" {
		delete(c.cache, oldestFile)
	}
}

func (c *ConfigCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute * 5) // Cleanup every 5 minutes
	defer ticker.Stop()
	
	for range ticker.C {
		c.cleanup()
	}
}

func (c *ConfigCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for file, entry := range c.cache {
		if now.Sub(entry.CachedAt) > c.maxAge {
			delete(c.cache, file)
		}
	}
}

// GetConfigPath resolves the docker-compose file path
func (c *ConfigCache) GetConfigPath(workDir string) string {
	// Look for common compose file names
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}
	
	for _, candidate := range candidates {
		path := filepath.Join(workDir, candidate)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// Default to docker-compose.yml
	return filepath.Join(workDir, "docker-compose.yml")
}