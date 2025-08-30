package plugin

import (
	"context"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/mcp"
)

// Plugin represents a loadable plugin that can extend docker-compose-mcp functionality
type Plugin interface {
	// Info returns plugin metadata
	Info() PluginInfo
	
	// Initialize is called when the plugin is loaded
	Initialize(ctx context.Context, config Config) error
	
	// Tools returns the MCP tools provided by this plugin
	Tools() []mcp.Tool
	
	// Hooks returns event hooks provided by this plugin
	Hooks() []Hook
	
	// Cleanup is called when the plugin is unloaded
	Cleanup() error
	
	// Health performs a health check on the plugin
	Health() HealthStatus
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Website     string   `json:"website,omitempty"`
	License     string   `json:"license,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	
	// Dependencies required by this plugin
	Dependencies []Dependency `json:"dependencies,omitempty"`
	
	// Minimum docker-compose-mcp version required
	MinVersion string `json:"minVersion"`
}

// Dependency represents a plugin dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // "plugin", "binary", "service"
}

// Config represents plugin configuration
type Config struct {
	// Plugin-specific configuration map
	Settings map[string]interface{} `json:"settings,omitempty"`
	
	// Workspace context
	WorkspacePath string `json:"workspacePath,omitempty"`
	
	// Docker context information
	DockerHost    string `json:"dockerHost,omitempty"`
	DockerContext string `json:"dockerContext,omitempty"`
	
	// Environment variables
	Environment map[string]string `json:"environment,omitempty"`
}

// Hook represents an event hook that plugins can register
type Hook struct {
	Event   EventType `json:"event"`
	Handler func(ctx context.Context, event Event) error
}

// EventType represents different events in the plugin lifecycle
type EventType string

const (
	EventPreCommand   EventType = "pre_command"
	EventPostCommand  EventType = "post_command"
	EventWorkspaceChange EventType = "workspace_change"
	EventDockerHostChange EventType = "docker_host_change"
	EventServiceStart EventType = "service_start"
	EventServiceStop  EventType = "service_stop"
	EventError        EventType = "error"
)

// Event contains event data passed to hook handlers
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Context   context.Context        `json:"-"`
}

// HealthStatus represents plugin health
type HealthStatus struct {
	Status  string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
	LastCheck time.Time            `json:"lastCheck"`
}

// PluginType represents different types of plugins
type PluginType string

const (
	PluginTypeCore      PluginType = "core"      // Core functionality extensions
	PluginTypeWorkflow  PluginType = "workflow"  // Workflow and automation
	PluginTypeIntegration PluginType = "integration" // Third-party integrations
	PluginTypeFilter    PluginType = "filter"    // Custom output filters
	PluginTypeMonitoring PluginType = "monitoring" // Monitoring and observability
)

// Registry interface for plugin discovery and loading
type Registry interface {
	// Discover finds available plugins in search paths
	Discover() ([]PluginDescriptor, error)
	
	// Load loads a plugin by name
	Load(name string) (Plugin, error)
	
	// Unload unloads a plugin by name
	Unload(name string) error
	
	// List returns all loaded plugins
	List() []Plugin
	
	// Get returns a specific loaded plugin
	Get(name string) (Plugin, bool)
}

// PluginDescriptor describes an available plugin
type PluginDescriptor struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     PluginType `json:"type"`
	Info     PluginInfo `json:"info"`
	Enabled  bool       `json:"enabled"`
	LoadTime time.Time  `json:"loadTime,omitempty"`
}

// Validator interface for plugin validation
type Validator interface {
	// ValidatePlugin validates a plugin before loading
	ValidatePlugin(descriptor PluginDescriptor) error
	
	// ValidateConfig validates plugin configuration
	ValidateConfig(plugin Plugin, config Config) error
	
	// ValidateDependencies validates plugin dependencies
	ValidateDependencies(plugin Plugin) error
}

// Manager interface for plugin lifecycle management
type Manager interface {
	Registry
	Validator
	
	// Initialize initializes the plugin manager
	Initialize(ctx context.Context, config ManagerConfig) error
	
	// Start starts the plugin manager
	Start() error
	
	// Stop stops the plugin manager
	Stop() error
	
	// Reload reloads all plugins
	Reload() error
	
	// Install installs a plugin from a source
	Install(source string) error
	
	// Uninstall uninstalls a plugin
	Uninstall(name string) error
	
	// Update updates a plugin to the latest version
	Update(name string) error
}

// ManagerConfig represents plugin manager configuration
type ManagerConfig struct {
	// Plugin search paths
	SearchPaths []string `json:"searchPaths"`
	
	// Enable plugin hot-reload
	HotReload bool `json:"hotReload"`
	
	// Plugin load timeout
	LoadTimeout time.Duration `json:"loadTimeout"`
	
	// Maximum concurrent plugins
	MaxPlugins int `json:"maxPlugins"`
	
	// Plugin configuration directory
	ConfigDir string `json:"configDir"`
	
	// Plugin data directory
	DataDir string `json:"dataDir"`
}