package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/logging"
	"github.com/jontolof/docker-compose-mcp/internal/mcp"
)

// pluginManager implements the Manager interface for complete plugin lifecycle management
type pluginManager struct {
	registry      Registry
	configManager *ConfigManager
	validator     Validator
	logger        *logging.Logger
	
	// Lifecycle management
	mu           sync.RWMutex
	running      bool
	ctx          context.Context
	cancel       context.CancelFunc
	
	// Event system
	eventHooks   map[EventType][]Hook
	eventMu      sync.RWMutex
	
	// Plugin health monitoring
	healthChecks map[string]*healthCheck
	healthMu     sync.RWMutex
	
	// Configuration
	config ManagerConfig
	
	// Hot reload
	reloadTicker *time.Ticker
}

// healthCheck tracks plugin health status
type healthCheck struct {
	plugin     Plugin
	lastCheck  time.Time
	status     HealthStatus
	failCount  int
	maxFails   int
}

// NewManager creates a new plugin manager
func NewManager(configDir string, logger *logging.Logger) (Manager, error) {
	registry := NewRegistry(logger)
	
	configManager, err := NewConfigManager(configDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}
	
	validator := NewValidator(logger)
	
	return &pluginManager{
		registry:      registry,
		configManager: configManager,
		validator:     validator,
		logger:        logger,
		eventHooks:    make(map[EventType][]Hook),
		healthChecks:  make(map[string]*healthCheck),
	}, nil
}

// Initialize initializes the plugin manager
func (pm *pluginManager) Initialize(ctx context.Context, config ManagerConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.ctx, pm.cancel = context.WithCancel(ctx)
	pm.config = config

	// Load global configuration
	globalConfig, err := pm.configManager.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Override config with global settings
	if len(globalConfig.SearchPaths) > 0 {
		pm.config.SearchPaths = globalConfig.SearchPaths
	}
	if globalConfig.Defaults.LoadTimeout > 0 {
		pm.config.LoadTimeout = globalConfig.Defaults.LoadTimeout
	}
	if globalConfig.Defaults.MaxPlugins > 0 {
		pm.config.MaxPlugins = globalConfig.Defaults.MaxPlugins
	}

	pm.config.HotReload = globalConfig.HotReload.Enabled

	pm.logger.Infof("Plugin manager initialized with config: SearchPaths=%v, MaxPlugins=%d, HotReload=%v", 
		pm.config.SearchPaths, pm.config.MaxPlugins, pm.config.HotReload)

	return nil
}

// Start starts the plugin manager
func (pm *pluginManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.running {
		return fmt.Errorf("plugin manager is already running")
	}

	// Discover and load all enabled plugins
	descriptors, err := pm.Discover()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	loadedCount := 0
	for _, desc := range descriptors {
		if desc.Enabled && loadedCount < pm.config.MaxPlugins {
			if err := pm.loadPluginWithValidation(desc.Name); err != nil {
				pm.logger.Warnf("Failed to load plugin %s: %v", desc.Name, err)
			} else {
				loadedCount++
			}
		}
	}

	pm.running = true

	// Start hot reload if enabled
	if pm.config.HotReload {
		pm.startHotReload()
	}

	// Start health monitoring
	pm.startHealthMonitoring()

	pm.logger.Infof("Plugin manager started, loaded %d plugins", loadedCount)
	return nil
}

// Stop stops the plugin manager
func (pm *pluginManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return nil
	}

	// Stop hot reload
	if pm.reloadTicker != nil {
		pm.reloadTicker.Stop()
		pm.reloadTicker = nil
	}

	// Cancel context
	if pm.cancel != nil {
		pm.cancel()
	}

	// Unload all plugins
	plugins := pm.registry.List()
	for _, plugin := range plugins {
		if err := plugin.Cleanup(); err != nil {
			pm.logger.Warnf("Plugin cleanup failed for %s: %v", plugin.Info().Name, err)
		}
	}

	pm.running = false
	pm.logger.Info("Plugin manager stopped")
	return nil
}

