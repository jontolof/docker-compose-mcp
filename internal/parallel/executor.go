package parallel

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Task struct {
	ID       string
	Name     string
	Func     func(ctx context.Context) (interface{}, error)
	Timeout  time.Duration
	Priority int
}

type TaskResult struct {
	ID       string
	Name     string
	Result   interface{}
	Error    error
	Duration time.Duration
	StartTime time.Time
	EndTime   time.Time
}

type Executor struct {
	maxWorkers   int
	taskQueue    chan *Task
	resultQueue  chan *TaskResult
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	results      map[string]*TaskResult
	resultsMutex sync.RWMutex
}

func NewExecutor(maxWorkers int) *Executor {
	ctx, cancel := context.WithCancel(context.Background())
	
	executor := &Executor{
		maxWorkers:  maxWorkers,
		taskQueue:   make(chan *Task, maxWorkers*2),
		resultQueue: make(chan *TaskResult, maxWorkers*2),
		ctx:         ctx,
		cancel:      cancel,
		results:     make(map[string]*TaskResult),
	}
	
	// Start workers
	for i := 0; i < maxWorkers; i++ {
		go executor.worker(i)
	}
	
	// Start result collector
	go executor.collectResults()
	
	return executor
}

func (e *Executor) worker(id int) {
	for task := range e.taskQueue {
		select {
		case <-e.ctx.Done():
			return
		default:
			e.executeTask(task)
		}
	}
}

func (e *Executor) executeTask(task *Task) {
	startTime := time.Now()
	
	// Create task context with timeout
	taskCtx := e.ctx
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(e.ctx, task.Timeout)
		defer cancel()
	}
	
	result := &TaskResult{
		ID:        task.ID,
		Name:      task.Name,
		StartTime: startTime,
	}
	
	// Execute task
	taskResult, err := task.Func(taskCtx)
	
	result.Result = taskResult
	result.Error = err
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	
	// Send result
	select {
	case e.resultQueue <- result:
	case <-e.ctx.Done():
		return
	}
}

func (e *Executor) collectResults() {
	for result := range e.resultQueue {
		e.resultsMutex.Lock()
		e.results[result.ID] = result
		e.resultsMutex.Unlock()
	}
}

func (e *Executor) SubmitTask(task *Task) error {
	select {
	case e.taskQueue <- task:
		return nil
	case <-e.ctx.Done():
		return fmt.Errorf("executor is shutting down")
	}
}

func (e *Executor) SubmitTasks(tasks []*Task) error {
	for _, task := range tasks {
		if err := e.SubmitTask(task); err != nil {
			return fmt.Errorf("failed to submit task %s: %w", task.ID, err)
		}
	}
	return nil
}

func (e *Executor) WaitForCompletion(taskIDs []string, timeout time.Duration) ([]*TaskResult, error) {
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeoutChan:
			return nil, fmt.Errorf("timeout waiting for tasks to complete")
		case <-ticker.C:
			if e.allTasksComplete(taskIDs) {
				return e.getResults(taskIDs), nil
			}
		case <-e.ctx.Done():
			return nil, fmt.Errorf("executor was cancelled")
		}
	}
}

func (e *Executor) allTasksComplete(taskIDs []string) bool {
	e.resultsMutex.RLock()
	defer e.resultsMutex.RUnlock()
	
	for _, id := range taskIDs {
		if _, exists := e.results[id]; !exists {
			return false
		}
	}
	return true
}

func (e *Executor) getResults(taskIDs []string) []*TaskResult {
	e.resultsMutex.RLock()
	defer e.resultsMutex.RUnlock()
	
	results := make([]*TaskResult, 0, len(taskIDs))
	for _, id := range taskIDs {
		if result, exists := e.results[id]; exists {
			results = append(results, result)
		}
	}
	return results
}

func (e *Executor) GetResult(taskID string) (*TaskResult, bool) {
	e.resultsMutex.RLock()
	defer e.resultsMutex.RUnlock()
	
	result, exists := e.results[taskID]
	return result, exists
}

