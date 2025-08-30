package tools

import (
	"fmt"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/logging"
	"github.com/jontolof/docker-compose-mcp/internal/mcp"
	"github.com/jontolof/docker-compose-mcp/internal/plugin"
)

// CreatePluginTools creates MCP tools for plugin management
func CreatePluginTools(manager plugin.Manager, logger *logging.Logger) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "plugin_list",
			Description: "List all available and loaded plugins",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"status": {Type: "string"}, // "all", "loaded", "available"
				},
			},
			Handler: func(params interface{}) (interface{}, error) {
				return handlePluginList(manager, params)
			},
		},
		{
			Name:        "plugin_load",
			Description: "Load a plugin by name",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"name": {Type: "string"},
				},
				Required: []string{"name"},
			},
			Handler: func(params interface{}) (interface{}, error) {
				return handlePluginLoad(manager, params)
			},
		},
		{
			Name:        "plugin_unload",
			Description: "Unload a plugin by name",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"name": {Type: "string"},
				},
				Required: []string{"name"},
			},
			Handler: func(params interface{}) (interface{}, error) {
				return handlePluginUnload(manager, params)
			},
		},
		{
			Name:        "plugin_info",
			Description: "Get detailed information about a plugin",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"name": {Type: "string"},
				},
				Required: []string{"name"},
			},
			Handler: func(params interface{}) (interface{}, error) {
				return handlePluginInfo(manager, params)
			},
		},
		{
			Name:        "plugin_health",
			Description: "Get health status of all plugins",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"name": {Type: "string"}, // Optional: specific plugin name
				},
			},
			Handler: func(params interface{}) (interface{}, error) {
				return handlePluginHealth(manager, params)
			},
		},
		{
			Name:        "plugin_tools",
			Description: "List tools provided by plugins",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"plugin": {Type: "string"}, // Optional: specific plugin name
				},
			},
			Handler: func(params interface{}) (interface{}, error) {
				return handlePluginTools(manager, params)
			},
		},
		{
			Name:        "plugin_reload",
			Description: "Reload all plugins or a specific plugin",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"name": {Type: "string"}, // Optional: specific plugin name
				},
			},
			Handler: func(params interface{}) (interface{}, error) {
				return handlePluginReload(manager, params)
			},
		},
		{
			Name:        "plugin_events",
			Description: "Fire a custom event to plugin event handlers",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"event": {Type: "string"},
					"data":  {Type: "object"},
				},
				Required: []string{"event"},
			},
			Handler: func(params interface{}) (interface{}, error) {
				return handlePluginEvents(manager, params)
			},
		},
	}
}

// handlePluginList handles the plugin_list tool
func handlePluginList(manager plugin.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	status, _ := paramsMap["status"].(string)
	if status == "" {
		status = "all"
	}

	result := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now(),
	}

	switch status {
	case "loaded":
		loadedPlugins := manager.List()
		plugins := make([]map[string]interface{}, len(loadedPlugins))
		for i, p := range loadedPlugins {
			info := p.Info()
			plugins[i] = map[string]interface{}{
				"name":        info.Name,
				"version":     info.Version,
				"description": info.Description,
				"status":      "loaded",
				"tools":       len(p.Tools()),
				"hooks":       len(p.Hooks()),
			}
		}
		result["plugins"] = plugins
		result["total"] = len(plugins)

	case "available":
		descriptors, err := manager.Discover()
		if err != nil {
			return nil, fmt.Errorf("failed to discover plugins: %w", err)
		}
		
		plugins := make([]map[string]interface{}, len(descriptors))
		for i, desc := range descriptors {
			plugins[i] = map[string]interface{}{
				"name":        desc.Name,
				"type":        desc.Type,
				"enabled":     desc.Enabled,
				"path":        desc.Path,
				"status":      "available",
				"version":     desc.Info.Version,
				"description": desc.Info.Description,
			}
		}
		result["plugins"] = plugins
		result["total"] = len(plugins)

	default: // "all"
		loadedPlugins := manager.List()
		descriptors, err := manager.Discover()
		if err != nil {
			return nil, fmt.Errorf("failed to discover plugins: %w", err)
		}

		// Create a map of loaded plugins for quick lookup
		loadedMap := make(map[string]bool)
		for _, p := range loadedPlugins {
			loadedMap[p.Info().Name] = true
		}

		plugins := make([]map[string]interface{}, 0)

		// Add loaded plugins
		for _, p := range loadedPlugins {
			info := p.Info()
			plugins = append(plugins, map[string]interface{}{
				"name":        info.Name,
				"version":     info.Version,
				"description": info.Description,
				"status":      "loaded",
				"tools":       len(p.Tools()),
				"hooks":       len(p.Hooks()),
			})
		}

		// Add available but not loaded plugins
		for _, desc := range descriptors {
			if !loadedMap[desc.Name] {
				plugins = append(plugins, map[string]interface{}{
					"name":        desc.Name,
					"type":        desc.Type,
					"enabled":     desc.Enabled,
					"path":        desc.Path,
					"status":      "available",
					"version":     desc.Info.Version,
					"description": desc.Info.Description,
				})
			}
		}

		result["plugins"] = plugins
		result["total"] = len(plugins)
	}

	return result, nil
}

