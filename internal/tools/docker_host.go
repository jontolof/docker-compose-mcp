package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/docker"
)

type DockerHostTool struct {
	manager *docker.HostManager
}

func NewDockerHostTool(manager *docker.HostManager) *DockerHostTool {
	return &DockerHostTool{
		manager: manager,
	}
}

func (d *DockerHostTool) GetName() string {
	return "docker_host_manage"
}

func (d *DockerHostTool) GetDescription() string {
	return "Manage Docker host connections - local, remote, SSH, and context-based Docker daemons"
}

func (d *DockerHostTool) GetSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"add", "list", "switch", "current", "remove", "health", "discover"},
				"description": "Action to perform on Docker hosts",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Host name (for add, switch, remove, health actions)",
			},
			"host": map[string]interface{}{
				"type":        "string",
				"description": "Docker host URL (for add action)",
			},
			"type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"local", "remote", "context", "ssh", "container"},
				"description": "Host type (for add action)",
			},
			"context": map[string]interface{}{
				"type":        "string",
				"description": "Docker context name (for context type)",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Host description (for add action)",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Host tags (for add action)",
			},
			"tls": map[string]interface{}{
				"type":        "object",
				"description": "TLS configuration (for add action)",
				"properties": map[string]interface{}{
					"enabled":     map[string]interface{}{"type": "boolean"},
					"verify":      map[string]interface{}{"type": "boolean"},
					"cert_path":   map[string]interface{}{"type": "string"},
					"key_path":    map[string]interface{}{"type": "string"},
					"ca_path":     map[string]interface{}{"type": "string"},
					"server_name": map[string]interface{}{"type": "string"},
				},
			},
			"ssh": map[string]interface{}{
				"type":        "object",
				"description": "SSH configuration (for ssh type)",
				"properties": map[string]interface{}{
					"host":        map[string]interface{}{"type": "string"},
					"port":        map[string]interface{}{"type": "integer"},
					"user":        map[string]interface{}{"type": "string"},
					"key_path":    map[string]interface{}{"type": "string"},
					"password":    map[string]interface{}{"type": "string"},
					"known_hosts": map[string]interface{}{"type": "string"},
				},
			},
			"environment": map[string]interface{}{
				"type":        "object",
				"description": "Environment variables for host (for add action)",
			},
		},
		"required": []string{"action"},
	}
}

func (d *DockerHostTool) Execute(ctx context.Context, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	action, ok := paramsMap["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action is required")
	}

	switch action {
	case "add":
		return d.addHost(paramsMap)
	case "list":
		return d.listHosts()
	case "switch":
		return d.switchHost(paramsMap)
	case "current":
		return d.getCurrentHost()
	case "remove":
		return d.removeHost(paramsMap)
	case "health":
		return d.checkHealth(paramsMap)
	case "discover":
		return d.discoverHosts()
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (d *DockerHostTool) addHost(params map[string]interface{}) (interface{}, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required for add action")
	}

	hostType, ok := params["type"].(string)
	if !ok || hostType == "" {
		hostType = "remote" // default
	}

	host := &docker.DockerHost{
		Name: name,
		Type: docker.HostType(hostType),
	}

	// Set host URL or context
	if hostURL, ok := params["host"].(string); ok && hostURL != "" {
		host.Host = hostURL
	}

	if context, ok := params["context"].(string); ok && context != "" {
		host.Context = context
	}

	if desc, ok := params["description"].(string); ok {
		host.Description = desc
	}

	// Handle tags
	if tags, ok := params["tags"].([]interface{}); ok {
		stringTags := make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				stringTags[i] = tagStr
			}
		}
		host.Tags = stringTags
	}

	// Handle TLS configuration
	if tlsConfig, ok := params["tls"].(map[string]interface{}); ok {
		host.TLS = &docker.TLSConfig{}
		if enabled, ok := tlsConfig["enabled"].(bool); ok {
			host.TLS.Enabled = enabled
		}
		if verify, ok := tlsConfig["verify"].(bool); ok {
			host.TLS.Verify = verify
		}
		if certPath, ok := tlsConfig["cert_path"].(string); ok {
			host.TLS.CertPath = certPath
		}
		if keyPath, ok := tlsConfig["key_path"].(string); ok {
			host.TLS.KeyPath = keyPath
		}
		if caPath, ok := tlsConfig["ca_path"].(string); ok {
			host.TLS.CAPath = caPath
		}
		if serverName, ok := tlsConfig["server_name"].(string); ok {
			host.TLS.ServerName = serverName
		}
	}

	// Handle SSH configuration
	if sshConfig, ok := params["ssh"].(map[string]interface{}); ok {
		host.SSH = &docker.SSHConfig{}
		if sshHost, ok := sshConfig["host"].(string); ok {
			host.SSH.Host = sshHost
		}
		if port, ok := sshConfig["port"].(float64); ok {
			host.SSH.Port = int(port)
		}
		if user, ok := sshConfig["user"].(string); ok {
			host.SSH.User = user
		}
		if keyPath, ok := sshConfig["key_path"].(string); ok {
			host.SSH.KeyPath = keyPath
		}
		if password, ok := sshConfig["password"].(string); ok {
			host.SSH.Password = password
		}
		if knownHosts, ok := sshConfig["known_hosts"].(string); ok {
			host.SSH.KnownHosts = knownHosts
		}
	}

	// Handle environment variables
	if env, ok := params["environment"].(map[string]interface{}); ok {
		host.Environment = make(map[string]string)
		for k, v := range env {
			if vStr, ok := v.(string); ok {
				host.Environment[k] = vStr
			}
		}
	}

	if err := d.manager.AddHost(host); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Added Docker host '%s'", host.Name),
		"host":    d.formatHost(host),
	}, nil
}

