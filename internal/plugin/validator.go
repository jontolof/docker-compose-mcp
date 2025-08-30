package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jontolof/docker-compose-mcp/internal/logging"
)

// pluginValidator implements the Validator interface
type pluginValidator struct {
	mu     sync.RWMutex
	logger *logging.Logger
	
	// Security settings
	allowUnsigned bool
	trustedPaths  []string
}

// NewValidator creates a new plugin validator
func NewValidator(logger *logging.Logger) Validator {
	return &pluginValidator{
		logger:        logger,
		allowUnsigned: false, // Default to secure
		trustedPaths: []string{
			"/usr/local/lib/docker-compose-mcp/plugins",
			"/opt/docker-compose-mcp/plugins",
		},
	}
}

// ValidatePlugin validates a plugin before loading
func (v *pluginValidator) ValidatePlugin(descriptor PluginDescriptor) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Validate basic descriptor fields
	if err := v.validateDescriptor(descriptor); err != nil {
		return fmt.Errorf("descriptor validation failed: %w", err)
	}

	// Validate file system permissions and location
	if err := v.validateFilesystem(descriptor); err != nil {
		return fmt.Errorf("filesystem validation failed: %w", err)
	}

	// Validate plugin info
	if err := v.validatePluginInfo(descriptor.Info); err != nil {
		return fmt.Errorf("plugin info validation failed: %w", err)
	}

	// Validate dependencies
	if err := v.validatePluginDependencies(descriptor.Info); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	// Security validation
	if err := v.validateSecurity(descriptor); err != nil {
		return fmt.Errorf("security validation failed: %w", err)
	}

	return nil
}

// ValidateConfig validates plugin configuration
func (v *pluginValidator) ValidateConfig(plugin Plugin, config Config) error {
	info := plugin.Info()

	// Validate required configuration fields based on plugin metadata
	if err := v.validateRequiredConfig(info, config); err != nil {
		return fmt.Errorf("required config validation failed: %w", err)
	}

	// Validate configuration values
	if err := v.validateConfigValues(info, config); err != nil {
		return fmt.Errorf("config value validation failed: %w", err)
	}

	return nil
}

// ValidateDependencies validates plugin dependencies
func (v *pluginValidator) ValidateDependencies(plugin Plugin) error {
	info := plugin.Info()

	for _, dep := range info.Dependencies {
		if err := v.validateDependency(dep); err != nil {
			return fmt.Errorf("dependency %s validation failed: %w", dep.Name, err)
		}
	}

	return nil
}

// validateDescriptor validates basic descriptor fields
func (v *pluginValidator) validateDescriptor(desc PluginDescriptor) error {
	if desc.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if desc.Path == "" {
		return fmt.Errorf("plugin path is required")
	}

	// Validate name format (alphanumeric, hyphens, underscores only)
	if !isValidPluginName(desc.Name) {
		return fmt.Errorf("invalid plugin name format: %s", desc.Name)
	}

	return nil
}

// validateFilesystem validates file system aspects of the plugin
func (v *pluginValidator) validateFilesystem(desc PluginDescriptor) error {
	// Check if file exists
	if _, err := os.Stat(desc.Path); os.IsNotExist(err) {
		return fmt.Errorf("plugin file does not exist: %s", desc.Path)
	}

	// Check file permissions
	info, err := os.Stat(desc.Path)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Check if file is executable
	mode := info.Mode()
	if mode&0111 == 0 {
		return fmt.Errorf("plugin file is not executable: %s", desc.Path)
	}

	// Check if file is in a trusted path (if security is enabled)
	if !v.allowUnsigned && !v.isInTrustedPath(desc.Path) {
		return fmt.Errorf("plugin not in trusted path: %s", desc.Path)
	}

	return nil
}

// validatePluginInfo validates the PluginInfo structure
func (v *pluginValidator) validatePluginInfo(info PluginInfo) error {
	if info.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if info.Version == "" {
		return fmt.Errorf("plugin version is required")
	}

	if info.Description == "" {
		return fmt.Errorf("plugin description is required")
	}

	// Validate version format (semantic versioning)
	if !isValidVersion(info.Version) {
		return fmt.Errorf("invalid version format: %s", info.Version)
	}

	// Validate minimum version requirement
	if info.MinVersion != "" && !isValidVersion(info.MinVersion) {
		return fmt.Errorf("invalid minimum version format: %s", info.MinVersion)
	}

	return nil
}