// handlePluginLoad handles the plugin_load tool
func handlePluginLoad(manager plugin.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	name, ok := paramsMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin name is required")
	}

	loadedPlugin, err := manager.Load(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin %s: %w", name, err)
	}

	info := loadedPlugin.Info()
	
	return map[string]interface{}{
		"message": fmt.Sprintf("Plugin %s loaded successfully", name),
		"plugin": map[string]interface{}{
			"name":         info.Name,
			"version":      info.Version,
			"description":  info.Description,
			"author":       info.Author,
			"tools":        len(loadedPlugin.Tools()),
			"hooks":        len(loadedPlugin.Hooks()),
			"dependencies": info.Dependencies,
		},
		"timestamp": time.Now(),
	}, nil
}

// handlePluginUnload handles the plugin_unload tool
func handlePluginUnload(manager plugin.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	name, ok := paramsMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin name is required")
	}

	if err := manager.Unload(name); err != nil {
		return nil, fmt.Errorf("failed to unload plugin %s: %w", name, err)
	}

	return map[string]interface{}{
		"message":   fmt.Sprintf("Plugin %s unloaded successfully", name),
		"timestamp": time.Now(),
	}, nil
}

// handlePluginInfo handles the plugin_info tool
func handlePluginInfo(manager plugin.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	name, ok := paramsMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin name is required")
	}

	loadedPlugin, exists := manager.Get(name)
	if !exists {
		// Check if it's available but not loaded
		descriptors, err := manager.Discover()
		if err != nil {
			return nil, fmt.Errorf("failed to discover plugins: %w", err)
		}

		for _, desc := range descriptors {
			if desc.Name == name {
				return map[string]interface{}{
					"plugin": map[string]interface{}{
						"name":        desc.Name,
						"version":     desc.Info.Version,
						"description": desc.Info.Description,
						"author":      desc.Info.Author,
						"type":        desc.Type,
						"enabled":     desc.Enabled,
						"path":        desc.Path,
						"status":      "available",
						"dependencies": desc.Info.Dependencies,
					},
					"timestamp": time.Now(),
				}, nil
			}
		}

		return nil, fmt.Errorf("plugin %s not found", name)
	}

	info := loadedPlugin.Info()
	tools := loadedPlugin.Tools()
	hooks := loadedPlugin.Hooks()
	health := loadedPlugin.Health()

	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name
	}

	hookEvents := make([]string, len(hooks))
	for i, hook := range hooks {
		hookEvents[i] = string(hook.Event)
	}

	return map[string]interface{}{
		"plugin": map[string]interface{}{
			"name":         info.Name,
			"version":      info.Version,
			"description":  info.Description,
			"author":       info.Author,
			"website":      info.Website,
			"license":      info.License,
			"tags":         info.Tags,
			"minVersion":   info.MinVersion,
			"dependencies": info.Dependencies,
			"status":       "loaded",
			"tools":        toolNames,
			"hooks":        hookEvents,
			"health":       health,
		},
		"timestamp": time.Now(),
	}, nil
}

