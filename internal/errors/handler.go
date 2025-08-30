package errors

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

var (
	ErrInvalidParams    = errors.New("invalid parameters")
	ErrServiceRequired  = errors.New("service name is required")
	ErrSessionNotFound  = errors.New("session not found")
	ErrCommandTimeout   = errors.New("command execution timeout")
	ErrDockerNotRunning = errors.New("docker daemon is not running")
	ErrComposeNotFound  = errors.New("docker-compose.yml not found")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

type ErrorHandler struct {
	logger      *log.Logger
	logLevel    LogLevel
	maxRetries  int
	retryDelay  time.Duration
	enableStack bool
}

type LogLevel int

const (
	LevelError LogLevel = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

func NewErrorHandler(level LogLevel) *ErrorHandler {
	return &ErrorHandler{
		logger:      log.New(os.Stderr, "[docker-compose-mcp] ", log.LstdFlags|log.Lshortfile),
		logLevel:    level,
		maxRetries:  3,
		retryDelay:  time.Second,
		enableStack: level >= LevelDebug,
	}
}

func (h *ErrorHandler) Handle(err error) error {
	if err == nil {
		return nil
	}

	if h.logLevel >= LevelError {
		h.logger.Printf("ERROR: %v", err)
		if h.enableStack {
			h.logger.Printf("Stack trace:\n%s", debug.Stack())
		}
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("operation timed out: %w", err)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("operation was canceled: %w", err)
	case strings.Contains(err.Error(), "Cannot connect to the Docker daemon"):
		return fmt.Errorf("%w: %v", ErrDockerNotRunning, err)
	case strings.Contains(err.Error(), "no such file or directory"):
		if strings.Contains(err.Error(), "docker-compose.yml") {
			return fmt.Errorf("%w: %v", ErrComposeNotFound, err)
		}
		return fmt.Errorf("file not found: %w", err)
	case strings.Contains(err.Error(), "permission denied"):
		return fmt.Errorf("permission denied: %w", err)
	case strings.Contains(err.Error(), "network"):
		return fmt.Errorf("network error: %w", err)
	default:
		return err
	}
}

func (h *ErrorHandler) HandleWithRetry(fn func() error) error {
	var lastErr error
	
	for i := 0; i < h.maxRetries; i++ {
		if i > 0 {
			h.Debug("Retrying operation (attempt %d/%d)", i+1, h.maxRetries)
			time.Sleep(h.retryDelay)
		}
		
		if err := fn(); err != nil {
			lastErr = err
			if !h.isRetriableError(err) {
				return h.Handle(err)
			}
			h.Warn("Retriable error (attempt %d/%d): %v", i+1, h.maxRetries, err)
		} else {
			return nil
		}
	}
	
	return h.Handle(fmt.Errorf("operation failed after %d attempts: %w", h.maxRetries, lastErr))
}

func (h *ErrorHandler) isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	retriablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"resource temporarily unavailable",
		"network is unreachable",
		"broken pipe",
	}
	
	for _, pattern := range retriablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}
	
	return false
}

func (h *ErrorHandler) Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	wrapped := fmt.Errorf("%s: %w", msg, err)
	return h.Handle(wrapped)
}

func (h *ErrorHandler) WrapWithContext(err error, msg string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	contextMsg := fmt.Sprintf(msg, args...)
	return h.Wrap(err, contextMsg)
}

func (h *ErrorHandler) Debug(format string, args ...interface{}) {
	if h.logLevel >= LevelDebug {
		h.logger.Printf("DEBUG: "+format, args...)
	}
}

func (h *ErrorHandler) Info(format string, args ...interface{}) {
	if h.logLevel >= LevelInfo {
		h.logger.Printf("INFO: "+format, args...)
	}
}

func (h *ErrorHandler) Warn(format string, args ...interface{}) {
	if h.logLevel >= LevelWarn {
		h.logger.Printf("WARN: "+format, args...)
	}
}

func (h *ErrorHandler) Error(format string, args ...interface{}) {
	if h.logLevel >= LevelError {
		h.logger.Printf("ERROR: "+format, args...)
	}
}

func (h *ErrorHandler) SetLogLevel(level LogLevel) {
	h.logLevel = level
	h.enableStack = level >= LevelDebug
}

func (h *ErrorHandler) GetLogLevel() LogLevel {
	return h.logLevel
}

type RecoverableError struct {
	Err       error
	Recovered bool
	Message   string
}

func (r *RecoverableError) Error() string {
	if r.Recovered {
		return fmt.Sprintf("recovered from error: %v", r.Err)
	}
	return r.Err.Error()
}

func (h *ErrorHandler) RecoverPanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic recovered: %v", r)
		h.Handle(err)
		if h.enableStack {
			h.logger.Printf("Panic stack trace:\n%s", debug.Stack())
		}
	}
}

func (h *ErrorHandler) ValidateParams(params map[string]interface{}, required []string) error {
	var missing []string
	
	for _, field := range required {
		value, exists := params[field]
		if !exists {
			missing = append(missing, field)
			continue
		}
		
		if str, ok := value.(string); ok && str == "" {
			missing = append(missing, field)
		}
	}
	
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing required fields: %s", ErrInvalidParams, strings.Join(missing, ", "))
	}
	
	return nil
}

func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}