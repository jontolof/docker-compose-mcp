package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/mcp"
	"github.com/jontolof/docker-compose-mcp/internal/plugin"
)

// WorkflowPlugin demonstrates a workflow automation plugin
type WorkflowPlugin struct {
	config plugin.Config
	workflows map[string]Workflow
}

// Workflow represents an automated workflow
type Workflow struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Steps       []WorkflowStep         `json:"steps"`
	Triggers    []WorkflowTrigger      `json:"triggers"`
	Settings    map[string]interface{} `json:"settings"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // "compose", "script", "notification"
	Command  string                 `json:"command"`
	Params   map[string]interface{} `json:"params"`
	OnError  string                 `json:"onError"` // "stop", "continue", "retry"
	Timeout  time.Duration          `json:"timeout"`
}

// WorkflowTrigger represents workflow triggers
type WorkflowTrigger struct {
	Type      string `json:"type"` // "schedule", "event", "manual"
	Schedule  string `json:"schedule,omitempty"` // Cron expression
	Event     string `json:"event,omitempty"`    // Event name
	Condition string `json:"condition,omitempty"`
}

// NewPlugin creates a new workflow plugin instance
func NewPlugin() plugin.Plugin {
	return &WorkflowPlugin{
		workflows: make(map[string]Workflow),
	}
}

// Info returns plugin metadata
func (wp *WorkflowPlugin) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "workflow-automation",
		Version:     "1.0.0",
		Description: "Automated workflow execution for Docker Compose operations",
		Author:      "Docker Compose MCP Team",
		License:     "MIT",
		Tags:        []string{"workflow", "automation", "ci-cd"},
		Dependencies: []plugin.Dependency{
			{
				Name:    "docker-compose",
				Version: "2.0.0",
				Type:    "binary",
			},
		},
		MinVersion: "1.0.0",
	}
}

// Initialize initializes the plugin
func (wp *WorkflowPlugin) Initialize(ctx context.Context, config plugin.Config) error {
	wp.config = config
	
	// Load default workflows
	wp.loadDefaultWorkflows()
	
	return nil
}

// Tools returns the MCP tools provided by this plugin
func (wp *WorkflowPlugin) Tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "workflow_execute",
			Description: "Execute a predefined workflow",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"workflow": {Type: "string"},
					"params":   {Type: "object"},
				},
				Required: []string{"workflow"},
			},
			Handler: wp.executeWorkflow,
		},
		{
			Name:        "workflow_list",
			Description: "List available workflows",
			InputSchema: mcp.Schema{
				Type: "object",
			},
			Handler: wp.listWorkflows,
		},
		{
			Name:        "workflow_create",
			Description: "Create a new workflow",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"workflow": {Type: "object"},
				},
				Required: []string{"workflow"},
			},
			Handler: wp.createWorkflow,
		},
		{
			Name:        "workflow_status",
			Description: "Get workflow execution status",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Schema{
					"execution_id": {Type: "string"},
				},
				Required: []string{"execution_id"},
			},
			Handler: wp.getWorkflowStatus,
		},
	}
}

// Hooks returns event hooks provided by this plugin
func (wp *WorkflowPlugin) Hooks() []plugin.Hook {
	return []plugin.Hook{
		{
			Event:   plugin.EventServiceStart,
			Handler: wp.onServiceStart,
		},
		{
			Event:   plugin.EventServiceStop,
			Handler: wp.onServiceStop,
		},
		{
			Event:   plugin.EventError,
			Handler: wp.onError,
		},
	}
}

// Cleanup is called when the plugin is unloaded
func (wp *WorkflowPlugin) Cleanup() error {
	// Clean up any running workflows
	return nil
}

// Health performs a health check on the plugin
func (wp *WorkflowPlugin) Health() plugin.HealthStatus {
	return plugin.HealthStatus{
		Status:    "healthy",
		Message:   "Workflow plugin is running",
		LastCheck: time.Now(),
	}
}

// executeWorkflow executes a predefined workflow
func (wp *WorkflowPlugin) executeWorkflow(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	workflowName, ok := paramsMap["workflow"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow name is required")
	}

	workflow, exists := wp.workflows[workflowName]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowName)
	}

	// Execute workflow steps
	executionID := fmt.Sprintf("exec_%d", time.Now().Unix())
	
	result := map[string]interface{}{
		"execution_id": executionID,
		"workflow":     workflowName,
		"status":       "running",
		"started_at":   time.Now(),
		"steps":        []map[string]interface{}{},
	}

	// Simulate workflow execution
	for i, step := range workflow.Steps {
		stepResult := map[string]interface{}{
			"step":       i + 1,
			"name":       step.Name,
			"status":     "completed",
			"duration":   "1.2s",
			"output":     fmt.Sprintf("Step %s executed successfully", step.Name),
		}
		
		result["steps"] = append(result["steps"].([]map[string]interface{}), stepResult)
	}

	result["status"] = "completed"
	result["completed_at"] = time.Now()

	return result, nil
}

// listWorkflows lists available workflows
func (wp *WorkflowPlugin) listWorkflows(params interface{}) (interface{}, error) {
	workflows := make([]map[string]interface{}, 0, len(wp.workflows))
	
	for name, workflow := range wp.workflows {
		workflows = append(workflows, map[string]interface{}{
			"name":        name,
			"description": workflow.Description,
			"steps":       len(workflow.Steps),
			"triggers":    len(workflow.Triggers),
		})
	}

	return map[string]interface{}{
		"workflows": workflows,
		"total":     len(workflows),
	}, nil
}

// createWorkflow creates a new workflow
func (wp *WorkflowPlugin) createWorkflow(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	workflowData, ok := paramsMap["workflow"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("workflow data is required")
	}

	// Convert to Workflow struct (simplified)
	name, _ := workflowData["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}

	description, _ := workflowData["description"].(string)
	
	workflow := Workflow{
		Name:        name,
		Description: description,
		Steps:       []WorkflowStep{},
		Triggers:    []WorkflowTrigger{},
		Settings:    make(map[string]interface{}),
	}

	wp.workflows[name] = workflow

	return map[string]interface{}{
		"message": fmt.Sprintf("Workflow %s created successfully", name),
		"workflow": workflow,
	}, nil
}

// getWorkflowStatus gets workflow execution status
func (wp *WorkflowPlugin) getWorkflowStatus(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	executionID, ok := paramsMap["execution_id"].(string)
	if !ok {
		return nil, fmt.Errorf("execution_id is required")
	}

	// In a real implementation, this would track actual execution status
	return map[string]interface{}{
		"execution_id": executionID,
		"status":       "completed",
		"progress":     100,
		"current_step": nil,
		"logs":         []string{
			"Workflow execution started",
			"Step 1: Build services - completed",
			"Step 2: Run tests - completed",
			"Step 3: Deploy - completed",
			"Workflow execution completed successfully",
		},
	}, nil
}

// Event handlers
func (wp *WorkflowPlugin) onServiceStart(ctx context.Context, event plugin.Event) error {
	// Handle service start events
	serviceName, _ := event.Data["service"].(string)
	fmt.Printf("Workflow plugin: Service %s started\n", serviceName)
	
	// Check if any workflows should be triggered
	return nil
}

func (wp *WorkflowPlugin) onServiceStop(ctx context.Context, event plugin.Event) error {
	// Handle service stop events
	serviceName, _ := event.Data["service"].(string)
	fmt.Printf("Workflow plugin: Service %s stopped\n", serviceName)
	return nil
}

func (wp *WorkflowPlugin) onError(ctx context.Context, event plugin.Event) error {
	// Handle error events
	errorMsg, _ := event.Data["error"].(string)
	fmt.Printf("Workflow plugin: Error occurred: %s\n", errorMsg)
	return nil
}

// loadDefaultWorkflows loads some example workflows
func (wp *WorkflowPlugin) loadDefaultWorkflows() {
	// CI/CD Pipeline
	wp.workflows["ci-cd"] = Workflow{
		Name:        "ci-cd",
		Description: "Complete CI/CD pipeline with build, test, and deploy",
		Steps: []WorkflowStep{
			{
				Name:    "Build Services",
				Type:    "compose",
				Command: "compose_build",
				Params:  map[string]interface{}{"no_cache": false},
				OnError: "stop",
				Timeout: 10 * time.Minute,
			},
			{
				Name:    "Run Tests",
				Type:    "compose",
				Command: "compose_test",
				Params:  map[string]interface{}{"service": "app"},
				OnError: "stop",
				Timeout: 5 * time.Minute,
			},
			{
				Name:    "Deploy",
				Type:    "compose",
				Command: "compose_up",
				Params:  map[string]interface{}{"detach": true},
				OnError: "stop",
				Timeout: 2 * time.Minute,
			},
		},
		Triggers: []WorkflowTrigger{
			{
				Type:  "event",
				Event: "git_push",
			},
		},
	}

	// Development Setup
	wp.workflows["dev-setup"] = Workflow{
		Name:        "dev-setup",
		Description: "Development environment setup",
		Steps: []WorkflowStep{
			{
				Name:    "Pull Images",
				Type:    "compose",
				Command: "compose_build",
				Params:  map[string]interface{}{"pull": true},
				OnError: "continue",
				Timeout: 5 * time.Minute,
			},
			{
				Name:    "Start Database",
				Type:    "compose",
				Command: "compose_up",
				Params:  map[string]interface{}{"services": []string{"db"}, "detach": true},
				OnError: "stop",
				Timeout: 1 * time.Minute,
			},
			{
				Name:    "Run Migrations",
				Type:    "compose",
				Command: "compose_migrate",
				Params:  map[string]interface{}{"direction": "up"},
				OnError: "stop",
				Timeout: 2 * time.Minute,
			},
			{
				Name:    "Start Application",
				Type:    "compose",
				Command: "compose_up",
				Params:  map[string]interface{}{"detach": true},
				OnError: "stop",
				Timeout: 2 * time.Minute,
			},
		},
	}
}

// main is required for plugin compilation but won't be used
func main() {
	plugin := NewPlugin()
	data, _ := json.MarshalIndent(plugin.Info(), "", "  ")
	fmt.Println(string(data))
}