// handlePluginHealth handles the plugin_health tool
func handlePluginHealth(manager plugin.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	name, _ := paramsMap["name"].(string)

	if name != "" {
		// Get health for specific plugin
		loadedPlugin, exists := manager.Get(name)
		if !exists {
			return nil, fmt.Errorf("plugin %s not found or not loaded", name)
		}

		health := loadedPlugin.Health()
		return map[string]interface{}{
			"plugin":    name,
			"health":    health,
			"timestamp": time.Now(),
		}, nil
	}

	// Get health for all plugins
	loadedPlugins := manager.List()
	pluginHealth := make(map[string]interface{})
	
	overallStatus := "healthy"
	for _, p := range loadedPlugins {
		health := p.Health()
		pluginHealth[p.Info().Name] = health
		
		if health.Status == "unhealthy" {
			overallStatus = "unhealthy"
		} else if health.Status == "degraded" && overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
	}

	return map[string]interface{}{
		"overall_status": overallStatus,
		"plugins":        pluginHealth,
		"total":          len(loadedPlugins),
		"timestamp":      time.Now(),
	}, nil
}

// handlePluginTools handles the plugin_tools tool
func handlePluginTools(manager plugin.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	pluginName, _ := paramsMap["plugin"].(string)

	if pluginName != "" {
		// Get tools for specific plugin
		loadedPlugin, exists := manager.Get(pluginName)
		if !exists {
			return nil, fmt.Errorf("plugin %s not found or not loaded", pluginName)
		}

		tools := loadedPlugin.Tools()
		toolList := make([]map[string]interface{}, len(tools))
		
		for i, tool := range tools {
			toolList[i] = map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"schema":      tool.InputSchema,
			}
		}

		return map[string]interface{}{
			"plugin":    pluginName,
			"tools":     toolList,
			"total":     len(toolList),
			"timestamp": time.Now(),
		}, nil
	}

	// Get tools from all plugins
	loadedPlugins := manager.List()
	allTools := make(map[string][]map[string]interface{})
	totalTools := 0
	
	for _, p := range loadedPlugins {
		pluginTools := p.Tools()
		toolList := make([]map[string]interface{}, len(pluginTools))
		
		for i, tool := range pluginTools {
			toolList[i] = map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"schema":      tool.InputSchema,
			}
		}
		
		allTools[p.Info().Name] = toolList
		totalTools += len(toolList)
	}

	return map[string]interface{}{
		"plugins":   allTools,
		"total":     totalTools,
		"timestamp": time.Now(),
	}, nil
}

// handlePluginReload handles the plugin_reload tool
func handlePluginReload(manager plugin.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	name, _ := paramsMap["name"].(string)

	if name != "" {
		// Reload specific plugin
		if err := manager.Unload(name); err != nil {
			return nil, fmt.Errorf("failed to unload plugin %s: %w", name, err)
		}

		_, err := manager.Load(name)
		if err != nil {
			return nil, fmt.Errorf("failed to reload plugin %s: %w", name, err)
		}

		return map[string]interface{}{
			"message":   fmt.Sprintf("Plugin %s reloaded successfully", name),
			"timestamp": time.Now(),
		}, nil
	}

	// Reload all plugins
	if err := manager.Reload(); err != nil {
		return nil, fmt.Errorf("failed to reload plugins: %w", err)
	}

	loadedPlugins := manager.List()
	return map[string]interface{}{
		"message":       "All plugins reloaded successfully",
		"loaded_count":  len(loadedPlugins),
		"timestamp":     time.Now(),
	}, nil
}

// handlePluginEvents handles the plugin_events tool
func handlePluginEvents(manager plugin.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	eventType, ok := paramsMap["event"].(string)
	if !ok {
		return nil, fmt.Errorf("event type is required")
	}

	data, _ := paramsMap["data"].(map[string]interface{})
	if data == nil {
		data = make(map[string]interface{})
	}

	// Fire the event through the manager
	if fireableManager, ok := manager.(interface {
		FireEvent(plugin.EventType, map[string]interface{})
	}); ok {
		fireableManager.FireEvent(plugin.EventType(eventType), data)
	} else {
		return nil, fmt.Errorf("manager does not support event firing")
	}

	return map[string]interface{}{
		"message":   fmt.Sprintf("Event %s fired successfully", eventType),
		"event":     eventType,
		"data":      data,
		"timestamp": time.Now(),
	}, nil
}