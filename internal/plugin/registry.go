package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/logging"
)

// pluginRegistry implements the Registry interface
type pluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]*loadedPlugin
	logger  *logging.Logger
}

// loadedPlugin wraps a plugin with metadata
type loadedPlugin struct {
	Plugin     Plugin
	Descriptor PluginDescriptor
	Config     Config
	LoadTime   time.Time
}

// NewRegistry creates a new plugin registry
func NewRegistry(logger *logging.Logger) Registry {
	return &pluginRegistry{
		plugins: make(map[string]*loadedPlugin),
		logger:  logger,
	}
}

// Discover finds available plugins in search paths
func (r *pluginRegistry) Discover() ([]PluginDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var descriptors []PluginDescriptor
	searchPaths := []string{
		"./plugins",
		"~/.docker-compose-mcp/plugins",
		"/usr/local/lib/docker-compose-mcp/plugins",
	}

	for _, searchPath := range searchPaths {
		// Expand home directory
		if strings.HasPrefix(searchPath, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			searchPath = filepath.Join(home, searchPath[2:])
		}

		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue on errors
			}

			if !strings.HasSuffix(path, ".so") && !strings.HasSuffix(path, ".dylib") {
				return nil
			}

			descriptor, err := r.loadDescriptor(path)
			if err != nil {
				r.logger.Warnf("Failed to load plugin descriptor: %s: %v", path, err)
				return nil
			}

			descriptors = append(descriptors, descriptor)
			return nil
		})

		if err != nil {
			r.logger.Warnf("Failed to scan plugin directory %s: %v", searchPath, err)
		}
	}

	return descriptors, nil
}

// loadDescriptor loads plugin metadata without fully loading the plugin
func (r *pluginRegistry) loadDescriptor(path string) (PluginDescriptor, error) {
	// For now, we'll load the plugin to get its info
	// In a production system, you might want to cache this or use a metadata file
	p, err := plugin.Open(path)
	if err != nil {
		return PluginDescriptor{}, fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// Look for the NewPlugin function
	newPluginSymbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return PluginDescriptor{}, fmt.Errorf("plugin %s missing NewPlugin function: %w", path, err)
	}

	newPlugin, ok := newPluginSymbol.(func() Plugin)
	if !ok {
		return PluginDescriptor{}, fmt.Errorf("plugin %s NewPlugin function has wrong signature", path)
	}

	pluginInstance := newPlugin()
	info := pluginInstance.Info()

	return PluginDescriptor{
		Name: info.Name,
		Path: path,
		Type: determinePluginType(info),
		Info: info,
		Enabled: true,
	}, nil
}

// determinePluginType determines plugin type from info
func determinePluginType(info PluginInfo) PluginType {
	// Simple heuristic based on tags and name
	for _, tag := range info.Tags {
		switch strings.ToLower(tag) {
		case "core":
			return PluginTypeCore
		case "workflow", "automation":
			return PluginTypeWorkflow
		case "integration", "third-party":
			return PluginTypeIntegration
		case "filter", "output":
			return PluginTypeFilter
		case "monitoring", "observability":
			return PluginTypeMonitoring
		}
	}

	// Default based on name patterns
	name := strings.ToLower(info.Name)
	if strings.Contains(name, "workflow") || strings.Contains(name, "automation") {
		return PluginTypeWorkflow
	}
	if strings.Contains(name, "filter") || strings.Contains(name, "output") {
		return PluginTypeFilter
	}
	if strings.Contains(name, "monitor") || strings.Contains(name, "metric") {
		return PluginTypeMonitoring
	}
	if strings.Contains(name, "integration") {
		return PluginTypeIntegration
	}

	return PluginTypeCore
}

// Load loads a plugin by name
func (r *pluginRegistry) Load(name string) (Plugin, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already loaded
	if loaded, exists := r.plugins[name]; exists {
		return loaded.Plugin, nil
	}

	// Discover available plugins
	descriptors, err := r.Discover()
	if err != nil {
		return nil, fmt.Errorf("failed to discover plugins: %w", err)
	}

	// Find the plugin
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
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	// Load the plugin
	p, err := plugin.Open(descriptor.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin %s: %w", descriptor.Path, err)
	}

	// Look for the NewPlugin function
	newPluginSymbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin %s missing NewPlugin function: %w", descriptor.Path, err)
	}

	newPlugin, ok := newPluginSymbol.(func() Plugin)
	if !ok {
		return nil, fmt.Errorf("plugin %s NewPlugin function has wrong signature", descriptor.Path)
	}

	pluginInstance := newPlugin()

	// Initialize the plugin
	config := Config{
		Settings: make(map[string]interface{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pluginInstance.Initialize(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Store the loaded plugin
	r.plugins[name] = &loadedPlugin{
		Plugin:     pluginInstance,
		Descriptor: descriptor,
		Config:     config,
		LoadTime:   time.Now(),
	}

	r.logger.Infof("Loaded plugin: %s v%s", name, pluginInstance.Info().Version)
	return pluginInstance, nil
}

// Unload unloads a plugin by name
func (r *pluginRegistry) Unload(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	loaded, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	// Cleanup the plugin
	if err := loaded.Plugin.Cleanup(); err != nil {
		r.logger.Warnf("Plugin cleanup failed for %s: %v", name, err)
	}

	delete(r.plugins, name)
	r.logger.Infof("Unloaded plugin: %s", name)
	return nil
}

// List returns all loaded plugins
func (r *pluginRegistry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, loaded := range r.plugins {
		plugins = append(plugins, loaded.Plugin)
	}

	return plugins
}

// Get returns a specific loaded plugin
func (r *pluginRegistry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if loaded, exists := r.plugins[name]; exists {
		return loaded.Plugin, true
	}

	return nil, false
}

// GetDescriptors returns all loaded plugin descriptors
func (r *pluginRegistry) GetDescriptors() []PluginDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	descriptors := make([]PluginDescriptor, 0, len(r.plugins))
	for _, loaded := range r.plugins {
		desc := loaded.Descriptor
		desc.LoadTime = loaded.LoadTime
		descriptors = append(descriptors, desc)
	}

	return descriptors
}

// GetLoadedPlugin returns loaded plugin with metadata
func (r *pluginRegistry) GetLoadedPlugin(name string) (*loadedPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	loaded, exists := r.plugins[name]
	return loaded, exists
}

// LoadAll loads all discovered plugins
func (r *pluginRegistry) LoadAll() error {
	descriptors, err := r.Discover()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	for _, descriptor := range descriptors {
		if descriptor.Enabled {
			if _, err := r.Load(descriptor.Name); err != nil {
				r.logger.Warnf("Failed to load plugin %s: %v", descriptor.Name, err)
			}
		}
	}

	return nil
}

// UnloadAll unloads all plugins
func (r *pluginRegistry) UnloadAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []string
	for name := range r.plugins {
		if err := r.Unload(name); err != nil {
			errors = append(errors, fmt.Sprintf("failed to unload %s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("unload errors: %s", strings.Join(errors, "; "))
	}

	return nil
}