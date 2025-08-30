package docker

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/logging"
)

type HostManager struct {
	logger    *logging.Logger
	hosts     map[string]*DockerHost
	current   string
	defaultTL time.Duration
}

type DockerHost struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Host        string            `json:"host"`
	Type        HostType          `json:"type"`
	Context     string            `json:"context,omitempty"`
	TLS         *TLSConfig        `json:"tls,omitempty"`
	SSH         *SSHConfig        `json:"ssh,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Active      bool              `json:"active"`
	LastUsed    time.Time         `json:"last_used"`
	Created     time.Time         `json:"created"`
	HealthCheck *HealthStatus     `json:"health,omitempty"`
}

type HostType string

const (
	HostTypeLocal      HostType = "local"
	HostTypeRemote     HostType = "remote"
	HostTypeContext    HostType = "context"
	HostTypeSSH        HostType = "ssh"
	HostTypeContainer  HostType = "container"
)

type TLSConfig struct {
	Enabled    bool   `json:"enabled"`
	Verify     bool   `json:"verify"`
	CertPath   string `json:"cert_path,omitempty"`
	KeyPath    string `json:"key_path,omitempty"`
	CAPath     string `json:"ca_path,omitempty"`
	ServerName string `json:"server_name,omitempty"`
}

type SSHConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	KeyPath    string `json:"key_path,omitempty"`
	Password   string `json:"password,omitempty"`
	KnownHosts string `json:"known_hosts,omitempty"`
}

type HealthStatus struct {
	Status      string    `json:"status"`
	LastChecked time.Time `json:"last_checked"`
	Latency     time.Duration `json:"latency"`
	Version     string    `json:"version,omitempty"`
	Error       string    `json:"error,omitempty"`
}

func NewHostManager() *HostManager {
	logger := logging.NewLogger("docker-host", logging.GetLogLevel(), logging.IsStructuredLogging())
	
	return &HostManager{
		logger:    logger,
		hosts:     make(map[string]*DockerHost),
		defaultTL: 30 * time.Second,
	}
}

func (h *HostManager) AddHost(host *DockerHost) error {
	if host.ID == "" {
		host.ID = h.generateID(host.Name)
	}
	
	// Validate host configuration
	if err := h.validateHost(host); err != nil {
		return fmt.Errorf("invalid host configuration: %w", err)
	}
	
	// Check for duplicates
	if existing, exists := h.hosts[host.ID]; exists {
		return fmt.Errorf("host with ID '%s' already exists: %s", host.ID, existing.Name)
	}
	
	// Set timestamps
	host.Created = time.Now()
	host.LastUsed = time.Now()
	host.Active = false
	
	// Test connection
	if err := h.testConnection(host); err != nil {
		h.logger.Warnf("Failed to connect to host '%s': %v", host.Name, err)
		host.HealthCheck = &HealthStatus{
			Status:      "unhealthy",
			LastChecked: time.Now(),
			Error:       err.Error(),
		}
	} else {
		host.HealthCheck = &HealthStatus{
			Status:      "healthy",
			LastChecked: time.Now(),
		}
	}
	
	h.hosts[host.ID] = host
	h.logger.Infof("Added Docker host '%s' (%s)", host.Name, host.Host)
	
	return nil
}

func (h *HostManager) GetHost(identifier string) (*DockerHost, error) {
	// Try by ID first
	if host, exists := h.hosts[identifier]; exists {
		return host, nil
	}
	
	// Try by name
	for _, host := range h.hosts {
		if host.Name == identifier {
			return host, nil
		}
	}
	
	return nil, fmt.Errorf("Docker host not found: %s", identifier)
}

func (h *HostManager) ListHosts() []*DockerHost {
	hosts := make([]*DockerHost, 0, len(h.hosts))
	for _, host := range h.hosts {
		hosts = append(hosts, host)
	}
	return hosts
}

func (h *HostManager) SwitchHost(identifier string) (*DockerHost, error) {
	host, err := h.GetHost(identifier)
	if err != nil {
		return nil, err
	}
	
	// Deactivate current host
	if h.current != "" {
		if currentHost, exists := h.hosts[h.current]; exists {
			currentHost.Active = false
		}
	}
	
	// Test connection before switching
	if err := h.testConnection(host); err != nil {
		return nil, fmt.Errorf("cannot switch to unhealthy host: %w", err)
	}
	
	// Activate new host
	host.Active = true
	host.LastUsed = time.Now()
	h.current = host.ID
	
	// Set environment variables
	if err := h.setEnvironment(host); err != nil {
		h.logger.Warnf("Failed to set environment for host '%s': %v", host.Name, err)
	}
	
	h.logger.Infof("Switched to Docker host '%s' (%s)", host.Name, host.Host)
	return host, nil
}

func (h *HostManager) GetCurrentHost() *DockerHost {
	if h.current == "" {
		return h.getDefaultHost()
	}
	return h.hosts[h.current]
}

func (h *HostManager) RemoveHost(identifier string) error {
	host, err := h.GetHost(identifier)
	if err != nil {
		return err
	}
	
	// Don't remove if it's the current host
	if host.ID == h.current {
		return fmt.Errorf("cannot remove active host '%s'", host.Name)
	}
	
	delete(h.hosts, host.ID)
	h.logger.Infof("Removed Docker host '%s'", host.Name)
	return nil
}

func (h *HostManager) CheckHealth(identifier string) (*HealthStatus, error) {
	host, err := h.GetHost(identifier)
	if err != nil {
		return nil, err
	}
	
	start := time.Now()
	err = h.testConnection(host)
	latency := time.Since(start)
	
	status := &HealthStatus{
		LastChecked: time.Now(),
		Latency:     latency,
	}
	
	if err != nil {
		status.Status = "unhealthy"
		status.Error = err.Error()
	} else {
		status.Status = "healthy"
		// Try to get Docker version
		if version, vErr := h.getDockerVersion(host); vErr == nil {
			status.Version = version
		}
	}
	
	// Update host health
	host.HealthCheck = status
	
	return status, nil
}

func (h *HostManager) DiscoverHosts() ([]*DockerHost, error) {
	var discovered []*DockerHost
	
	// Check local Docker daemon
	if localHost := h.discoverLocalHost(); localHost != nil {
		discovered = append(discovered, localHost)
	}
	
	// Discover Docker contexts
	contexts, err := h.discoverContexts()
	if err != nil {
		h.logger.Warnf("Failed to discover contexts: %v", err)
	} else {
		discovered = append(discovered, contexts...)
	}
	
	// Check common environment variables
	if envHost := h.discoverFromEnvironment(); envHost != nil {
		discovered = append(discovered, envHost)
	}
	
	return discovered, nil
}

// Helper methods

func (h *HostManager) validateHost(host *DockerHost) error {
	if host.Name == "" {
		return fmt.Errorf("host name is required")
	}
	
	if host.Host == "" && host.Context == "" {
		return fmt.Errorf("either host or context is required")
	}
	
	// Validate host URL format
	if host.Host != "" {
		if _, err := url.Parse(host.Host); err != nil {
			return fmt.Errorf("invalid host URL: %w", err)
		}
	}
	
	// Validate TLS configuration
	if host.TLS != nil && host.TLS.Enabled {
		if host.TLS.CertPath != "" {
			if _, err := os.Stat(host.TLS.CertPath); os.IsNotExist(err) {
				return fmt.Errorf("TLS certificate file not found: %s", host.TLS.CertPath)
			}
		}
		if host.TLS.KeyPath != "" {
			if _, err := os.Stat(host.TLS.KeyPath); os.IsNotExist(err) {
				return fmt.Errorf("TLS key file not found: %s", host.TLS.KeyPath)
			}
		}
	}
	
	// Validate SSH configuration
	if host.SSH != nil {
		if host.SSH.Host == "" {
			return fmt.Errorf("SSH host is required")
		}
		if host.SSH.User == "" {
			return fmt.Errorf("SSH user is required")
		}
		if host.SSH.KeyPath != "" {
			if _, err := os.Stat(host.SSH.KeyPath); os.IsNotExist(err) {
				return fmt.Errorf("SSH key file not found: %s", host.SSH.KeyPath)
			}
		}
	}
	
	return nil
}

func (h *HostManager) testConnection(host *DockerHost) error {
	ctx, cancel := context.WithTimeout(context.Background(), h.defaultTL)
	defer cancel()
	
	switch host.Type {
	case HostTypeLocal:
		return h.testLocalConnection(ctx)
	case HostTypeRemote:
		return h.testRemoteConnection(ctx, host)
	case HostTypeContext:
		return h.testContextConnection(ctx, host)
	case HostTypeSSH:
		return h.testSSHConnection(ctx, host)
	default:
		return fmt.Errorf("unsupported host type: %s", host.Type)
	}
}

func (h *HostManager) testLocalConnection(ctx context.Context) error {
	// Try common local Docker socket paths
	sockets := []string{
		"/var/run/docker.sock",
		"/tmp/docker.sock",
		"~/.docker/docker.sock",
	}
	
	for _, socket := range sockets {
		if socket[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			socket = filepath.Join(home, socket[1:])
		}
		
		if _, err := os.Stat(socket); err == nil {
			// Try to connect
			conn, err := net.DialTimeout("unix", socket, h.defaultTL)
			if err != nil {
				continue
			}
			conn.Close()
			return nil
		}
	}
	
	return fmt.Errorf("no accessible local Docker daemon found")
}

func (h *HostManager) testRemoteConnection(ctx context.Context, host *DockerHost) error {
	parsed, err := url.Parse(host.Host)
	if err != nil {
		return fmt.Errorf("invalid host URL: %w", err)
	}
	
	var dialer net.Dialer
	dialer.Timeout = h.defaultTL
	
	switch parsed.Scheme {
	case "tcp":
		conn, err := dialer.DialContext(ctx, "tcp", parsed.Host)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
		
	case "https":
		config := &tls.Config{InsecureSkipVerify: true}
		if host.TLS != nil {
			config.InsecureSkipVerify = !host.TLS.Verify
			if host.TLS.ServerName != "" {
				config.ServerName = host.TLS.ServerName
			}
		}
		
		conn, err := tls.DialWithDialer(&dialer, "tcp", parsed.Host, config)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
		
	default:
		return fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
	}
}

func (h *HostManager) testContextConnection(ctx context.Context, host *DockerHost) error {
	// This would typically use docker context commands
	// For now, we'll simulate the check
	return fmt.Errorf("context connection testing not yet implemented")
}

func (h *HostManager) testSSHConnection(ctx context.Context, host *DockerHost) error {
	// This would typically use SSH client libraries
	// For now, we'll simulate the check
	if host.SSH == nil {
		return fmt.Errorf("SSH configuration missing")
	}
	
	address := fmt.Sprintf("%s:%d", host.SSH.Host, host.SSH.Port)
	if host.SSH.Port == 0 {
		address = fmt.Sprintf("%s:22", host.SSH.Host)
	}
	
	// Basic TCP connectivity test
	var dialer net.Dialer
	dialer.Timeout = h.defaultTL
	
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	conn.Close()
	
	return nil
}

func (h *HostManager) setEnvironment(host *DockerHost) error {
	// Clear existing Docker environment
	dockerEnvVars := []string{
		"DOCKER_HOST",
		"DOCKER_TLS_VERIFY",
		"DOCKER_CERT_PATH",
		"DOCKER_CONTEXT",
	}
	
	for _, envVar := range dockerEnvVars {
		os.Unsetenv(envVar)
	}
	
	// Set new environment based on host configuration
	switch host.Type {
	case HostTypeContext:
		if host.Context != "" {
			os.Setenv("DOCKER_CONTEXT", host.Context)
		}
		
	case HostTypeRemote, HostTypeSSH:
		if host.Host != "" {
			os.Setenv("DOCKER_HOST", host.Host)
		}
		
		if host.TLS != nil && host.TLS.Enabled {
			if host.TLS.Verify {
				os.Setenv("DOCKER_TLS_VERIFY", "1")
			}
			if host.TLS.CertPath != "" {
				os.Setenv("DOCKER_CERT_PATH", filepath.Dir(host.TLS.CertPath))
			}
		}
	}
	
	// Set custom environment variables
	for key, value := range host.Environment {
		os.Setenv(key, value)
	}
	
	return nil
}

func (h *HostManager) getDockerVersion(host *DockerHost) (string, error) {
	// This would typically execute `docker version` command
	// For now, return a placeholder
	return "20.10.0", nil
}

func (h *HostManager) generateID(name string) string {
	base := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	id := base
	
	counter := 1
	for {
		if _, exists := h.hosts[id]; !exists {
			break
		}
		id = fmt.Sprintf("%s-%d", base, counter)
		counter++
	}
	
	return id
}

func (h *HostManager) getDefaultHost() *DockerHost {
	// Return a default local host
	return &DockerHost{
		ID:     "local",
		Name:   "Local Docker",
		Host:   "unix:///var/run/docker.sock",
		Type:   HostTypeLocal,
		Active: true,
	}
}

func (h *HostManager) discoverLocalHost() *DockerHost {
	if err := h.testLocalConnection(context.Background()); err == nil {
		return &DockerHost{
			ID:          "discovered-local",
			Name:        "Local Docker Daemon",
			Host:        "unix:///var/run/docker.sock",
			Type:        HostTypeLocal,
			Description: "Automatically discovered local Docker daemon",
		}
	}
	return nil
}

func (h *HostManager) discoverContexts() ([]*DockerHost, error) {
	// This would typically execute `docker context ls` command
	// For now, return empty list
	return []*DockerHost{}, nil
}

func (h *HostManager) discoverFromEnvironment() *DockerHost {
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
		hostType := HostTypeRemote
		if strings.HasPrefix(dockerHost, "unix://") {
			hostType = HostTypeLocal
		}
		
		return &DockerHost{
			ID:          "discovered-env",
			Name:        "Environment Docker Host",
			Host:        dockerHost,
			Type:        hostType,
			Description: "Discovered from DOCKER_HOST environment variable",
		}
	}
	return nil
}