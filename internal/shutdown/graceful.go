package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/logging"
)

type Handler func(ctx context.Context) error

type Manager struct {
	logger   *logging.Logger
	handlers []Handler
	timeout  time.Duration
	mu       sync.RWMutex
	done     chan struct{}
}

func NewManager(timeout time.Duration) *Manager {
	return &Manager{
		logger:   logging.NewLogger("shutdown", logging.GetLogLevel(), logging.IsStructuredLogging()),
		handlers: make([]Handler, 0),
		timeout:  timeout,
		done:     make(chan struct{}),
	}
}

func (m *Manager) AddHandler(handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

func (m *Manager) AddHandlerFunc(name string, fn func() error) {
	m.AddHandler(func(ctx context.Context) error {
		m.logger.Infof("Executing shutdown handler: %s", name)
		return fn()
	})
}

func (m *Manager) Listen() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		m.logger.Infof("Received signal: %v", sig)
		m.Shutdown()
	}()
}

func (m *Manager) Shutdown() {
	select {
	case <-m.done:
		return // Already shutting down
	default:
		close(m.done)
	}

	m.logger.Info("Initiating graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	m.mu.RLock()
	handlers := make([]Handler, len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(handlers))

	for i, handler := range handlers {
		wg.Add(1)
		go func(index int, h Handler) {
			defer wg.Done()
			
			handlerCtx, handlerCancel := context.WithTimeout(ctx, m.timeout/2)
			defer handlerCancel()
			
			m.logger.Debugf("Executing shutdown handler %d", index+1)
			if err := h(handlerCtx); err != nil {
				m.logger.Errorf("Shutdown handler %d failed: %v", index+1, err)
				errChan <- fmt.Errorf("handler %d: %w", index+1, err)
			} else {
				m.logger.Debugf("Shutdown handler %d completed successfully", index+1)
			}
		}(i, handler)
	}

	// Wait for all handlers to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info("All shutdown handlers completed successfully")
	case <-ctx.Done():
		m.logger.Warn("Shutdown timeout reached, some handlers may not have completed")
	}

	// Collect any errors
	close(errChan)
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		m.logger.Errorf("Shutdown completed with %d errors", len(errors))
		for _, err := range errors {
			m.logger.Error(err.Error())
		}
	} else {
		m.logger.Info("Graceful shutdown completed successfully")
	}
}

func (m *Manager) Done() <-chan struct{} {
	return m.done
}

func (m *Manager) SetTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeout = timeout
}

func (m *Manager) IsShuttingDown() bool {
	select {
	case <-m.done:
		return true
	default:
		return false
	}
}

type Resource interface {
	Close() error
}

func (m *Manager) RegisterResource(name string, resource Resource) {
	m.AddHandlerFunc(fmt.Sprintf("close_%s", name), func() error {
		return resource.Close()
	})
}

type Cleaner interface {
	Cleanup() error
}

func (m *Manager) RegisterCleaner(name string, cleaner Cleaner) {
	m.AddHandlerFunc(fmt.Sprintf("cleanup_%s", name), func() error {
		return cleaner.Cleanup()
	})
}

// Predefined handlers for common cleanup tasks
func (m *Manager) RegisterSessionCleanup(sessionManager interface {
	StopAllSessions() error
}) {
	m.AddHandlerFunc("stop_all_sessions", func() error {
		return sessionManager.StopAllSessions()
	})
}

func (m *Manager) RegisterCacheFlush(cache interface {
	Flush() error
}) {
	m.AddHandlerFunc("flush_cache", func() error {
		return cache.Flush()
	})
}

func (m *Manager) RegisterTempCleanup(tempDirs []string) {
	m.AddHandlerFunc("cleanup_temp_files", func() error {
		var errors []error
		for _, dir := range tempDirs {
			if err := os.RemoveAll(dir); err != nil {
				errors = append(errors, fmt.Errorf("failed to remove %s: %w", dir, err))
			}
		}
		if len(errors) > 0 {
			return fmt.Errorf("temp cleanup errors: %v", errors)
		}
		return nil
	})
}