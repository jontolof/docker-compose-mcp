package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jontolof/docker-compose-mcp/internal/compose"
)

type OptimizationTool struct {
	client *compose.Client
}

func NewOptimizationTool(client *compose.Client) *OptimizationTool {
	return &OptimizationTool{client: client}
}

type OptimizationParams struct {
	Action     string `json:"action"`     // "stats", "reset", "cache", "export"
	Format     string `json:"format"`     // "json", "summary"
	Operation  string `json:"operation"`  // Filter by specific operation
}

type OptimizationResult struct {
	Action    string      `json:"action"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Message   string      `json:"message"`
	Timestamp string      `json:"timestamp"`
}

func (t *OptimizationTool) Execute(ctx context.Context, params interface{}) (interface{}, error) {
	var p OptimizationParams
	if err := parseParams(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	switch p.Action {
	case "stats":
		return t.getStats(p.Format, p.Operation)
	case "reset":
		return t.resetStats()
	case "cache":
		return t.getCacheStats()
	case "export":
		return t.exportStats(p.Format)
	default:
		return nil, fmt.Errorf("unknown action: %s", p.Action)
	}
}

func (t *OptimizationTool) getStats(format, operation string) (*OptimizationResult, error) {
	if t.client.GetMetrics() == nil {
		return &OptimizationResult{
			Action:  "stats",
			Success: false,
			Message: "Metrics not enabled for this client",
		}, nil
	}
	
	var data interface{}
	var message string
	
	switch format {
	case "summary":
		message = t.client.GetMetrics().GetSummaryString()
		data = t.client.GetMetrics().GetOverallStats()
	case "json":
		fallthrough
	default:
		if operation != "" {
			// Filter by specific operation
			allOps := t.client.GetMetrics().GetOperationStats()
			for _, op := range allOps {
				if op.Name == operation {
					data = op
					break
				}
			}
			if data == nil {
				return &OptimizationResult{
					Action:  "stats",
					Success: false,
					Message: fmt.Sprintf("No stats found for operation: %s", operation),
				}, nil
			}
		} else {
			data = t.client.GetMetrics().GetDetailedReport()
		}
	}
	
	return &OptimizationResult{
		Action:  "stats",
		Success: true,
		Data:    data,
		Message: message,
	}, nil
}

func (t *OptimizationTool) resetStats() (*OptimizationResult, error) {
	if t.client.GetMetrics() == nil {
		return &OptimizationResult{
			Action:  "reset",
			Success: false,
			Message: "Metrics not enabled for this client",
		}, nil
	}
	
	t.client.GetMetrics().Reset()
	
	return &OptimizationResult{
		Action:  "reset",
		Success: true,
		Message: "All metrics have been reset",
	}, nil
}

func (t *OptimizationTool) getCacheStats() (*OptimizationResult, error) {
	cache := t.client.GetCache()
	if cache == nil {
		return &OptimizationResult{
			Action:  "cache",
			Success: false,
			Message: "Cache not enabled for this client",
		}, nil
	}
	
	stats := cache.Stats()
	
	return &OptimizationResult{
		Action:  "cache",
		Success: true,
		Data:    stats,
		Message: fmt.Sprintf("Cache contains %d entries using %dKB", stats.Entries, stats.TotalSizeKB),
	}, nil
}

func (t *OptimizationTool) exportStats(format string) (*OptimizationResult, error) {
	if t.client.GetMetrics() == nil {
		return &OptimizationResult{
			Action:  "export",
			Success: false,
			Message: "Metrics not enabled for this client",
		}, nil
	}
	
	switch format {
	case "json":
		data, err := t.client.GetMetrics().ExportToJSON()
		if err != nil {
			return &OptimizationResult{
				Action:  "export",
				Success: false,
				Message: fmt.Sprintf("Failed to export: %v", err),
			}, nil
		}
		
		return &OptimizationResult{
			Action:  "export",
			Success: true,
			Data:    string(data),
			Message: "Metrics exported to JSON format",
		}, nil
		
	default:
		return &OptimizationResult{
			Action:  "export",
			Success: false,
			Message: fmt.Sprintf("Unsupported export format: %s", format),
		}, nil
	}
}

func (t *OptimizationTool) GetSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Action to perform: 'stats', 'reset', 'cache', or 'export'",
				"enum":        []string{"stats", "reset", "cache", "export"},
			},
			"format": map[string]interface{}{
				"type":        "string",
				"description": "Output format: 'json' or 'summary' (for stats action)",
				"enum":        []string{"json", "summary"},
			},
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "Filter stats by specific operation (optional)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *OptimizationTool) GetDescription() string {
	return "Get optimization statistics, cache information, and performance metrics for Docker Compose operations. Supports filtering, resetting, and exporting metrics data."
}

// Helper function to parse parameters
func parseParams(params interface{}, target interface{}) error {
	jsonData, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}