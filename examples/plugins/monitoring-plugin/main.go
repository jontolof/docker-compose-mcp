package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/mcp"
	"github.com/jontolof/docker-compose-mcp/internal/plugin"
)

// MonitoringPlugin demonstrates monitoring and observability features
type MonitoringPlugin struct {
	config  plugin.Config
	metrics map[string]*ServiceMetrics
	alerts  []Alert
	mu      sync.RWMutex
}

// ServiceMetrics tracks metrics for a service
type ServiceMetrics struct {
	ServiceName   string            `json:"serviceName"`
	CPU           float64           `json:"cpu"`
	Memory        int64             `json:"memory"`
	NetworkRx     int64             `json:"networkRx"`
	NetworkTx     int64             `json:"networkTx"`
	RestartCount  int               `json:"restartCount"`
	Uptime        time.Duration     `json:"uptime"`
	HealthChecks  int               `json:"healthChecks"`
	LastUpdated   time.Time         `json:"lastUpdated"`
	CustomMetrics map[string]float64 `json:"customMetrics"`
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "threshold", "anomaly", "health"
	Severity    string                 `json:"severity"` // "info", "warning", "critical"
	Service     string                 `json:"service"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Acknowledged bool                  `json:"acknowledged"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewPlugin creates a new monitoring plugin instance
func NewPlugin() plugin.Plugin {
	return &MonitoringPlugin{
		metrics: make(map[string]*ServiceMetrics),
		alerts:  make([]Alert, 0),
	}
}

// Info returns plugin metadata
func (mp *MonitoringPlugin) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "monitoring-observability",
		Version:     "1.0.0",
		Description: "Advanced monitoring and observability for Docker Compose services",
		Author:      "Docker Compose MCP Team",
		License:     "MIT",
		Tags:        []string{"monitoring", "observability", "metrics", "alerts"},
		Dependencies: []plugin.Dependency{
			{
				Name:    "docker",
				Version: "20.0.0",
				Type:    "binary",
			},
		},
		MinVersion: "1.0.0",
	}
}

// Initialize initializes the plugin
func (mp *MonitoringPlugin) Initialize(ctx context.Context, config plugin.Config) error {
	mp.config = config
	
	// Start metrics collection
	go mp.startMetricsCollection(ctx)
	
	// Start alert processing
	go mp.startAlertProcessing(ctx)
	
	return nil
}

// Tools returns the MCP tools provided by this plugin
func (mp *MonitoringPlugin) Tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "monitor_metrics",
			Description: "Get service metrics and performance data",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"service": {Type: "string"},
					"timeframe": {Type: "string"},
				},
			},
			Handler: mp.getMetrics,
		},
		{
			Name:        "monitor_alerts",
			Description: "List and manage monitoring alerts",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"severity": {Type: "string"},
					"service":  {Type: "string"},
					"limit":    {Type: "number"},
				},
			},
			Handler: mp.getAlerts,
		},
		{
			Name:        "monitor_health",
			Description: "Get comprehensive health status of all services",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"detailed": {Type: "boolean"},
				},
			},
			Handler: mp.getHealthStatus,
		},
		{
			Name:        "monitor_dashboard",
			Description: "Generate monitoring dashboard data",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"view": {Type: "string"}, // "overview", "detailed", "alerts"
				},
			},
			Handler: mp.getDashboard,
		},
		{
			Name:        "monitor_alert_acknowledge",
			Description: "Acknowledge monitoring alerts",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"alert_id": {Type: "string"},
				},
				Required: []string{"alert_id"},
			},
			Handler: mp.acknowledgeAlert,
		},
	}
}

// Hooks returns event hooks provided by this plugin
func (mp *MonitoringPlugin) Hooks() []plugin.Hook {
	return []plugin.Hook{
		{
			Event:   plugin.EventServiceStart,
			Handler: mp.onServiceStart,
		},
		{
			Event:   plugin.EventServiceStop,
			Handler: mp.onServiceStop,
		},
		{
			Event:   plugin.EventError,
			Handler: mp.onError,
		},
	}
}

// Cleanup is called when the plugin is unloaded
func (mp *MonitoringPlugin) Cleanup() error {
	return nil
}

// Health performs a health check on the plugin
func (mp *MonitoringPlugin) Health() plugin.HealthStatus {
	mp.mu.RLock()
	metricsCount := len(mp.metrics)
	alertsCount := len(mp.alerts)
	mp.mu.RUnlock()

	return plugin.HealthStatus{
		Status:  "healthy",
		Message: "Monitoring plugin is collecting metrics",
		Details: map[string]interface{}{
			"tracked_services": metricsCount,
			"active_alerts":    alertsCount,
		},
		LastCheck: time.Now(),
	}
}

