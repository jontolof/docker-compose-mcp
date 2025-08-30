package tests

import (
	"context"
	"testing"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/cache"
	"github.com/jontolof/docker-compose-mcp/internal/compose"
	"github.com/jontolof/docker-compose-mcp/internal/parallel"
	"github.com/jontolof/docker-compose-mcp/internal/tools"
)

func TestSimpleConfigCache(t *testing.T) {
	cache := cache.NewConfigCache(".", 10, time.Hour)
	
	// Test cache stats
	stats := cache.Stats()
	if stats.MaxEntries != 10 {
		t.Errorf("Expected max entries 10, got %d", stats.MaxEntries)
	}
	
	if stats.Entries != 0 {
		t.Errorf("Expected 0 initial entries, got %d", stats.Entries)
	}
}

func TestSimpleParallelExecutor(t *testing.T) {
	executor := parallel.NewExecutor(2)
	defer executor.Shutdown()
	
	// Test executor stats
	stats := executor.Stats()
	if stats.MaxWorkers != 2 {
		t.Errorf("Expected 2 max workers, got %d", stats.MaxWorkers)
	}
	
	if stats.CompletedTasks != 0 {
		t.Errorf("Expected 0 initial completed tasks, got %d", stats.CompletedTasks)
	}
}

func TestOptimizedClient(t *testing.T) {
	opts := &compose.ClientOptions{
		WorkDir:        ".",
		EnableCache:    true,
		EnableMetrics:  true,
		EnableParallel: true,
		CacheSize:      10,
		CacheMaxAge:    time.Hour,
		MaxWorkers:     2,
	}
	
	client := compose.NewClient(opts)
	defer client.Shutdown()
	
	// Test that optimization features are enabled
	if client.GetCache() == nil {
		t.Errorf("Cache should be enabled")
	}
	
	if client.GetMetrics() == nil {
		t.Errorf("Metrics should be enabled")
	}
	
	if client.GetExecutor() == nil {
		t.Errorf("Parallel executor should be enabled")
	}
}

func TestOptimizationToolBasic(t *testing.T) {
	opts := &compose.ClientOptions{
		EnableMetrics: true,
		EnableCache:   true,
	}
	client := compose.NewClient(opts)
	defer client.Shutdown()
	
	tool := tools.NewOptimizationTool(client)
	
	// Test getting schema
	schema := tool.GetSchema()
	if schema == nil {
		t.Errorf("Schema should not be nil")
	}
	
	// Test getting description
	desc := tool.GetDescription()
	if len(desc) == 0 {
		t.Errorf("Description should not be empty")
	}
	
	// Test stats action with metrics disabled client first
	params := map[string]interface{}{
		"action": "stats",
		"format": "json",
	}
	
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Failed to execute stats action: %v", err)
	}
	
	optResult, ok := result.(*tools.OptimizationResult)
	if !ok {
		t.Fatalf("Expected OptimizationResult, got %T", result)
	}
	
	if optResult.Action != "stats" {
		t.Errorf("Expected action 'stats', got '%s'", optResult.Action)
	}
}