// validatePluginDependencies validates plugin dependencies
func (v *pluginValidator) validatePluginDependencies(info PluginInfo) error {
	for _, dep := range info.Dependencies {
		if dep.Name == "" {
			return fmt.Errorf("dependency name is required")
		}

		if dep.Version == "" {
			return fmt.Errorf("dependency version is required for %s", dep.Name)
		}

		if dep.Type == "" {
			return fmt.Errorf("dependency type is required for %s", dep.Name)
		}

		// Validate dependency type
		switch dep.Type {
		case "plugin", "binary", "service":
			// Valid types
		default:
			return fmt.Errorf("invalid dependency type %s for %s", dep.Type, dep.Name)
		}

		// Validate version format
		if !isValidVersion(dep.Version) {
			return fmt.Errorf("invalid dependency version format %s for %s", dep.Version, dep.Name)
		}
	}

	return nil
}

// validateSecurity performs security validation
func (v *pluginValidator) validateSecurity(desc PluginDescriptor) error {
	// Check for dangerous file extensions
	ext := strings.ToLower(filepath.Ext(desc.Path))
	if ext != ".so" && ext != ".dylib" && ext != ".dll" {
		return fmt.Errorf("invalid plugin file extension: %s", ext)
	}

	// Additional security checks could be added here:
	// - Code signing verification
	// - Hash validation
	// - Malware scanning
	// - Sandboxing checks

	return nil
}

// validateRequiredConfig validates required configuration fields
func (v *pluginValidator) validateRequiredConfig(info PluginInfo, config Config) error {
	// This would be enhanced to check plugin-specific required fields
	// For now, just validate basic structure

	if config.Settings == nil {
		return fmt.Errorf("plugin settings are required")
	}

	return nil
}

// validateConfigValues validates configuration values
func (v *pluginValidator) validateConfigValues(info PluginInfo, config Config) error {
	// This would be enhanced with plugin-specific validation rules
	// For now, just basic validation

	for key, value := range config.Settings {
		if key == "" {
			return fmt.Errorf("empty configuration key")
		}

		if value == nil {
			return fmt.Errorf("null value for configuration key %s", key)
		}
	}

	return nil
}

// validateDependency validates a single dependency
func (v *pluginValidator) validateDependency(dep Dependency) error {
	switch dep.Type {
	case "plugin":
		return v.validatePluginDependency(dep)
	case "binary":
		return v.validateBinaryDependency(dep)
	case "service":
		return v.validateServiceDependency(dep)
	default:
		return fmt.Errorf("unknown dependency type: %s", dep.Type)
	}
}

// validatePluginDependency validates a plugin dependency
func (v *pluginValidator) validatePluginDependency(dep Dependency) error {
	// Check if the required plugin is available
	// This would integrate with the plugin registry
	v.logger.Debugf("Validating plugin dependency: %s v%s", dep.Name, dep.Version)
	return nil
}

// validateBinaryDependency validates a binary dependency
func (v *pluginValidator) validateBinaryDependency(dep Dependency) error {
	// Check if the required binary is available in PATH
	if err := v.checkBinaryExists(dep.Name); err != nil {
		return fmt.Errorf("binary dependency %s not found: %w", dep.Name, err)
	}

	return nil
}

// validateServiceDependency validates a service dependency
func (v *pluginValidator) validateServiceDependency(dep Dependency) error {
	// Check if the required service is available
	// This could check Docker services, system services, etc.
	v.logger.Debugf("Validating service dependency: %s", dep.Name)
	return nil
}

// checkBinaryExists checks if a binary exists in PATH
func (v *pluginValidator) checkBinaryExists(name string) error {
	// Simple check - look for binary in PATH
	_, err := os.Stat(filepath.Join("/usr/bin", name))
	if err == nil {
		return nil
	}

	_, err = os.Stat(filepath.Join("/usr/local/bin", name))
	if err == nil {
		return nil
	}

	// More comprehensive PATH search could be added
	return fmt.Errorf("binary %s not found in common locations", name)
}

// isInTrustedPath checks if a path is in the trusted paths list
func (v *pluginValidator) isInTrustedPath(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, trustedPath := range v.trustedPaths {
		absTrustedPath, err := filepath.Abs(trustedPath)
		if err != nil {
			continue
		}

		if strings.HasPrefix(absPath, absTrustedPath) {
			return true
		}
	}

	return false
}

// SetSecuritySettings configures security settings
func (v *pluginValidator) SetSecuritySettings(allowUnsigned bool, trustedPaths []string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.allowUnsigned = allowUnsigned
	if len(trustedPaths) > 0 {
		v.trustedPaths = trustedPaths
	}
}

// isValidPluginName validates plugin name format
func isValidPluginName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
		return false
	}

	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// isValidVersion validates version format (simple semantic versioning)
func isValidVersion(version string) bool {
	if len(version) == 0 {
		return false
	}

	// Simple check for semantic versioning pattern (X.Y.Z)
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}

	return true
}