// getMetrics returns service metrics
func (mp *MonitoringPlugin) getMetrics(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	serviceName, _ := paramsMap["service"].(string)
	timeframe, _ := paramsMap["timeframe"].(string)

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if serviceName != "" {
		// Return metrics for specific service
		if metrics, exists := mp.metrics[serviceName]; exists {
			return map[string]interface{}{
				"service":   serviceName,
				"timeframe": timeframe,
				"metrics":   metrics,
			}, nil
		}
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// Return metrics for all services
	return map[string]interface{}{
		"timeframe": timeframe,
		"services":  mp.metrics,
		"summary": map[string]interface{}{
			"total_services": len(mp.metrics),
			"timestamp":     time.Now(),
		},
	}, nil
}

// getAlerts returns monitoring alerts
func (mp *MonitoringPlugin) getAlerts(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	severity, _ := paramsMap["severity"].(string)
	service, _ := paramsMap["service"].(string)
	limit, _ := paramsMap["limit"].(float64)

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	filteredAlerts := make([]Alert, 0)
	for _, alert := range mp.alerts {
		if severity != "" && alert.Severity != severity {
			continue
		}
		if service != "" && alert.Service != service {
			continue
		}
		filteredAlerts = append(filteredAlerts, alert)
		
		if limit > 0 && len(filteredAlerts) >= int(limit) {
			break
		}
	}

	return map[string]interface{}{
		"alerts": filteredAlerts,
		"total":  len(filteredAlerts),
		"filters": map[string]interface{}{
			"severity": severity,
			"service":  service,
		},
	}, nil
}

// getHealthStatus returns comprehensive health status
func (mp *MonitoringPlugin) getHealthStatus(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	detailed, _ := paramsMap["detailed"].(bool)

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	healthStatus := map[string]interface{}{
		"overall_status": "healthy",
		"timestamp":     time.Now(),
		"services":      make(map[string]interface{}),
	}

	for serviceName, metrics := range mp.metrics {
		serviceHealth := mp.calculateServiceHealth(metrics)
		
		if detailed {
			healthStatus["services"].(map[string]interface{})[serviceName] = map[string]interface{}{
				"status":      serviceHealth,
				"metrics":     metrics,
				"last_check":  metrics.LastUpdated,
			}
		} else {
			healthStatus["services"].(map[string]interface{})[serviceName] = serviceHealth
		}
	}

	return healthStatus, nil
}

// getDashboard generates dashboard data
func (mp *MonitoringPlugin) getDashboard(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	view, _ := paramsMap["view"].(string)
	if view == "" {
		view = "overview"
	}

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	dashboard := map[string]interface{}{
		"view":      view,
		"timestamp": time.Now(),
	}

	switch view {
	case "overview":
		dashboard["data"] = mp.generateOverviewDashboard()
	case "detailed":
		dashboard["data"] = mp.generateDetailedDashboard()
	case "alerts":
		dashboard["data"] = mp.generateAlertsDashboard()
	default:
		return nil, fmt.Errorf("unknown dashboard view: %s", view)
	}

	return dashboard, nil
}

// acknowledgeAlert acknowledges a monitoring alert
func (mp *MonitoringPlugin) acknowledgeAlert(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	alertID, ok := paramsMap["alert_id"].(string)
	if !ok {
		return nil, fmt.Errorf("alert_id is required")
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	for i := range mp.alerts {
		if mp.alerts[i].ID == alertID {
			mp.alerts[i].Acknowledged = true
			return map[string]interface{}{
				"message":  "Alert acknowledged",
				"alert_id": alertID,
			}, nil
		}
	}

	return nil, fmt.Errorf("alert %s not found", alertID)
}

// Event handlers
func (mp *MonitoringPlugin) onServiceStart(ctx context.Context, event plugin.Event) error {
	serviceName, _ := event.Data["service"].(string)
	
	mp.mu.Lock()
	if _, exists := mp.metrics[serviceName]; !exists {
		mp.metrics[serviceName] = &ServiceMetrics{
			ServiceName:   serviceName,
			CustomMetrics: make(map[string]float64),
			LastUpdated:   time.Now(),
		}
	}
	mp.mu.Unlock()

	mp.addAlert(Alert{
		ID:        fmt.Sprintf("start_%s_%d", serviceName, time.Now().Unix()),
		Type:      "health",
		Severity:  "info",
		Service:   serviceName,
		Message:   fmt.Sprintf("Service %s started", serviceName),
		Timestamp: time.Now(),
		Metadata:  event.Data,
	})

	return nil
}

func (mp *MonitoringPlugin) onServiceStop(ctx context.Context, event plugin.Event) error {
	serviceName, _ := event.Data["service"].(string)

	mp.addAlert(Alert{
		ID:        fmt.Sprintf("stop_%s_%d", serviceName, time.Now().Unix()),
		Type:      "health",
		Severity:  "warning",
		Service:   serviceName,
		Message:   fmt.Sprintf("Service %s stopped", serviceName),
		Timestamp: time.Now(),
		Metadata:  event.Data,
	})

	return nil
}

func (mp *MonitoringPlugin) onError(ctx context.Context, event plugin.Event) error {
	serviceName, _ := event.Data["service"].(string)
	errorMsg, _ := event.Data["error"].(string)

	mp.addAlert(Alert{
		ID:        fmt.Sprintf("error_%d", time.Now().Unix()),
		Type:      "threshold",
		Severity:  "critical",
		Service:   serviceName,
		Message:   fmt.Sprintf("Error in service %s: %s", serviceName, errorMsg),
		Timestamp: time.Now(),
		Metadata:  event.Data,
	})

	return nil
}

// Helper methods
func (mp *MonitoringPlugin) startMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mp.collectMetrics()
		case <-ctx.Done():
			return
		}
	}
}

