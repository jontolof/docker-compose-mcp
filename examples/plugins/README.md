# Example Plugins

This directory contains example plugins that demonstrate the plugin architecture capabilities of docker-compose-mcp.

## Available Plugins

### 1. Workflow Automation Plugin (`workflow-plugin`)

**Purpose**: Automated workflow execution for Docker Compose operations

**Features**:
- Pre-defined workflows (CI/CD pipeline, dev setup)
- Custom workflow creation and execution
- Event-triggered workflows
- Step-by-step execution with error handling

**Tools Provided**:
- `workflow_execute` - Execute a predefined workflow
- `workflow_list` - List available workflows
- `workflow_create` - Create a new workflow
- `workflow_status` - Get workflow execution status

**Example Usage**:
```bash
# List available workflows
./docker-compose-mcp call workflow_list

# Execute CI/CD pipeline
./docker-compose-mcp call workflow_execute '{"workflow": "ci-cd"}'

# Get workflow status
./docker-compose-mcp call workflow_status '{"execution_id": "exec_12345"}'
```

### 2. Monitoring & Observability Plugin (`monitoring-plugin`)

**Purpose**: Advanced monitoring and observability for Docker Compose services

**Features**:
- Real-time metrics collection
- Service health monitoring
- Alert generation and management
- Dashboard data generation
- Performance tracking

**Tools Provided**:
- `monitor_metrics` - Get service metrics and performance data
- `monitor_alerts` - List and manage monitoring alerts
- `monitor_health` - Get comprehensive health status
- `monitor_dashboard` - Generate monitoring dashboard data
- `monitor_alert_acknowledge` - Acknowledge monitoring alerts

**Example Usage**:
```bash
# Get metrics for all services
./docker-compose-mcp call monitor_metrics

# Get health status
./docker-compose-mcp call monitor_health '{"detailed": true}'

# List active alerts
./docker-compose-mcp call monitor_alerts '{"severity": "critical"}'
```

### 3. Third-party Integrations Plugin (`integration-plugin`)

**Purpose**: Third-party integrations for notifications, webhooks, and external services

**Features**:
- Slack notifications
- Generic webhook support
- GitHub integration
- Event-driven notifications
- Configurable integrations

**Tools Provided**:
- `integration_notify` - Send notifications to integrated services
- `integration_webhook` - Send webhook to external services
- `integration_list` - List configured integrations
- `integration_configure` - Configure an integration
- `integration_test` - Test an integration connection

**Example Usage**:
```bash
# List integrations
./docker-compose-mcp call integration_list

# Configure Slack integration
./docker-compose-mcp call integration_configure '{
  "integration": "slack", 
  "config": {
    "webhook_url": "https://hooks.slack.com/...",
    "channel": "#deployments",
    "enabled": true
  }
}'

# Send notification
./docker-compose-mcp call integration_notify '{
  "integration": "slack",
  "title": "Deployment Complete", 
  "message": "Application deployed successfully",
  "severity": "info"
}'
```

## Building the Plugins

### Prerequisites
- Go 1.19+ with plugin support
- Docker Compose MCP project dependencies

### Build Instructions

1. **Run the build script**:
   ```bash
   cd examples/plugins
   ./build.sh
   ```

2. **Manual build** (alternative):
   ```bash
   # Workflow plugin
   cd workflow-plugin
   go build -buildmode=plugin -o ../bin/workflow-plugin.so main.go
   
   # Monitoring plugin
   cd ../monitoring-plugin  
   go build -buildmode=plugin -o ../bin/monitoring-plugin.so main.go
   
   # Integration plugin
   cd ../integration-plugin
   go build -buildmode=plugin -o ../bin/integration-plugin.so main.go
   ```

### Installation

1. **Copy plugins to plugin directory**:
   ```bash
   mkdir -p ~/.docker-compose-mcp/plugins
   cp examples/plugins/bin/*.so ~/.docker-compose-mcp/plugins/
   ```

2. **Load plugins using MCP tools**:
   ```bash
   # List available plugins
   ./docker-compose-mcp call plugin_list '{"status": "available"}'
   
   # Load a specific plugin
   ./docker-compose-mcp call plugin_load '{"name": "workflow-automation"}'
   
   # Verify plugin is loaded
   ./docker-compose-mcp call plugin_list '{"status": "loaded"}'
   ```

## Plugin Development Guide

### Creating a New Plugin

1. **Implement the Plugin interface**:
   ```go
   package main
   
   import (
       "github.com/jontolof/docker-compose-mcp/internal/plugin"
       "github.com/jontolof/docker-compose-mcp/internal/mcp"
   )
   
   type MyPlugin struct {
       config plugin.Config
   }
   
   func NewPlugin() plugin.Plugin {
       return &MyPlugin{}
   }
   
   func (p *MyPlugin) Info() plugin.PluginInfo {
       return plugin.PluginInfo{
           Name:        "my-plugin",
           Version:     "1.0.0", 
           Description: "My custom plugin",
           // ... other fields
       }
   }
   
   func (p *MyPlugin) Tools() []mcp.Tool {
       return []mcp.Tool{
           // Define your MCP tools here
       }
   }
   
   // Implement other required methods...
   ```

2. **Build as a plugin**:
   ```bash
   go build -buildmode=plugin -o my-plugin.so main.go
   ```

3. **Test and load**:
   ```bash
   cp my-plugin.so ~/.docker-compose-mcp/plugins/
   ./docker-compose-mcp call plugin_load '{"name": "my-plugin"}'
   ```

### Plugin Architecture

- **Interface-based**: All plugins implement the `plugin.Plugin` interface
- **Event-driven**: Plugins can register event hooks for system events
- **Configuration**: Plugins receive configuration through the Config structure
- **Lifecycle**: Plugins have Initialize, Cleanup, and Health methods
- **Tools**: Plugins provide MCP tools that extend docker-compose-mcp functionality

### Best Practices

1. **Error Handling**: Always handle errors gracefully and provide meaningful messages
2. **Health Checks**: Implement proper health checking for monitoring
3. **Configuration**: Use the plugin configuration system for settings
4. **Event Hooks**: Register event hooks to respond to system events
5. **Resource Cleanup**: Properly clean up resources in the Cleanup method
6. **Dependencies**: Clearly specify plugin dependencies
7. **Documentation**: Document your plugin's tools and configuration options

## Troubleshooting

### Common Issues

1. **Plugin not loading**: Check that the .so file is built for the correct architecture
2. **Missing NewPlugin function**: Ensure your plugin exports a `NewPlugin() plugin.Plugin` function
3. **Dependencies not found**: Verify all required dependencies are available
4. **Permission errors**: Check file permissions on plugin files

### Debug Information

```bash
# Get plugin information
./docker-compose-mcp call plugin_info '{"name": "my-plugin"}'

# Check plugin health
./docker-compose-mcp call plugin_health '{"name": "my-plugin"}'

# List plugin tools
./docker-compose-mcp call plugin_tools '{"plugin": "my-plugin"}'
```

## Contributing

When creating new example plugins:

1. Follow the existing plugin structure
2. Include comprehensive documentation
3. Add build configuration to build.sh
4. Provide usage examples
5. Test thoroughly before submitting

## License

These example plugins are provided under the same license as the main docker-compose-mcp project.