// loadPluginWithValidation loads a plugin with full validation
func (pm *pluginManager) loadPluginWithValidation(name string) error {
	// Load plugin configuration
	config, err := pm.configManager.LoadConfig(name)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !config.Enabled {
		return fmt.Errorf("plugin %s is disabled", name)
	}

	// Discover plugin
	descriptors, err := pm.Discover()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	var descriptor PluginDescriptor
	found := false
	for _, desc := range descriptors {
		if desc.Name == name {
			descriptor = desc
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Validate plugin
	if err := pm.validator.ValidatePlugin(descriptor); err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	// Load the plugin
	plugin, err := pm.registry.Load(name)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Validate plugin configuration
	effectiveConfig, err := pm.configManager.GetEffectiveConfig(name, pm.config.ConfigDir)
	if err != nil {
		return fmt.Errorf("failed to get effective config: %w", err)
	}

	if err := pm.validator.ValidateConfig(plugin, *effectiveConfig); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Validate dependencies
	if err := pm.validator.ValidateDependencies(plugin); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	// Register event hooks
	pm.registerPluginHooks(plugin)

	// Start health monitoring for the plugin
	pm.startPluginHealthCheck(plugin)

	// Fire plugin loaded event
	pm.fireEvent(EventType("plugin_loaded"), map[string]interface{}{
		"plugin": plugin.Info().Name,
		"version": plugin.Info().Version,
	})

	return nil
}

// registerPluginHooks registers event hooks from a plugin
func (pm *pluginManager) registerPluginHooks(plugin Plugin) {
	pm.eventMu.Lock()
	defer pm.eventMu.Unlock()

	hooks := plugin.Hooks()
	for _, hook := range hooks {
		if pm.eventHooks[hook.Event] == nil {
			pm.eventHooks[hook.Event] = make([]Hook, 0)
		}
		pm.eventHooks[hook.Event] = append(pm.eventHooks[hook.Event], hook)
	}
}

// fireEvent fires an event to all registered hooks
func (pm *pluginManager) fireEvent(eventType EventType, data map[string]interface{}) {
	pm.eventMu.RLock()
	hooks := pm.eventHooks[eventType]
	pm.eventMu.RUnlock()

	if len(hooks) == 0 {
		return
	}

	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		Context:   pm.ctx,
	}

	for _, hook := range hooks {
		go func(h Hook) {
			if err := h.Handler(pm.ctx, event); err != nil {
				pm.logger.Warnf("Event hook failed for %s: %v", eventType, err)
			}
		}(hook)
	}
}

// startHotReload starts hot reload monitoring
func (pm *pluginManager) startHotReload() {
	interval := 5 * time.Second
	if pm.config.LoadTimeout > 0 {
		interval = pm.config.LoadTimeout / 6 // Check every 1/6 of load timeout
	}

	pm.reloadTicker = time.NewTicker(interval)
	
	go func() {
		for {
			select {
			case <-pm.reloadTicker.C:
				if err := pm.configManager.CheckForUpdates(); err != nil {
					pm.logger.Warnf("Config update check failed: %v", err)
				}
			case <-pm.ctx.Done():
				return
			}
		}
	}()
}

// startHealthMonitoring starts plugin health monitoring
func (pm *pluginManager) startHealthMonitoring() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pm.checkPluginHealth()
			case <-pm.ctx.Done():
				return
			}
		}
	}()
}

// startPluginHealthCheck starts health checking for a specific plugin
func (pm *pluginManager) startPluginHealthCheck(plugin Plugin) {
	pm.healthMu.Lock()
	defer pm.healthMu.Unlock()

	pm.healthChecks[plugin.Info().Name] = &healthCheck{
		plugin:    plugin,
		lastCheck: time.Now(),
		status:    HealthStatus{Status: "unknown"},
		maxFails:  3,
	}
}

// checkPluginHealth performs health checks on all plugins
func (pm *pluginManager) checkPluginHealth() {
	pm.healthMu.RLock()
	checks := make(map[string]*healthCheck)
	for name, check := range pm.healthChecks {
		checks[name] = check
	}
	pm.healthMu.RUnlock()

	for name, check := range checks {
		status := check.plugin.Health()
		check.lastCheck = time.Now()
		
		// Count consecutive failures
		if status.Status == "unhealthy" {
			check.failCount++
		} else {
			check.failCount = 0
		}

		// Auto-unload if too many failures
		if check.failCount >= check.maxFails {
			pm.logger.Warnf("Plugin %s failed health check %d times, unloading", name, check.failCount)
			if err := pm.Unload(name); err != nil {
				pm.logger.Errorf("Failed to unload unhealthy plugin %s: %v", name, err)
			}
			continue
		}

		check.status = status
	}
}