func (e *Executor) Shutdown() {
	e.cancel()
	close(e.taskQueue)
	
	// Wait for all tasks to complete
	e.wg.Wait()
	
	close(e.resultQueue)
}

func (e *Executor) Stats() ExecutorStats {
	e.resultsMutex.RLock()
	defer e.resultsMutex.RUnlock()
	
	stats := ExecutorStats{
		MaxWorkers:    e.maxWorkers,
		CompletedTasks: len(e.results),
		QueuedTasks:   len(e.taskQueue),
	}
	
	var totalDuration time.Duration
	successCount := 0
	
	for _, result := range e.results {
		totalDuration += result.Duration
		if result.Error == nil {
			successCount++
		}
	}
	
	if len(e.results) > 0 {
		stats.AverageDuration = totalDuration / time.Duration(len(e.results))
		stats.SuccessRate = float64(successCount) / float64(len(e.results))
	}
	
	return stats
}

type ExecutorStats struct {
	MaxWorkers      int           `json:"maxWorkers"`
	CompletedTasks  int           `json:"completedTasks"`
	QueuedTasks     int           `json:"queuedTasks"`
	AverageDuration time.Duration `json:"averageDuration"`
	SuccessRate     float64       `json:"successRate"`
}

// ComposeTaskBuilder helps build common Docker Compose parallel tasks
type ComposeTaskBuilder struct {
	workDir string
	timeout time.Duration
}

func NewComposeTaskBuilder(workDir string, timeout time.Duration) *ComposeTaskBuilder {
	return &ComposeTaskBuilder{
		workDir: workDir,
		timeout: timeout,
	}
}

func (b *ComposeTaskBuilder) BuildParallelUp(services []string) []*Task {
	tasks := make([]*Task, 0, len(services))
	
	for i, service := range services {
		task := &Task{
			ID:       fmt.Sprintf("up_%s", service),
			Name:     fmt.Sprintf("Start %s", service),
			Priority: i, // Start in order
			Timeout:  b.timeout,
			Func: func(service string) func(ctx context.Context) (interface{}, error) {
				return func(ctx context.Context) (interface{}, error) {
					return b.executeDockerCommand(ctx, "up", "-d", service)
				}
			}(service),
		}
		tasks = append(tasks, task)
	}
	
	return tasks
}

func (b *ComposeTaskBuilder) BuildParallelBuild(services []string) []*Task {
	tasks := make([]*Task, 0, len(services))
	
	for _, service := range services {
		task := &Task{
			ID:      fmt.Sprintf("build_%s", service),
			Name:    fmt.Sprintf("Build %s", service),
			Timeout: b.timeout,
			Func: func(service string) func(ctx context.Context) (interface{}, error) {
				return func(ctx context.Context) (interface{}, error) {
					return b.executeDockerCommand(ctx, "build", service)
				}
			}(service),
		}
		tasks = append(tasks, task)
	}
	
	return tasks
}

func (b *ComposeTaskBuilder) BuildHealthChecks(services []string) []*Task {
	tasks := make([]*Task, 0, len(services))
	
	for _, service := range services {
		task := &Task{
			ID:      fmt.Sprintf("health_%s", service),
			Name:    fmt.Sprintf("Health check %s", service),
			Timeout: 30 * time.Second,
			Func: func(service string) func(ctx context.Context) (interface{}, error) {
				return func(ctx context.Context) (interface{}, error) {
					return b.executeDockerCommand(ctx, "ps", "--filter", fmt.Sprintf("name=%s", service), "--format", "table")
				}
			}(service),
		}
		tasks = append(tasks, task)
	}
	
	return tasks
}

func (b *ComposeTaskBuilder) executeDockerCommand(ctx context.Context, args ...string) (interface{}, error) {
	// This is a placeholder - in real implementation, this would execute docker-compose commands
	// For now, return a mock result
	return fmt.Sprintf("Executed: docker-compose %v", args), nil
}