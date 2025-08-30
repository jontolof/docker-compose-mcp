package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/workspace"
)

type WorkspaceTool struct {
	manager *workspace.Manager
}

func NewWorkspaceTool(manager *workspace.Manager) *WorkspaceTool {
	return &WorkspaceTool{
		manager: manager,
	}
}

func (w *WorkspaceTool) GetName() string {
	return "workspace_manage"
}

func (w *WorkspaceTool) GetDescription() string {
	return "Manage Docker Compose project workspaces - create, switch, list, and organize multiple projects"
}

func (w *WorkspaceTool) GetSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"create", "list", "switch", "current", "remove", "update", "discover"},
				"description": "Action to perform on workspaces",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Workspace name (for create, switch, remove, update actions)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Project path (for create action or discover search path)",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Workspace description (for create/update actions)",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Workspace tags (for create/update actions)",
			},
			"compose_file": map[string]interface{}{
				"type":        "string",
				"description": "Docker Compose file name (for create/update actions)",
			},
			"variables": map[string]interface{}{
				"type":        "object",
				"description": "Environment variables for workspace (for create/update actions)",
			},
		},
		"required": []string{"action"},
	}
}

func (w *WorkspaceTool) Execute(ctx context.Context, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	action, ok := paramsMap["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action is required")
	}

	switch action {
	case "create":
		return w.createWorkspace(paramsMap)
	case "list":
		return w.listWorkspaces()
	case "switch":
		return w.switchWorkspace(paramsMap)
	case "current":
		return w.getCurrentWorkspace()
	case "remove":
		return w.removeWorkspace(paramsMap)
	case "update":
		return w.updateWorkspace(paramsMap)
	case "discover":
		return w.discoverWorkspaces(paramsMap)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (w *WorkspaceTool) createWorkspace(params map[string]interface{}) (interface{}, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required for create action")
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path is required for create action")
	}

	options := &workspace.WorkspaceOptions{}

	if desc, ok := params["description"].(string); ok {
		options.Description = desc
	}

	if composeFile, ok := params["compose_file"].(string); ok {
		options.ComposeFile = composeFile
	}

	if tags, ok := params["tags"].([]interface{}); ok {
		stringTags := make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				stringTags[i] = tagStr
			}
		}
		options.Tags = stringTags
	}

	if variables, ok := params["variables"].(map[string]interface{}); ok {
		stringVars := make(map[string]string)
		for k, v := range variables {
			if vStr, ok := v.(string); ok {
				stringVars[k] = vStr
			}
		}
		options.Variables = stringVars
	}

	workspace, err := w.manager.CreateWorkspace(name, path, options)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Created workspace '%s'", workspace.Name),
		"workspace": w.formatWorkspace(workspace),
	}, nil
}

func (w *WorkspaceTool) listWorkspaces() (interface{}, error) {
	workspaces := w.manager.ListWorkspaces()

	result := make([]map[string]interface{}, len(workspaces))
	for i, ws := range workspaces {
		result[i] = w.formatWorkspace(ws)
	}

	return map[string]interface{}{
		"success":    true,
		"count":      len(workspaces),
		"workspaces": result,
	}, nil
}

func (w *WorkspaceTool) switchWorkspace(params map[string]interface{}) (interface{}, error) {
	identifier, ok := params["name"].(string)
	if !ok || identifier == "" {
		return nil, fmt.Errorf("name is required for switch action")
	}

	workspace, err := w.manager.SwitchWorkspace(identifier)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Switched to workspace '%s'", workspace.Name),
		"workspace": w.formatWorkspace(workspace),
	}, nil
}

func (w *WorkspaceTool) getCurrentWorkspace() (interface{}, error) {
	workspace := w.manager.GetCurrentWorkspace()
	if workspace == nil {
		return map[string]interface{}{
			"success": true,
			"message": "No active workspace",
			"current": nil,
		}, nil
	}

	return map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Current workspace: %s", workspace.Name),
		"workspace": w.formatWorkspace(workspace),
	}, nil
}

func (w *WorkspaceTool) removeWorkspace(params map[string]interface{}) (interface{}, error) {
	identifier, ok := params["name"].(string)
	if !ok || identifier == "" {
		return nil, fmt.Errorf("name is required for remove action")
	}

	// Get workspace info before removal
	workspace, err := w.manager.GetWorkspace(identifier)
	if err != nil {
		return nil, err
	}

	if err := w.manager.RemoveWorkspace(identifier); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Removed workspace '%s'", workspace.Name),
	}, nil
}