func (mp *MonitoringPlugin) startAlertProcessing(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mp.processThresholdAlerts()
		case <-ctx.Done():
			return
		}
	}
}

func (mp *MonitoringPlugin) collectMetrics() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Simulate metric collection (in real implementation, would query Docker)
	for serviceName, metrics := range mp.metrics {
		// Simulate metrics updates
		metrics.CPU = 25.0 + (float64(time.Now().Unix()%10) * 5.0)
		metrics.Memory = 512 + (time.Now().Unix()%500)*1024*1024
		metrics.NetworkRx += 1024
		metrics.NetworkTx += 512
		metrics.HealthChecks++
		metrics.LastUpdated = time.Now()
		
		// Add custom metrics
		metrics.CustomMetrics["response_time"] = 100.0 + (float64(time.Now().Unix()%20) * 10.0)
		metrics.CustomMetrics["request_rate"] = 50.0 + (float64(time.Now().Unix()%30) * 2.0)
	}
}

func (mp *MonitoringPlugin) processThresholdAlerts() {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	for serviceName, metrics := range mp.metrics {
		// Check CPU threshold
		if metrics.CPU > 80.0 {
			mp.addAlert(Alert{
				ID:        fmt.Sprintf("cpu_high_%s_%d", serviceName, time.Now().Unix()),
				Type:      "threshold",
				Severity:  "warning",
				Service:   serviceName,
				Message:   fmt.Sprintf("High CPU usage: %.1f%%", metrics.CPU),
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"metric": "cpu",
					"value":  metrics.CPU,
					"threshold": 80.0,
				},
			})
		}

		// Check memory threshold
		if metrics.Memory > 1024*1024*1024 { // 1GB
			mp.addAlert(Alert{
				ID:        fmt.Sprintf("memory_high_%s_%d", serviceName, time.Now().Unix()),
				Type:      "threshold",
				Severity:  "warning",
				Service:   serviceName,
				Message:   fmt.Sprintf("High memory usage: %d bytes", metrics.Memory),
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"metric": "memory",
					"value":  metrics.Memory,
					"threshold": 1024*1024*1024,
				},
			})
		}
	}
}

func (mp *MonitoringPlugin) addAlert(alert Alert) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.alerts = append(mp.alerts, alert)

	// Keep only recent alerts (last 100)
	if len(mp.alerts) > 100 {
		mp.alerts = mp.alerts[len(mp.alerts)-100:]
	}
}

func (mp *MonitoringPlugin) calculateServiceHealth(metrics *ServiceMetrics) string {
	if metrics.CPU > 90 || metrics.Memory > 2*1024*1024*1024 {
		return "critical"
	}
	if metrics.CPU > 70 || metrics.Memory > 1024*1024*1024 {
		return "warning"
	}
	return "healthy"
}

func (mp *MonitoringPlugin) generateOverviewDashboard() map[string]interface{} {
	totalServices := len(mp.metrics)
	healthyServices := 0
	warningServices := 0
	criticalServices := 0

	for _, metrics := range mp.metrics {
		health := mp.calculateServiceHealth(metrics)
		switch health {
		case "healthy":
			healthyServices++
		case "warning":
			warningServices++
		case "critical":
			criticalServices++
		}
	}

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"total_services":    totalServices,
			"healthy_services":  healthyServices,
			"warning_services":  warningServices,
			"critical_services": criticalServices,
		},
		"alerts": map[string]interface{}{
			"total":    len(mp.alerts),
			"critical": mp.countAlertsBySeverity("critical"),
			"warning":  mp.countAlertsBySeverity("warning"),
		},
	}
}

func (mp *MonitoringPlugin) generateDetailedDashboard() map[string]interface{} {
	return map[string]interface{}{
		"services": mp.metrics,
		"alerts":   mp.alerts,
	}
}

func (mp *MonitoringPlugin) generateAlertsDashboard() map[string]interface{} {
	return map[string]interface{}{
		"alerts": mp.alerts,
		"summary": map[string]interface{}{
			"total":    len(mp.alerts),
			"critical": mp.countAlertsBySeverity("critical"),
			"warning":  mp.countAlertsBySeverity("warning"),
			"info":     mp.countAlertsBySeverity("info"),
		},
	}
}

func (mp *MonitoringPlugin) countAlertsBySeverity(severity string) int {
	count := 0
	for _, alert := range mp.alerts {
		if alert.Severity == severity {
			count++
		}
	}
	return count
}

func main() {
	// Plugin entry point for compilation
}