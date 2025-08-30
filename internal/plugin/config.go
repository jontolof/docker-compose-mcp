package plugin

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/logging"
)

// ConfigManager handles plugin configuration management
type ConfigManager struct {
	mu         sync.RWMutex
	configDir  string
	configs    map[string]*PluginConfig
	watchers   map[string]*configWatcher
	logger     *logging.Logger
}

// PluginConfig represents configuration for a specific plugin
type PluginConfig struct {
	Name     string                 `json:"name"`
	Version  string                 `json:"version"`
	Enabled  bool                   `json:"enabled"`
	Settings map[string]interface{} `json:"settings"`
	
	// Environment-specific overrides
	Environments map[string]PluginEnvironmentConfig `json:"environments,omitempty"`
	
	// Dependencies configuration
	Dependencies map[string]interface{} `json:"dependencies,omitempty"`
	
	// Metadata
	LastModified time.Time `json:"lastModified"`
	ConfigPath   string    `json:"configPath"`
}

// PluginEnvironmentConfig represents environment-specific plugin settings
type PluginEnvironmentConfig struct {
	Enabled  *bool                  `json:"enabled,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// configWatcher watches for configuration file changes
type configWatcher struct {
	path     string
	lastMod  time.Time
	callback func(*PluginConfig) error
}

// GlobalConfig represents global plugin system configuration
type GlobalConfig struct {
	// Plugin search paths
	SearchPaths []string `json:"searchPaths"`
	
	// Default plugin settings
	Defaults struct {
		Enabled     bool          `json:"enabled"`
		LoadTimeout time.Duration `json:"loadTimeout"`
		MaxPlugins  int          `json:"maxPlugins"`
	} `json:"defaults"`
	
	// Environment configuration
	Environment string `json:"environment"`
	
	// Hot reload settings
	HotReload struct {
		Enabled  bool          `json:"enabled"`
		Interval time.Duration `json:"interval"`
	} `json:"hotReload"`
	
	// Security settings
	Security struct {
		AllowUnsignedPlugins bool     `json:"allowUnsignedPlugins"`
		TrustedPaths        []string `json:"trustedPaths"`
	} `json:"security"`
}

// NewConfigManager creates a new plugin configuration manager
func NewConfigManager(configDir string, logger *logging.Logger) (*ConfigManager, error) {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &ConfigManager{
		configDir: configDir,
		configs:   make(map[string]*PluginConfig),
		watchers:  make(map[string]*configWatcher),
		logger:    logger,
	}, nil
}

// LoadConfig loads configuration for a plugin
func (cm *ConfigManager) LoadConfig(pluginName string) (*PluginConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configPath := filepath.Join(cm.configDir, pluginName+".json")
	
	// Check if already loaded and up-to-date
	if config, exists := cm.configs[pluginName]; exists {
		if stat, err := os.Stat(configPath); err == nil {
			if !stat.ModTime().After(config.LastModified) {
				return config, nil
			}
		}
	}

	// Load from file
	config, err := cm.loadConfigFromFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config
			config = &PluginConfig{
				Name:         pluginName,
				Enabled:      true,
				Settings:     make(map[string]interface{}),
				Environments: make(map[string]PluginEnvironmentConfig),
				Dependencies: make(map[string]interface{}),
				LastModified: time.Now(),
				ConfigPath:   configPath,
			}
			
			if err := cm.saveConfigToFile(config); err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	cm.configs[pluginName] = config
	return config, nil
}

// SaveConfig saves configuration for a plugin
func (cm *ConfigManager) SaveConfig(config *PluginConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	config.LastModified = time.Now()
	
	if err := cm.saveConfigToFile(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	cm.configs[config.Name] = config
	return nil
}

// GetConfig returns the current configuration for a plugin
func (cm *ConfigManager) GetConfig(pluginName string) (*PluginConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config, exists := cm.configs[pluginName]
	return config, exists
}

// UpdateConfig updates specific settings for a plugin
func (cm *ConfigManager) UpdateConfig(pluginName string, updates map[string]interface{}) error {
	config, err := cm.LoadConfig(pluginName)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply updates
	for key, value := range updates {
		if key == "enabled" {
			if enabled, ok := value.(bool); ok {
				config.Enabled = enabled
			}
		} else {
			config.Settings[key] = value
		}
	}

	return cm.SaveConfig(config)
}

// GetEffectiveConfig returns the effective configuration for a plugin in the current environment
func (cm *ConfigManager) GetEffectiveConfig(pluginName, environment string) (*Config, error) {
	pluginConfig, err := cm.LoadConfig(pluginName)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Settings:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	// Copy base settings
	for key, value := range pluginConfig.Settings {
		config.Settings[key] = value
	}

	// Apply environment-specific overrides
	if envConfig, exists := pluginConfig.Environments[environment]; exists {
		if envConfig.Enabled != nil {
			// Environment can override enabled state
		}
		
		for key, value := range envConfig.Settings {
			config.Settings[key] = value
		}
	}

	// Add system environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			config.Environment[parts[0]] = parts[1]
		}
	}

	return config, nil
}

// LoadGlobalConfig loads the global plugin system configuration
func (cm *ConfigManager) LoadGlobalConfig() (*GlobalConfig, error) {
	configPath := filepath.Join(cm.configDir, "global.json")
	
	config := &GlobalConfig{}
	
	// Set defaults
	config.SearchPaths = []string{
		"./plugins",
		"~/.docker-compose-mcp/plugins",
		"/usr/local/lib/docker-compose-mcp/plugins",
	}
	config.Defaults.Enabled = true
	config.Defaults.LoadTimeout = 30 * time.Second
	config.Defaults.MaxPlugins = 50
	config.Environment = "production"
	config.HotReload.Enabled = false
	config.HotReload.Interval = 5 * time.Second
	config.Security.AllowUnsignedPlugins = false

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Save default config
		if err := cm.saveGlobalConfig(configPath, config); err != nil {
			return nil, fmt.Errorf("failed to save default global config: %w", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	return config, nil
}

// SaveGlobalConfig saves the global plugin system configuration
func (cm *ConfigManager) SaveGlobalConfig(config *GlobalConfig) error {
	configPath := filepath.Join(cm.configDir, "global.json")
	return cm.saveGlobalConfig(configPath, config)
}

// WatchConfig starts watching a plugin configuration for changes
func (cm *ConfigManager) WatchConfig(pluginName string, callback func(*PluginConfig) error) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configPath := filepath.Join(cm.configDir, pluginName+".json")
	
	watcher := &configWatcher{
		path:     configPath,
		callback: callback,
	}
	
	if stat, err := os.Stat(configPath); err == nil {
		watcher.lastMod = stat.ModTime()
	}
	
	cm.watchers[pluginName] = watcher
	return nil
}

// StopWatching stops watching a plugin configuration
func (cm *ConfigManager) StopWatching(pluginName string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.watchers, pluginName)
}

// CheckForUpdates checks all watched configurations for updates
func (cm *ConfigManager) CheckForUpdates() error {
	cm.mu.RLock()
	watchers := make(map[string]*configWatcher)
	for name, watcher := range cm.watchers {
		watchers[name] = watcher
	}
	cm.mu.RUnlock()

	for pluginName, watcher := range watchers {
		stat, err := os.Stat(watcher.path)
		if err != nil {
			continue
		}

		if stat.ModTime().After(watcher.lastMod) {
			config, err := cm.LoadConfig(pluginName)
			if err != nil {
				cm.logger.Warnf("Failed to reload config for %s: %v", pluginName, err)
				continue
			}

			if err := watcher.callback(config); err != nil {
				cm.logger.Warnf("Config update callback failed for %s: %v", pluginName, err)
			}

			watcher.lastMod = stat.ModTime()
		}
	}

	return nil
}

// loadConfigFromFile loads configuration from a JSON file
func (cm *ConfigManager) loadConfigFromFile(path string) (*PluginConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config PluginConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.ConfigPath = path
	
	// Set modification time
	if stat, err := os.Stat(path); err == nil {
		config.LastModified = stat.ModTime()
	}

	return &config, nil
}

// saveConfigToFile saves configuration to a JSON file
func (cm *ConfigManager) saveConfigToFile(config *PluginConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(config.ConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// saveGlobalConfig saves global configuration to a JSON file
func (cm *ConfigManager) saveGlobalConfig(path string, config *GlobalConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write global config file: %w", err)
	}

	return nil
}

// ValidateConfig validates a plugin configuration
func (cm *ConfigManager) ValidateConfig(config *PluginConfig) error {
	if config.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if config.Settings == nil {
		config.Settings = make(map[string]interface{})
	}

	if config.Environments == nil {
		config.Environments = make(map[string]PluginEnvironmentConfig)
	}

	if config.Dependencies == nil {
		config.Dependencies = make(map[string]interface{})
	}

	return nil
}

// ListConfigs returns all available plugin configurations
func (cm *ConfigManager) ListConfigs() ([]PluginConfig, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var configs []PluginConfig

	// Walk config directory for .json files
	err := filepath.WalkDir(cm.configDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(path, ".json") && !strings.HasSuffix(path, "global.json") {
			config, err := cm.loadConfigFromFile(path)
			if err != nil {
				cm.logger.Warnf("Failed to load config %s: %v", path, err)
				return nil
			}
			configs = append(configs, *config)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan config directory: %w", err)
	}

	return configs, nil
}