func (w *WorkspaceTool) updateWorkspace(params map[string]interface{}) (interface{}, error) {
	identifier, ok := params["name"].(string)
	if !ok || identifier == "" {
		return nil, fmt.Errorf("name is required for update action")
	}

	updates := &workspace.WorkspaceUpdates{}

	if desc, ok := params["description"].(string); ok {
		updates.Description = &desc
	}

	if composeFile, ok := params["compose_file"].(string); ok {
		updates.ComposeFile = composeFile
	}

	if tags, ok := params["tags"].([]interface{}); ok {
		stringTags := make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				stringTags[i] = tagStr
			}
		}
		updates.Tags = stringTags
	}

	if variables, ok := params["variables"].(map[string]interface{}); ok {
		stringVars := make(map[string]string)
		for k, v := range variables {
			if vStr, ok := v.(string); ok {
				stringVars[k] = vStr
			}
		}
		updates.Variables = stringVars
	}

	workspace, err := w.manager.UpdateWorkspace(identifier, updates)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Updated workspace '%s'", workspace.Name),
		"workspace": w.formatWorkspace(workspace),
	}, nil
}

func (w *WorkspaceTool) discoverWorkspaces(params map[string]interface{}) (interface{}, error) {
	searchPath, ok := params["path"].(string)
	if !ok || searchPath == "" {
		searchPath = "." // Default to current directory
	}

	candidates, err := w.manager.DiscoverWorkspaces(searchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover workspaces: %w", err)
	}

	result := make([]map[string]interface{}, len(candidates))
	for i, candidate := range candidates {
		result[i] = map[string]interface{}{
			"name":         candidate.Name,
			"path":         candidate.Path,
			"compose_file": candidate.ComposeFile,
			"detected":     candidate.Detected.Format(time.RFC3339),
		}
	}

	return map[string]interface{}{
		"success":    true,
		"message":    fmt.Sprintf("Discovered %d potential workspaces", len(candidates)),
		"candidates": result,
		"search_path": searchPath,
	}, nil
}

func (w *WorkspaceTool) formatWorkspace(ws *workspace.Workspace) map[string]interface{} {
	result := map[string]interface{}{
		"id":           ws.ID,
		"name":         ws.Name,
		"path":         ws.Path,
		"compose_file": ws.ComposeFile,
		"active":       ws.Active,
		"created":      ws.Created.Format(time.RFC3339),
		"last_used":    ws.LastUsed.Format(time.RFC3339),
	}

	if ws.Description != "" {
		result["description"] = ws.Description
	}

	if len(ws.Tags) > 0 {
		result["tags"] = ws.Tags
	}

	if len(ws.Variables) > 0 {
		result["variables"] = ws.Variables
	}

	if len(ws.Settings) > 0 {
		result["settings"] = ws.Settings
	}

	// Add convenience information
	absPath, _ := filepath.Abs(ws.Path)
	result["absolute_path"] = absPath
	result["compose_path"] = filepath.Join(absPath, ws.ComposeFile)

	return result
}

// Project Discovery Tool - separate tool for discovering projects
type ProjectDiscoveryTool struct {
	manager *workspace.Manager
}

func NewProjectDiscoveryTool(manager *workspace.Manager) *ProjectDiscoveryTool {
	return &ProjectDiscoveryTool{
		manager: manager,
	}
}

func (p *ProjectDiscoveryTool) GetName() string {
	return "project_discover"
}

func (p *ProjectDiscoveryTool) GetDescription() string {
	return "Discover Docker Compose projects in directory tree and analyze their structure"
}

func (p *ProjectDiscoveryTool) GetSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to search for Docker Compose projects",
				"default":     ".",
			},
			"max_depth": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum directory depth to search",
				"default":     3,
			},
			"include_existing": map[string]interface{}{
				"type":        "boolean",
				"description": "Include projects that are already workspaces",
				"default":     false,
			},
		},
	}
}

