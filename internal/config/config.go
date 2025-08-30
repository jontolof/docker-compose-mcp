package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/logging"
)

type Config struct {
	// Server configuration
	WorkDir         string        `json:"work_dir"`
	CommandTimeout  time.Duration `json:"command_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// Cache configuration
	EnableCache     bool          `json:"enable_cache"`
	CacheSize       int           `json:"cache_size"`
	CacheMaxAge     time.Duration `json:"cache_max_age"`
	CacheTTL        time.Duration `json:"cache_ttl"`

	// Performance configuration
	EnableMetrics   bool `json:"enable_metrics"`
	EnableParallel  bool `json:"enable_parallel"`
	MaxWorkers      int  `json:"max_workers"`

	// Logging configuration
	LogLevel        logging.Level `json:"log_level"`
	LogFormat       string        `json:"log_format"`
	LogFile         string        `json:"log_file"`

	// Docker configuration
	DockerHost      string `json:"docker_host"`
	ComposeFile     string `json:"compose_file"`
	ProjectName     string `json:"project_name"`

	// Session configuration
	MaxSessions     int           `json:"max_sessions"`
	SessionTimeout  time.Duration `json:"session_timeout"`

	// Security configuration
	AllowedCommands []string `json:"allowed_commands"`
	RestrictedPaths []string `json:"restricted_paths"`

	// Development configuration
	EnableDebug     bool `json:"enable_debug"`
	EnableProfile   bool `json:"enable_profile"`
}

var DefaultConfig = &Config{
	WorkDir:         ".",
	CommandTimeout:  5 * time.Minute,
	ShutdownTimeout: 30 * time.Second,
	
	EnableCache:     true,
	CacheSize:       100,
	CacheMaxAge:     30 * time.Minute,
	CacheTTL:        5 * time.Minute,
	
	EnableMetrics:   true,
	EnableParallel:  true,
	MaxWorkers:      4,
	
	LogLevel:        logging.InfoLevel,
	LogFormat:       "text",
	LogFile:         "",
	
	DockerHost:      "",
	ComposeFile:     "docker-compose.yml",
	ProjectName:     "",
	
	MaxSessions:     10,
	SessionTimeout:  1 * time.Hour,
	
	AllowedCommands: []string{
		"up", "down", "ps", "logs", "build", "exec", "test",
		"watch", "migrate", "db_reset", "db_backup",
	},
	RestrictedPaths: []string{
		"/etc", "/usr", "/var", "/root", "/home",
	},
	
	EnableDebug:     false,
	EnableProfile:   false,
}

func Load() (*Config, error) {
	config := *DefaultConfig // Copy defaults
	
	// Load from environment variables
	if err := loadFromEnv(&config); err != nil {
		return nil, fmt.Errorf("failed to load config from environment: %w", err)
	}
	
	// Load from config file if specified
	if configFile := os.Getenv("MCP_CONFIG_FILE"); configFile != "" {
		if err := loadFromFile(&config, configFile); err != nil {
			return nil, fmt.Errorf("failed to load config from file %s: %w", configFile, err)
		}
	}
	
	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &config, nil
}

func loadFromEnv(config *Config) error {
	envMappings := map[string]func(string) error{
		"MCP_WORK_DIR": func(val string) error {
			config.WorkDir = val
			return nil
		},
		"MCP_COMMAND_TIMEOUT": func(val string) error {
			timeout, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("invalid command timeout: %w", err)
			}
			config.CommandTimeout = timeout
			return nil
		},
		"MCP_SHUTDOWN_TIMEOUT": func(val string) error {
			timeout, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("invalid shutdown timeout: %w", err)
			}
			config.ShutdownTimeout = timeout
			return nil
		},
		"MCP_ENABLE_CACHE": func(val string) error {
			enabled, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid cache enable flag: %w", err)
			}
			config.EnableCache = enabled
			return nil
		},
		"MCP_CACHE_SIZE": func(val string) error {
			size, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid cache size: %w", err)
			}
			config.CacheSize = size
			return nil
		},
		"MCP_CACHE_MAX_AGE": func(val string) error {
			age, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("invalid cache max age: %w", err)
			}
			config.CacheMaxAge = age
			return nil
		},
		"MCP_ENABLE_METRICS": func(val string) error {
			enabled, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid metrics enable flag: %w", err)
			}
			config.EnableMetrics = enabled
			return nil
		},
		"MCP_ENABLE_PARALLEL": func(val string) error {
			enabled, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid parallel enable flag: %w", err)
			}
			config.EnableParallel = enabled
			return nil
		},
		"MCP_MAX_WORKERS": func(val string) error {
			workers, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid max workers: %w", err)
			}
			config.MaxWorkers = workers
			return nil
		},
		"MCP_LOG_LEVEL": func(val string) error {
			config.LogLevel = logging.ParseLevel(val)
			return nil
		},
		"MCP_LOG_FORMAT": func(val string) error {
			config.LogFormat = val
			return nil
		},
		"MCP_LOG_FILE": func(val string) error {
			config.LogFile = val
			return nil
		},
		"DOCKER_HOST": func(val string) error {
			config.DockerHost = val
			return nil
		},
		"COMPOSE_FILE": func(val string) error {
			config.ComposeFile = val
			return nil
		},
		"COMPOSE_PROJECT_NAME": func(val string) error {
			config.ProjectName = val
			return nil
		},
		"MCP_MAX_SESSIONS": func(val string) error {
			sessions, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid max sessions: %w", err)
			}
			config.MaxSessions = sessions
			return nil
		},
		"MCP_SESSION_TIMEOUT": func(val string) error {
			timeout, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("invalid session timeout: %w", err)
			}
			config.SessionTimeout = timeout
			return nil
		},
		"MCP_ALLOWED_COMMANDS": func(val string) error {
			config.AllowedCommands = strings.Split(val, ",")
			return nil
		},
		"MCP_RESTRICTED_PATHS": func(val string) error {
			config.RestrictedPaths = strings.Split(val, ",")
			return nil
		},
		"MCP_ENABLE_DEBUG": func(val string) error {
			enabled, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid debug enable flag: %w", err)
			}
			config.EnableDebug = enabled
			return nil
		},
		"MCP_ENABLE_PROFILE": func(val string) error {
			enabled, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("invalid profile enable flag: %w", err)
			}
			config.EnableProfile = enabled
			return nil
		},
	}
	
	for envVar, setter := range envMappings {
		if val := os.Getenv(envVar); val != "" {
			if err := setter(val); err != nil {
				return fmt.Errorf("error processing %s=%s: %w", envVar, val, err)
			}
		}
	}
	
	return nil
}

func loadFromFile(config *Config, filename string) error {
	// For now, we'll just check if file exists
	// In a full implementation, you might want to support JSON/YAML/TOML
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", filename)
	}
	
	// TODO: Implement actual file parsing (JSON/YAML/TOML)
	// This is a placeholder for now
	return nil
}

func validate(config *Config) error {
	var errors []string
	
	// Validate work directory
	if config.WorkDir == "" {
		errors = append(errors, "work_dir cannot be empty")
	} else if !filepath.IsAbs(config.WorkDir) {
		// Convert relative path to absolute
		absPath, err := filepath.Abs(config.WorkDir)
		if err != nil {
			errors = append(errors, fmt.Sprintf("invalid work_dir: %v", err))
		} else {
			config.WorkDir = absPath
		}
	}
	
	// Validate timeouts
	if config.CommandTimeout <= 0 {
		errors = append(errors, "command_timeout must be positive")
	}
	if config.ShutdownTimeout <= 0 {
		errors = append(errors, "shutdown_timeout must be positive")
	}
	
	// Validate cache settings
	if config.EnableCache {
		if config.CacheSize <= 0 {
			errors = append(errors, "cache_size must be positive when cache is enabled")
		}
		if config.CacheMaxAge <= 0 {
			errors = append(errors, "cache_max_age must be positive when cache is enabled")
		}
	}
	
	// Validate parallel settings
	if config.EnableParallel {
		if config.MaxWorkers <= 0 {
			errors = append(errors, "max_workers must be positive when parallel is enabled")
		}
		if config.MaxWorkers > 100 {
			errors = append(errors, "max_workers should not exceed 100")
		}
	}
	
	// Validate log settings
	if config.LogFormat != "" && config.LogFormat != "text" && config.LogFormat != "json" {
		errors = append(errors, "log_format must be 'text' or 'json'")
	}
	
	// Validate compose file
	if config.ComposeFile == "" {
		errors = append(errors, "compose_file cannot be empty")
	}
	
	// Validate session settings
	if config.MaxSessions <= 0 {
		errors = append(errors, "max_sessions must be positive")
	}
	if config.SessionTimeout <= 0 {
		errors = append(errors, "session_timeout must be positive")
	}
	
	// Validate allowed commands
	if len(config.AllowedCommands) == 0 {
		errors = append(errors, "allowed_commands cannot be empty")
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

func (c *Config) IsCommandAllowed(command string) bool {
	for _, allowed := range c.AllowedCommands {
		if command == allowed {
			return true
		}
	}
	return false
}

func (c *Config) IsPathRestricted(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return true // Err on the side of caution
	}
	
	for _, restricted := range c.RestrictedPaths {
		if strings.HasPrefix(absPath, restricted) {
			return true
		}
	}
	return false
}

func (c *Config) GetComposeFilePath() string {
	if filepath.IsAbs(c.ComposeFile) {
		return c.ComposeFile
	}
	return filepath.Join(c.WorkDir, c.ComposeFile)
}