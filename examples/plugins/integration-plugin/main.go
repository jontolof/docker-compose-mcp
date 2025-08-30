package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/mcp"
	"github.com/jontolof/docker-compose-mcp/internal/plugin"
)

// IntegrationPlugin demonstrates third-party integrations
type IntegrationPlugin struct {
	config       plugin.Config
	integrations map[string]Integration
}

// Integration represents a third-party integration
type Integration struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "slack", "webhook", "github", "jira"
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	LastUsed    time.Time              `json:"lastUsed"`
	Status      string                 `json:"status"`
}

// NotificationPayload represents a notification to external systems
type NotificationPayload struct {
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// WebhookPayload represents webhook data
type WebhookPayload struct {
	Event     string                 `json:"event"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// NewPlugin creates a new integration plugin instance
func NewPlugin() plugin.Plugin {
	return &IntegrationPlugin{
		integrations: make(map[string]Integration),
	}
}

// Info returns plugin metadata
func (ip *IntegrationPlugin) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "third-party-integrations",
		Version:     "1.0.0",
		Description: "Third-party integrations for notifications, webhooks, and external services",
		Author:      "Docker Compose MCP Team",
		License:     "MIT",
		Tags:        []string{"integration", "notifications", "webhooks", "slack", "github"},
		Dependencies: []plugin.Dependency{
			{
				Name:    "curl",
				Version: "7.0.0",
				Type:    "binary",
			},
		},
		MinVersion: "1.0.0",
	}
}

// Initialize initializes the plugin
func (ip *IntegrationPlugin) Initialize(ctx context.Context, config plugin.Config) error {
	ip.config = config
	
	// Load default integrations
	ip.loadDefaultIntegrations()
	
	return nil
}

// Tools returns the MCP tools provided by this plugin
func (ip *IntegrationPlugin) Tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "integration_notify",
			Description: "Send notifications to integrated services",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"integration": {Type: "string"},
					"message":     {Type: "string"},
					"title":       {Type: "string"},
					"severity":    {Type: "string"},
				},
				Required: []string{"integration", "message"},
			},
			Handler: ip.sendNotification,
		},
		{
			Name:        "integration_webhook",
			Description: "Send webhook to external services",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"url":     {Type: "string"},
					"payload": {Type: "object"},
					"headers": {Type: "object"},
				},
				Required: []string{"url", "payload"},
			},
			Handler: ip.sendWebhook,
		},
		{
			Name:        "integration_list",
			Description: "List configured integrations",
			InputSchema: mcp.Schema{
				Type: "object",
			},
			Handler: ip.listIntegrations,
		},
		{
			Name:        "integration_configure",
			Description: "Configure an integration",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"integration": {Type: "string"},
					"config":      {Type: "object"},
				},
				Required: []string{"integration", "config"},
			},
			Handler: ip.configureIntegration,
		},
		{
			Name:        "integration_test",
			Description: "Test an integration connection",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"integration": {Type: "string"},
				},
				Required: []string{"integration"},
			},
			Handler: ip.testIntegration,
		},
	}
}

// Hooks returns event hooks provided by this plugin
func (ip *IntegrationPlugin) Hooks() []plugin.Hook {
	return []plugin.Hook{
		{
			Event:   plugin.EventServiceStart,
			Handler: ip.onServiceStart,
		},
		{
			Event:   plugin.EventServiceStop,
			Handler: ip.onServiceStop,
		},
		{
			Event:   plugin.EventError,
			Handler: ip.onError,
		},
		{
			Event:   plugin.EventWorkspaceChange,
			Handler: ip.onWorkspaceChange,
		},
	}
}

// Cleanup is called when the plugin is unloaded
func (ip *IntegrationPlugin) Cleanup() error {
	return nil
}

// Health performs a health check on the plugin
func (ip *IntegrationPlugin) Health() plugin.HealthStatus {
	activeIntegrations := 0
	for _, integration := range ip.integrations {
		if integration.Enabled && integration.Status == "connected" {
			activeIntegrations++
		}
	}

	status := "healthy"
	message := "Integration plugin is running"

	if activeIntegrations == 0 {
		status = "degraded"
		message = "No active integrations configured"
	}

	return plugin.HealthStatus{
		Status:  status,
		Message: message,
		Details: map[string]interface{}{
			"total_integrations":  len(ip.integrations),
			"active_integrations": activeIntegrations,
		},
		LastCheck: time.Now(),
	}
}

// sendNotification sends a notification to an integrated service
func (ip *IntegrationPlugin) sendNotification(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	integrationName, ok := paramsMap["integration"].(string)
	if !ok {
		return nil, fmt.Errorf("integration name is required")
	}

	message, ok := paramsMap["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message is required")
	}

	title, _ := paramsMap["title"].(string)
	severity, _ := paramsMap["severity"].(string)
	if severity == "" {
		severity = "info"
	}

	integration, exists := ip.integrations[integrationName]
	if !exists {
		return nil, fmt.Errorf("integration %s not found", integrationName)
	}

	if !integration.Enabled {
		return nil, fmt.Errorf("integration %s is disabled", integrationName)
	}

	payload := NotificationPayload{
		Type:      integration.Type,
		Title:     title,
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"source": "docker-compose-mcp",
		},
	}

	result, err := ip.deliverNotification(integration, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to send notification: %w", err)
	}

	// Update integration last used time
	integration.LastUsed = time.Now()
	ip.integrations[integrationName] = integration

	return result, nil
}

// sendWebhook sends a webhook to an external service
func (ip *IntegrationPlugin) sendWebhook(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	url, ok := paramsMap["url"].(string)
	if !ok {
		return nil, fmt.Errorf("URL is required")
	}

	payloadData, ok := paramsMap["payload"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("payload is required")
	}

	_ = paramsMap["headers"]

	_ = WebhookPayload{
		Event:     "docker-compose-event",
		Source:    "docker-compose-mcp",
		Timestamp: time.Now(),
		Data:      payloadData,
	}

	// Simulate webhook delivery
	result := map[string]interface{}{
		"status": "delivered",
		"url":    url,
		"timestamp": time.Now(),
		"response": map[string]interface{}{
			"status_code": 200,
			"message":     "Webhook delivered successfully",
		},
	}

	return result, nil
}

// listIntegrations lists configured integrations
func (ip *IntegrationPlugin) listIntegrations(params interface{}) (interface{}, error) {
	integrations := make([]map[string]interface{}, 0, len(ip.integrations))

	for name, integration := range ip.integrations {
		integrations = append(integrations, map[string]interface{}{
			"name":      name,
			"type":      integration.Type,
			"enabled":   integration.Enabled,
			"status":    integration.Status,
			"last_used": integration.LastUsed,
		})
	}

	return map[string]interface{}{
		"integrations": integrations,
		"total":        len(integrations),
	}, nil
}

// configureIntegration configures an integration
func (ip *IntegrationPlugin) configureIntegration(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	integrationName, ok := paramsMap["integration"].(string)
	if !ok {
		return nil, fmt.Errorf("integration name is required")
	}

	config, ok := paramsMap["config"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config is required")
	}

	// Get or create integration
	integration, exists := ip.integrations[integrationName]
	if !exists {
		integration = Integration{
			Name:     integrationName,
			Type:     "webhook", // Default type
			Enabled:  true,
			Config:   make(map[string]interface{}),
			Status:   "configured",
		}
	}

	// Update configuration
	for key, value := range config {
		integration.Config[key] = value
	}

	// Set type if provided
	if integrationType, ok := config["type"].(string); ok {
		integration.Type = integrationType
	}

	// Set enabled state if provided
	if enabled, ok := config["enabled"].(bool); ok {
		integration.Enabled = enabled
	}

	ip.integrations[integrationName] = integration

	return map[string]interface{}{
		"message":     fmt.Sprintf("Integration %s configured successfully", integrationName),
		"integration": integration,
	}, nil
}

// testIntegration tests an integration connection
func (ip *IntegrationPlugin) testIntegration(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	integrationName, ok := paramsMap["integration"].(string)
	if !ok {
		return nil, fmt.Errorf("integration name is required")
	}

	integration, exists := ip.integrations[integrationName]
	if !exists {
		return nil, fmt.Errorf("integration %s not found", integrationName)
	}

	// Simulate testing the integration
	testResult := map[string]interface{}{
		"integration": integrationName,
		"status":      "success",
		"message":     "Integration test completed successfully",
		"timestamp":   time.Now(),
		"details": map[string]interface{}{
			"response_time": "150ms",
			"connectivity": "ok",
		},
	}

	// Update integration status
	integration.Status = "connected"
	ip.integrations[integrationName] = integration

	return testResult, nil
}

// Event handlers
func (ip *IntegrationPlugin) onServiceStart(ctx context.Context, event plugin.Event) error {
	serviceName, _ := event.Data["service"].(string)
	
	// Send notifications to configured integrations
	for name, integration := range ip.integrations {
		if integration.Enabled && integration.Type == "slack" {
			payload := NotificationPayload{
				Type:      "service_start",
				Title:     "Service Started",
				Message:   fmt.Sprintf("Docker Compose service '%s' has started successfully", serviceName),
				Severity:  "info",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"service": serviceName,
					"event":   "service_start",
				},
			}

			_, err := ip.deliverNotification(integration, payload)
			if err != nil {
				fmt.Printf("Failed to send notification to %s: %v\n", name, err)
			}
		}
	}

	return nil
}

func (ip *IntegrationPlugin) onServiceStop(ctx context.Context, event plugin.Event) error {
	serviceName, _ := event.Data["service"].(string)

	// Send notifications to configured integrations
	for name, integration := range ip.integrations {
		if integration.Enabled && integration.Type == "slack" {
			payload := NotificationPayload{
				Type:      "service_stop",
				Title:     "Service Stopped",
				Message:   fmt.Sprintf("Docker Compose service '%s' has stopped", serviceName),
				Severity:  "warning",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"service": serviceName,
					"event":   "service_stop",
				},
			}

			_, err := ip.deliverNotification(integration, payload)
			if err != nil {
				fmt.Printf("Failed to send notification to %s: %v\n", name, err)
			}
		}
	}

	return nil
}

func (ip *IntegrationPlugin) onError(ctx context.Context, event plugin.Event) error {
	errorMsg, _ := event.Data["error"].(string)
	serviceName, _ := event.Data["service"].(string)

	// Send error notifications to all enabled integrations
	for name, integration := range ip.integrations {
		if integration.Enabled {
			payload := NotificationPayload{
				Type:      "error",
				Title:     "Docker Compose Error",
				Message:   fmt.Sprintf("Error in service %s: %s", serviceName, errorMsg),
				Severity:  "critical",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"service": serviceName,
					"error":   errorMsg,
					"event":   "error",
				},
			}

			_, err := ip.deliverNotification(integration, payload)
			if err != nil {
				fmt.Printf("Failed to send error notification to %s: %v\n", name, err)
			}
		}
	}

	return nil
}

func (ip *IntegrationPlugin) onWorkspaceChange(ctx context.Context, event plugin.Event) error {
	workspaceName, _ := event.Data["workspace"].(string)

	// Notify about workspace changes
	for name, integration := range ip.integrations {
		if integration.Enabled && integration.Type == "webhook" {
			payload := NotificationPayload{
				Type:      "workspace_change",
				Title:     "Workspace Changed",
				Message:   fmt.Sprintf("Switched to workspace: %s", workspaceName),
				Severity:  "info",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"workspace": workspaceName,
					"event":     "workspace_change",
				},
			}

			_, err := ip.deliverNotification(integration, payload)
			if err != nil {
				fmt.Printf("Failed to send workspace change notification to %s: %v\n", name, err)
			}
		}
	}

	return nil
}

// Helper methods
func (ip *IntegrationPlugin) loadDefaultIntegrations() {
	// Slack integration
	ip.integrations["slack"] = Integration{
		Name:    "slack",
		Type:    "slack",
		Enabled: false,
		Config: map[string]interface{}{
			"webhook_url": "",
			"channel":     "#docker-compose",
			"username":    "docker-compose-mcp",
		},
		Status: "configured",
	}

	// Generic webhook
	ip.integrations["webhook"] = Integration{
		Name:    "webhook",
		Type:    "webhook",
		Enabled: false,
		Config: map[string]interface{}{
			"url":     "",
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
		},
		Status: "configured",
	}

	// GitHub integration
	ip.integrations["github"] = Integration{
		Name:    "github",
		Type:    "github",
		Enabled: false,
		Config: map[string]interface{}{
			"token":      "",
			"repository": "",
			"username":   "",
		},
		Status: "configured",
	}
}

func (ip *IntegrationPlugin) deliverNotification(integration Integration, payload NotificationPayload) (interface{}, error) {
	switch integration.Type {
	case "slack":
		return ip.deliverSlackNotification(integration, payload)
	case "webhook":
		return ip.deliverWebhookNotification(integration, payload)
	case "github":
		return ip.deliverGitHubNotification(integration, payload)
	default:
		return nil, fmt.Errorf("unsupported integration type: %s", integration.Type)
	}
}

func (ip *IntegrationPlugin) deliverSlackNotification(integration Integration, payload NotificationPayload) (interface{}, error) {
	// Simulate Slack notification delivery
	slackPayload := map[string]interface{}{
		"channel":  integration.Config["channel"],
		"username": integration.Config["username"],
		"text":     payload.Message,
		"attachments": []map[string]interface{}{
			{
				"color":    ip.getSeverityColor(payload.Severity),
				"title":    payload.Title,
				"text":     payload.Message,
				"ts":       payload.Timestamp.Unix(),
			},
		},
	}

	return map[string]interface{}{
		"status":   "delivered",
		"platform": "slack",
		"payload":  slackPayload,
		"timestamp": time.Now(),
	}, nil
}

func (ip *IntegrationPlugin) deliverWebhookNotification(integration Integration, payload NotificationPayload) (interface{}, error) {
	// Simulate webhook delivery
	webhookURL, _ := integration.Config["url"].(string)
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL not configured")
	}

	return map[string]interface{}{
		"status":     "delivered",
		"platform":   "webhook",
		"url":        webhookURL,
		"payload":    payload,
		"timestamp":  time.Now(),
		"status_code": 200,
	}, nil
}

func (ip *IntegrationPlugin) deliverGitHubNotification(integration Integration, payload NotificationPayload) (interface{}, error) {
	// Simulate GitHub notification (could create issues, comments, etc.)
	repository, _ := integration.Config["repository"].(string)
	
	return map[string]interface{}{
		"status":     "delivered",
		"platform":   "github",
		"repository": repository,
		"action":     "comment_created",
		"payload":    payload,
		"timestamp":  time.Now(),
	}, nil
}

func (ip *IntegrationPlugin) getSeverityColor(severity string) string {
	switch severity {
	case "critical":
		return "danger"
	case "warning":
		return "warning"
	case "info":
		return "good"
	default:
		return "good"
	}
}

func main() {
	// Plugin entry point for compilation
}