func (p *ProjectDiscoveryTool) Execute(ctx context.Context, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	searchPath := "."
	if path, ok := paramsMap["path"].(string); ok && path != "" {
		searchPath = path
	}

	includeExisting := false
	if include, ok := paramsMap["include_existing"].(bool); ok {
		includeExisting = include
	}

	candidates, err := p.manager.DiscoverWorkspaces(searchPath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover projects: %w", err)
	}

	// Filter out existing workspaces if requested
	if !includeExisting {
		existingWorkspaces := p.manager.ListWorkspaces()
		existingPaths := make(map[string]bool)
		for _, ws := range existingWorkspaces {
			existingPaths[ws.Path] = true
		}

		var filtered []*workspace.WorkspaceCandidate
		for _, candidate := range candidates {
			if !existingPaths[candidate.Path] {
				filtered = append(filtered, candidate)
			}
		}
		candidates = filtered
	}

	// Analyze each candidate
	projects := make([]map[string]interface{}, len(candidates))
	for i, candidate := range candidates {
		analysis := p.analyzeProject(candidate)
		projects[i] = map[string]interface{}{
			"name":         candidate.Name,
			"path":         candidate.Path,
			"compose_file": candidate.ComposeFile,
			"detected":     candidate.Detected.Format(time.RFC3339),
			"analysis":     analysis,
		}
	}

	return map[string]interface{}{
		"success":          true,
		"message":          fmt.Sprintf("Discovered %d Docker Compose projects", len(projects)),
		"projects":         projects,
		"search_path":      searchPath,
		"include_existing": includeExisting,
	}, nil
}

func (p *ProjectDiscoveryTool) analyzeProject(candidate *workspace.WorkspaceCandidate) map[string]interface{} {
	analysis := map[string]interface{}{
		"has_dockerfile": false,
		"service_count":  0,
		"networks":       0,
		"volumes":        0,
		"languages":      []string{},
		"frameworks":     []string{},
	}

	composePath := filepath.Join(candidate.Path, candidate.ComposeFile)
	
	// Basic file existence checks
	if _, err := filepath.Glob(filepath.Join(candidate.Path, "*Dockerfile*")); err == nil {
		analysis["has_dockerfile"] = true
	}

	// Detect languages and frameworks based on files
	languages := p.detectLanguages(candidate.Path)
	frameworks := p.detectFrameworks(candidate.Path)
	
	analysis["languages"] = languages
	analysis["frameworks"] = frameworks

	// Try to read and parse compose file for service count
	// This is basic parsing - in a full implementation you'd use a YAML parser
	// For now, just estimate based on file size and structure
	if stat, err := os.Stat(composePath); err == nil {
		if stat.Size() > 1000 {
			analysis["complexity"] = "complex"
		} else if stat.Size() > 500 {
			analysis["complexity"] = "medium"
		} else {
			analysis["complexity"] = "simple"
		}
	}

	return analysis
}

func (p *ProjectDiscoveryTool) detectLanguages(projectPath string) []string {
	var languages []string
	
	languageFiles := map[string]string{
		"go.mod":         "Go",
		"package.json":   "JavaScript/Node.js",
		"requirements.txt": "Python",
		"Pipfile":        "Python",
		"Cargo.toml":     "Rust",
		"pom.xml":        "Java",
		"build.gradle":   "Java",
		"composer.json":  "PHP",
		"Gemfile":        "Ruby",
		".csproj":        "C#",
	}

	for file, language := range languageFiles {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			languages = append(languages, language)
		}
	}

	return languages
}

func (p *ProjectDiscoveryTool) detectFrameworks(projectPath string) []string {
	var frameworks []string

	// Check for common framework indicators
	if _, err := os.Stat(filepath.Join(projectPath, "next.config.js")); err == nil {
		frameworks = append(frameworks, "Next.js")
	}
	if _, err := os.Stat(filepath.Join(projectPath, "angular.json")); err == nil {
		frameworks = append(frameworks, "Angular")
	}
	if _, err := os.Stat(filepath.Join(projectPath, "vue.config.js")); err == nil {
		frameworks = append(frameworks, "Vue.js")
	}
	if _, err := os.Stat(filepath.Join(projectPath, "manage.py")); err == nil {
		frameworks = append(frameworks, "Django")
	}
	if _, err := os.Stat(filepath.Join(projectPath, "app.py")); err == nil {
		frameworks = append(frameworks, "Flask")
	}

	return frameworks
}