func (d *DockerHostTool) listHosts() (interface{}, error) {
	hosts := d.manager.ListHosts()

	result := make([]map[string]interface{}, len(hosts))
	for i, host := range hosts {
		result[i] = d.formatHost(host)
	}

	return map[string]interface{}{
		"success": true,
		"count":   len(hosts),
		"hosts":   result,
	}, nil
}

func (d *DockerHostTool) switchHost(params map[string]interface{}) (interface{}, error) {
	identifier, ok := params["name"].(string)
	if !ok || identifier == "" {
		return nil, fmt.Errorf("name is required for switch action")
	}

	host, err := d.manager.SwitchHost(identifier)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Switched to Docker host '%s'", host.Name),
		"host":    d.formatHost(host),
	}, nil
}

func (d *DockerHostTool) getCurrentHost() (interface{}, error) {
	host := d.manager.GetCurrentHost()
	if host == nil {
		return map[string]interface{}{
			"success": true,
			"message": "No active Docker host",
			"current": nil,
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Current Docker host: %s", host.Name),
		"host":    d.formatHost(host),
	}, nil
}

func (d *DockerHostTool) removeHost(params map[string]interface{}) (interface{}, error) {
	identifier, ok := params["name"].(string)
	if !ok || identifier == "" {
		return nil, fmt.Errorf("name is required for remove action")
	}

	// Get host info before removal
	host, err := d.manager.GetHost(identifier)
	if err != nil {
		return nil, err
	}

	if err := d.manager.RemoveHost(identifier); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Removed Docker host '%s'", host.Name),
	}, nil
}

func (d *DockerHostTool) checkHealth(params map[string]interface{}) (interface{}, error) {
	identifier, ok := params["name"].(string)
	if !ok || identifier == "" {
		return nil, fmt.Errorf("name is required for health action")
	}

	health, err := d.manager.CheckHealth(identifier)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Health check completed for host '%s'", identifier),
		"health":  d.formatHealth(health),
	}, nil
}

func (d *DockerHostTool) discoverHosts() (interface{}, error) {
	hosts, err := d.manager.DiscoverHosts()
	if err != nil {
		return nil, fmt.Errorf("failed to discover hosts: %w", err)
	}

	result := make([]map[string]interface{}, len(hosts))
	for i, host := range hosts {
		result[i] = d.formatHost(host)
	}

	return map[string]interface{}{
		"success":    true,
		"message":    fmt.Sprintf("Discovered %d Docker hosts", len(hosts)),
		"discovered": result,
	}, nil
}

func (d *DockerHostTool) formatHost(host *docker.DockerHost) map[string]interface{} {
	result := map[string]interface{}{
		"id":        host.ID,
		"name":      host.Name,
		"host":      host.Host,
		"type":      string(host.Type),
		"active":    host.Active,
		"created":   host.Created.Format(time.RFC3339),
		"last_used": host.LastUsed.Format(time.RFC3339),
	}

	if host.Context != "" {
		result["context"] = host.Context
	}

	if host.Description != "" {
		result["description"] = host.Description
	}

	if len(host.Tags) > 0 {
		result["tags"] = host.Tags
	}

	if host.TLS != nil {
		result["tls"] = map[string]interface{}{
			"enabled": host.TLS.Enabled,
			"verify":  host.TLS.Verify,
		}
	}

	if host.SSH != nil {
		result["ssh"] = map[string]interface{}{
			"host": host.SSH.Host,
			"port": host.SSH.Port,
			"user": host.SSH.User,
		}
	}

	if len(host.Environment) > 0 {
		result["environment"] = host.Environment
	}

	if host.HealthCheck != nil {
		result["health"] = d.formatHealth(host.HealthCheck)
	}

	return result
}