// Reload reloads all plugins
func (pm *pluginManager) Reload() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return fmt.Errorf("plugin manager is not running")
	}

	// Get currently loaded plugins
	currentPlugins := pm.registry.List()
	pluginNames := make([]string, len(currentPlugins))
	for i, plugin := range currentPlugins {
		pluginNames[i] = plugin.Info().Name
	}

	// Unload all current plugins
	for _, name := range pluginNames {
		if err := pm.Unload(name); err != nil {
			pm.logger.Warnf("Failed to unload plugin %s during reload: %v", name, err)
		}
	}

	// Clear event hooks and health checks
	pm.eventMu.Lock()
	pm.eventHooks = make(map[EventType][]Hook)
	pm.eventMu.Unlock()

	pm.healthMu.Lock()
	pm.healthChecks = make(map[string]*healthCheck)
	pm.healthMu.Unlock()

	// Reload all plugins
	descriptors, err := pm.Discover()
	if err != nil {
		return fmt.Errorf("failed to discover plugins during reload: %w", err)
	}

	loadedCount := 0
	for _, desc := range descriptors {
		if desc.Enabled && loadedCount < pm.config.MaxPlugins {
			if err := pm.loadPluginWithValidation(desc.Name); err != nil {
				pm.logger.Warnf("Failed to load plugin %s during reload: %v", desc.Name, err)
			} else {
				loadedCount++
			}
		}
	}

	pm.logger.Infof("Plugin reload completed, loaded %d plugins", loadedCount)
	return nil
}

// Install installs a plugin from a source
func (pm *pluginManager) Install(source string) error {
	// This would implement plugin installation from various sources
	// For now, just return not implemented
	return fmt.Errorf("plugin installation not yet implemented")
}

// Uninstall uninstalls a plugin
func (pm *pluginManager) Uninstall(name string) error {
	// Unload the plugin first
	if err := pm.Unload(name); err != nil {
		return fmt.Errorf("failed to unload plugin: %w", err)
	}

	// This would implement plugin removal from filesystem
	// For now, just return success
	pm.logger.Infof("Plugin %s uninstalled (removal from filesystem not implemented)", name)
	return nil
}

// Update updates a plugin to the latest version
func (pm *pluginManager) Update(name string) error {
	// This would implement plugin updates
	// For now, just return not implemented
	return fmt.Errorf("plugin updates not yet implemented")
}

// GetTools returns all tools from all loaded plugins
func (pm *pluginManager) GetTools() []mcp.Tool {
	plugins := pm.registry.List()
	var allTools []mcp.Tool

	for _, plugin := range plugins {
		tools := plugin.Tools()
		allTools = append(allTools, tools...)
	}

	return allTools
}

// FireEvent fires an event (public method for external use)
func (pm *pluginManager) FireEvent(eventType EventType, data map[string]interface{}) {
	pm.fireEvent(eventType, data)
}

// GetHealthStatus returns health status for all plugins
func (pm *pluginManager) GetHealthStatus() map[string]HealthStatus {
	pm.healthMu.RLock()
	defer pm.healthMu.RUnlock()

	status := make(map[string]HealthStatus)
	for name, check := range pm.healthChecks {
		status[name] = check.status
	}

	return status
}

// Delegate Registry interface methods
func (pm *pluginManager) Discover() ([]PluginDescriptor, error) {
	return pm.registry.Discover()
}

func (pm *pluginManager) Load(name string) (Plugin, error) {
	return pm.registry.Load(name)
}

func (pm *pluginManager) Unload(name string) error {
	// Remove from health checks
	pm.healthMu.Lock()
	delete(pm.healthChecks, name)
	pm.healthMu.Unlock()

	// Fire event
	pm.fireEvent(EventType("plugin_unloaded"), map[string]interface{}{
		"plugin": name,
	})

	return pm.registry.Unload(name)
}

func (pm *pluginManager) List() []Plugin {
	return pm.registry.List()
}

func (pm *pluginManager) Get(name string) (Plugin, bool) {
	return pm.registry.Get(name)
}

// Delegate Validator interface methods
func (pm *pluginManager) ValidatePlugin(descriptor PluginDescriptor) error {
	return pm.validator.ValidatePlugin(descriptor)
}

func (pm *pluginManager) ValidateConfig(plugin Plugin, config Config) error {
	return pm.validator.ValidateConfig(plugin, config)
}

func (pm *pluginManager) ValidateDependencies(plugin Plugin) error {
	return pm.validator.ValidateDependencies(plugin)
}