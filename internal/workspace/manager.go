package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/logging"
)

type Manager struct {
	logger      *logging.Logger
	configPath  string
	workspaces  map[string]*Workspace
	current     string
	mu          sync.RWMutex
}

type Workspace struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Path        string                 `json:"path"`
	ComposeFile string                 `json:"compose_file"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Variables   map[string]string      `json:"variables,omitempty"`
	LastUsed    time.Time              `json:"last_used"`
	Created     time.Time              `json:"created"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	Active      bool                   `json:"active"`
}

type WorkspaceConfig struct {
	Version    string                `json:"version"`
	Current    string                `json:"current"`
	Workspaces map[string]*Workspace `json:"workspaces"`
	Settings   map[string]interface{} `json:"settings,omitempty"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

func NewManager(configDir string) *Manager {
	logger := logging.NewLogger("workspace", logging.GetLogLevel(), logging.IsStructuredLogging())
	configPath := filepath.Join(configDir, "workspaces.json")

	manager := &Manager{
		logger:     logger,
		configPath: configPath,
		workspaces: make(map[string]*Workspace),
	}

	if err := manager.load(); err != nil {
		logger.Warnf("Failed to load workspace config: %v", err)
	}

	return manager
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		m.logger.Info("No existing workspace configuration found")
		return m.save()
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read workspace config: %w", err)
	}

	var config WorkspaceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse workspace config: %w", err)
	}

	m.workspaces = config.Workspaces
	m.current = config.Current

	// Ensure workspaces map is initialized
	if m.workspaces == nil {
		m.workspaces = make(map[string]*Workspace)
	}

	m.logger.Infof("Loaded %d workspaces", len(m.workspaces))
	return nil
}

func (m *Manager) save() error {
	// Should be called with lock held
	config := WorkspaceConfig{
		Version:    "1.0",
		Current:    m.current,
		Workspaces: m.workspaces,
		UpdatedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspace config: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write workspace config: %w", err)
	}

	return nil
}

func (m *Manager) CreateWorkspace(name, path string, options *WorkspaceOptions) (*Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if options == nil {
		options = &WorkspaceOptions{}
	}

	// Validate path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if workspace already exists
	for _, ws := range m.workspaces {
		if ws.Path == absPath {
			return nil, fmt.Errorf("workspace already exists at path: %s", absPath)
		}
		if ws.Name == name {
			return nil, fmt.Errorf("workspace with name '%s' already exists", name)
		}
	}

	// Detect compose file
	composeFile := options.ComposeFile
	if composeFile == "" {
		composeFile = m.detectComposeFile(absPath)
	}

	// Generate ID
	id := m.generateID(name)

	workspace := &Workspace{
		ID:          id,
		Name:        name,
		Path:        absPath,
		ComposeFile: composeFile,
		Description: options.Description,
		Tags:        options.Tags,
		Variables:   options.Variables,
		Settings:    options.Settings,
		Created:     time.Now(),
		LastUsed:    time.Now(),
		Active:      false,
	}

	m.workspaces[id] = workspace
	
	if err := m.save(); err != nil {
		delete(m.workspaces, id)
		return nil, fmt.Errorf("failed to save workspace: %w", err)
	}

	m.logger.Infof("Created workspace '%s' at %s", name, absPath)
	return workspace, nil
}

func (m *Manager) GetWorkspace(identifier string) (*Workspace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try by ID first
	if ws, exists := m.workspaces[identifier]; exists {
		return ws, nil
	}

	// Try by name
	for _, ws := range m.workspaces {
		if ws.Name == identifier {
			return ws, nil
		}
	}

	return nil, fmt.Errorf("workspace not found: %s", identifier)
}

func (m *Manager) ListWorkspaces() []*Workspace {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workspaces := make([]*Workspace, 0, len(m.workspaces))
	for _, ws := range m.workspaces {
		workspaces = append(workspaces, ws)
	}

	// Sort by last used (most recent first)
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].LastUsed.After(workspaces[j].LastUsed)
	})

	return workspaces
}

func (m *Manager) SwitchWorkspace(identifier string) (*Workspace, error) {
	workspace, err := m.GetWorkspace(identifier)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Deactivate current workspace
	if m.current != "" {
		if currentWs, exists := m.workspaces[m.current]; exists {
			currentWs.Active = false
		}
	}

	// Activate new workspace
	workspace.Active = true
	workspace.LastUsed = time.Now()
	m.current = workspace.ID

	if err := m.save(); err != nil {
		return nil, fmt.Errorf("failed to save workspace switch: %w", err)
	}

	// Change working directory
	if err := os.Chdir(workspace.Path); err != nil {
		m.logger.Warnf("Failed to change directory to %s: %v", workspace.Path, err)
	}

	m.logger.Infof("Switched to workspace '%s' at %s", workspace.Name, workspace.Path)
	return workspace, nil
}

func (m *Manager) GetCurrentWorkspace() *Workspace {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == "" {
		return nil
	}

	return m.workspaces[m.current]
}

func (m *Manager) RemoveWorkspace(identifier string) error {
	workspace, err := m.GetWorkspace(identifier)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Don't remove if it's the current workspace
	if workspace.ID == m.current {
		return fmt.Errorf("cannot remove active workspace '%s'", workspace.Name)
	}

	delete(m.workspaces, workspace.ID)

	if err := m.save(); err != nil {
		// Restore workspace if save fails
		m.workspaces[workspace.ID] = workspace
		return fmt.Errorf("failed to save after workspace removal: %w", err)
	}

	m.logger.Infof("Removed workspace '%s'", workspace.Name)
	return nil
}

func (m *Manager) UpdateWorkspace(identifier string, updates *WorkspaceUpdates) (*Workspace, error) {
	workspace, err := m.GetWorkspace(identifier)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Apply updates
	if updates.Name != "" && updates.Name != workspace.Name {
		// Check name uniqueness
		for _, ws := range m.workspaces {
			if ws.ID != workspace.ID && ws.Name == updates.Name {
				return nil, fmt.Errorf("workspace name '%s' already exists", updates.Name)
			}
		}
		workspace.Name = updates.Name
	}

	if updates.Description != nil {
		workspace.Description = *updates.Description
	}

	if updates.Tags != nil {
		workspace.Tags = updates.Tags
	}

	if updates.Variables != nil {
		if workspace.Variables == nil {
			workspace.Variables = make(map[string]string)
		}
		for k, v := range updates.Variables {
			if v == "" {
				delete(workspace.Variables, k)
			} else {
				workspace.Variables[k] = v
			}
		}
	}

	if updates.Settings != nil {
		if workspace.Settings == nil {
			workspace.Settings = make(map[string]interface{})
		}
		for k, v := range updates.Settings {
			workspace.Settings[k] = v
		}
	}

	if updates.ComposeFile != "" {
		workspace.ComposeFile = updates.ComposeFile
	}

	if err := m.save(); err != nil {
		return nil, fmt.Errorf("failed to save workspace updates: %w", err)
	}

	m.logger.Infof("Updated workspace '%s'", workspace.Name)
	return workspace, nil
}

func (m *Manager) DiscoverWorkspaces(searchPath string) ([]*WorkspaceCandidate, error) {
	var candidates []*WorkspaceCandidate

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}

		// Skip hidden directories and files
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Look for docker-compose files
		if !info.IsDir() && m.isComposeFile(info.Name()) {
			dir := filepath.Dir(path)
			candidate := &WorkspaceCandidate{
				Path:        dir,
				ComposeFile: info.Name(),
				Name:        m.generateNameFromPath(dir),
				Detected:    time.Now(),
			}
			candidates = append(candidates, candidate)
		}

		return nil
	})

	return candidates, err
}

// Helper methods

func (m *Manager) detectComposeFile(path string) string {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	for _, candidate := range candidates {
		fullPath := filepath.Join(path, candidate)
		if _, err := os.Stat(fullPath); err == nil {
			return candidate
		}
	}

	return "docker-compose.yml" // Default
}

func (m *Manager) isComposeFile(filename string) bool {
	composeFiles := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	for _, cf := range composeFiles {
		if filename == cf {
			return true
		}
	}

	return false
}

func (m *Manager) generateID(name string) string {
	base := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	id := base

	counter := 1
	for {
		if _, exists := m.workspaces[id]; !exists {
			break
		}
		id = fmt.Sprintf("%s-%d", base, counter)
		counter++
	}

	return id
}

func (m *Manager) generateNameFromPath(path string) string {
	return filepath.Base(path)
}

// Supporting types

type WorkspaceOptions struct {
	ComposeFile string
	Description string
	Tags        []string
	Variables   map[string]string
	Settings    map[string]interface{}
}

type WorkspaceUpdates struct {
	Name        string
	Description *string
	Tags        []string
	Variables   map[string]string
	Settings    map[string]interface{}
	ComposeFile string
}

type WorkspaceCandidate struct {
	Path        string    `json:"path"`
	ComposeFile string    `json:"compose_file"`
	Name        string    `json:"name"`
	Detected    time.Time `json:"detected"`
}