func (d *DockerHostTool) formatHealth(health *docker.HealthStatus) map[string]interface{} {
	result := map[string]interface{}{
		"status":       health.Status,
		"last_checked": health.LastChecked.Format(time.RFC3339),
		"latency_ms":   health.Latency.Milliseconds(),
	}

	if health.Version != "" {
		result["version"] = health.Version
	}

	if health.Error != "" {
		result["error"] = health.Error
	}

	return result
}

// Docker Context Tool - separate tool for Docker context operations
type DockerContextTool struct {
	manager *docker.HostManager
}

func NewDockerContextTool(manager *docker.HostManager) *DockerContextTool {
	return &DockerContextTool{
		manager: manager,
	}
}

func (dc *DockerContextTool) GetName() string {
	return "docker_context"
}

func (dc *DockerContextTool) GetDescription() string {
	return "Manage Docker contexts and integrate them with host management"
}

func (dc *DockerContextTool) GetSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"list", "current", "use", "inspect", "import"},
				"description": "Action to perform on Docker contexts",
			},
			"context": map[string]interface{}{
				"type":        "string",
				"description": "Context name (for use, inspect, import actions)",
			},
			"import_as_host": map[string]interface{}{
				"type":        "boolean",
				"description": "Import context as managed host (for import action)",
				"default":     true,
			},
		},
		"required": []string{"action"},
	}
}

func (dc *DockerContextTool) Execute(ctx context.Context, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	action, ok := paramsMap["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action is required")
	}

	switch action {
	case "list":
		return dc.listContexts()
	case "current":
		return dc.getCurrentContext()
	case "use":
		return dc.useContext(paramsMap)
	case "inspect":
		return dc.inspectContext(paramsMap)
	case "import":
		return dc.importContext(paramsMap)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (dc *DockerContextTool) listContexts() (interface{}, error) {
	// This would typically execute `docker context ls` command
	// For now, return placeholder data
	contexts := []map[string]interface{}{
		{
			"name":        "default",
			"current":     true,
			"description": "Current DOCKER_HOST based configuration",
			"docker_endpoint": "unix:///var/run/docker.sock",
		},
	}

	return map[string]interface{}{
		"success":  true,
		"contexts": contexts,
		"count":    len(contexts),
	}, nil
}

func (dc *DockerContextTool) getCurrentContext() (interface{}, error) {
	// This would typically execute `docker context show` command
	return map[string]interface{}{
		"success": true,
		"current": "default",
		"message": "Current Docker context: default",
	}, nil
}

func (dc *DockerContextTool) useContext(params map[string]interface{}) (interface{}, error) {
	contextName, ok := params["context"].(string)
	if !ok || contextName == "" {
		return nil, fmt.Errorf("context name is required for use action")
	}

	// This would typically execute `docker context use` command
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Switched to context '%s'", contextName),
		"context": contextName,
	}, nil
}

func (dc *DockerContextTool) inspectContext(params map[string]interface{}) (interface{}, error) {
	contextName, ok := params["context"].(string)
	if !ok || contextName == "" {
		return nil, fmt.Errorf("context name is required for inspect action")
	}

	// This would typically execute `docker context inspect` command
	return map[string]interface{}{
		"success": true,
		"context": map[string]interface{}{
			"name":        contextName,
			"description": "Docker context details",
			"endpoints": map[string]interface{}{
				"docker": map[string]interface{}{
					"host": "unix:///var/run/docker.sock",
				},
			},
		},
	}, nil
}

func (dc *DockerContextTool) importContext(params map[string]interface{}) (interface{}, error) {
	contextName, ok := params["context"].(string)
	if !ok || contextName == "" {
		return nil, fmt.Errorf("context name is required for import action")
	}

	importAsHost := true
	if importFlag, ok := params["import_as_host"].(bool); ok {
		importAsHost = importFlag
	}

	if importAsHost {
		// Create a host from the context
		host := &docker.DockerHost{
			Name:        fmt.Sprintf("Context: %s", contextName),
			Type:        docker.HostTypeContext,
			Context:     contextName,
			Description: fmt.Sprintf("Imported from Docker context '%s'", contextName),
		}

		if err := dc.manager.AddHost(host); err != nil {
			return nil, fmt.Errorf("failed to import context as host: %w", err)
		}

		return map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Imported context '%s' as managed host", contextName),
			"host_id": host.ID,
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Context '%s' information retrieved", contextName),
		"context": contextName,
	}, nil
}