package compose

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/cache"
	"github.com/jontolof/docker-compose-mcp/internal/metrics"
	"github.com/jontolof/docker-compose-mcp/internal/parallel"
)

type Client struct {
	configCache   *cache.ConfigCache
	metrics       *metrics.FilterMetrics
	executor      *parallel.Executor
	workDir       string
	enableMetrics bool
}

type ClientOptions struct {
	WorkDir         string
	EnableCache     bool
	EnableMetrics   bool
	EnableParallel  bool
	CacheSize       int
	CacheMaxAge     time.Duration
	MaxWorkers      int
	CommandTimeout  time.Duration
}

func NewClient(opts *ClientOptions) *Client {
	if opts == nil {
		opts = &ClientOptions{
			WorkDir:        ".",
			EnableCache:    true,
			EnableMetrics:  true,
			EnableParallel: true,
			CacheSize:      100,
			CacheMaxAge:    30 * time.Minute,
			MaxWorkers:     4,
			CommandTimeout: 5 * time.Minute,
		}
	}

	client := &Client{
		workDir:       opts.WorkDir,
		enableMetrics: opts.EnableMetrics,
	}

	if opts.EnableCache {
		client.configCache = cache.NewConfigCache(opts.WorkDir, opts.CacheSize, opts.CacheMaxAge)
	}

	if opts.EnableMetrics {
		client.metrics = metrics.NewFilterMetrics()
	}

	if opts.EnableParallel {
		client.executor = parallel.NewExecutor(opts.MaxWorkers)
	}

	return client
}

func (c *Client) Execute(ctx context.Context, args []string) (string, error) {
	startTime := time.Now()
	cmd := exec.CommandContext(ctx, "docker-compose", args...)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", &ComposeError{
				Message: strings.TrimSpace(stderr.String()),
				Output:  stdout.String(),
			}
		}
		return "", err
	}
	
	output := stdout.String()
	
	// Record metrics if enabled
	if c.enableMetrics && c.metrics != nil {
		filterResult := &metrics.FilteringResult{
			Operation:      strings.Join(args, " "),
			InputSize:      int64(len(output)),
			OutputSize:     int64(len(output)), // Will be updated after filtering
			FilterTime:     time.Since(startTime),
			FiltersApplied: []string{"none"}, // Will be updated after filtering
			LinesFiltered:  0,
			LinesPreserved: int64(strings.Count(output, "\n")),
			ReductionRatio: 0.0,
		}
		c.metrics.RecordFilteringResult(filterResult)
	}
	
	return output, nil
}

func (c *Client) ExecuteWithFiltering(ctx context.Context, args []string, filterFunc func(string) string) (string, error) {
	startTime := time.Now()
	
	// Execute the command
	output, err := c.Execute(ctx, args)
	if err != nil {
		return output, err
	}
	
	// Apply filtering if provided
	originalSize := int64(len(output))
	originalLines := int64(strings.Count(output, "\n"))
	
	if filterFunc != nil {
		output = filterFunc(output)
	}
	
	// Record metrics if enabled
	if c.enableMetrics && c.metrics != nil {
		filteredSize := int64(len(output))
		filteredLines := int64(strings.Count(output, "\n"))
		
		var reductionRatio float64
		if originalSize > 0 {
			reductionRatio = 1.0 - (float64(filteredSize) / float64(originalSize))
		}
		
		filterResult := &metrics.FilteringResult{
			Operation:      strings.Join(args, " "),
			InputSize:      originalSize,
			OutputSize:     filteredSize,
			FilterTime:     time.Since(startTime),
			FiltersApplied: []string{"custom"},
			LinesFiltered:  originalLines - filteredLines,
			LinesPreserved: filteredLines,
			ReductionRatio: reductionRatio,
		}
		c.metrics.RecordFilteringResult(filterResult)
	}
	
	return output, nil
}

// GetServices returns cached service names or parses them from compose file
func (c *Client) GetServices() []string {
	if c.configCache != nil {
		configPath := c.configCache.GetConfigPath(c.workDir)
		return c.configCache.GetServices(configPath)
	}
	
	// Fallback: execute docker-compose config to get services
	ctx := context.Background()
	output, err := c.Execute(ctx, []string{"config", "--services"})
	if err != nil {
		return nil
	}
	
	services := strings.Fields(strings.TrimSpace(output))
	return services
}

// GetMetrics returns current filtering metrics
func (c *Client) GetMetrics() *metrics.FilterMetrics {
	return c.metrics
}

// GetCache returns the config cache
func (c *Client) GetCache() *cache.ConfigCache {
	return c.configCache
}

// GetExecutor returns the parallel executor
func (c *Client) GetExecutor() *parallel.Executor {
	return c.executor
}

// Shutdown cleanly shuts down the client and its resources
func (c *Client) Shutdown() {
	if c.executor != nil {
		c.executor.Shutdown()
	}
}

// Close implements the Resource interface for graceful shutdown
func (c *Client) Close() error {
	c.Shutdown()
	return nil
}

type ComposeError struct {
	Message string
	Output  string
}

func (e *ComposeError) Error() string {
	return e.Message
}