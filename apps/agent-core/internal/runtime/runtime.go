package runtime

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Task represents a scheduled or repeating background task.
type Task struct {
	Name     string
	Interval time.Duration
	RunOnce  bool
	Fn       func(ctx context.Context) error
}

// Runtime manages the agent's main event loop and background task scheduler.
type Runtime struct {
	tasks  []Task
	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New() *Runtime {
	return &Runtime{}
}

// Register adds a task to the runtime. Must be called before Start.
func (r *Runtime) Register(task Task) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks = append(r.tasks, task)
}

// Start launches all registered tasks in their own goroutines.
func (r *Runtime) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(ctx)

	r.mu.Lock()
	tasks := make([]Task, len(r.tasks))
	copy(tasks, r.tasks)
	r.mu.Unlock()

	for _, t := range tasks {
		r.wg.Add(1)
		go r.runTask(ctx, t)
	}

	slog.Info("[runtime] Started", "tasks", len(tasks))
}

func (r *Runtime) runTask(ctx context.Context, t Task) {
	defer r.wg.Done()

	slog.Info("[runtime] Task started", "task", t.Name)

	if err := t.Fn(ctx); err != nil {
		slog.Warn("[runtime] Task initial run failed", "task", t.Name, "error", err)
	}

	if t.RunOnce {
		slog.Info("[runtime] One-shot task completed", "task", t.Name)
		return
	}

	ticker := time.NewTicker(t.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("[runtime] Task stopping", "task", t.Name)
			return
		case <-ticker.C:
			if err := t.Fn(ctx); err != nil {
				slog.Warn("[runtime] Task tick failed", "task", t.Name, "error", err)
			}
		}
	}
}

// Stop cancels all running tasks and waits for them to finish.
func (r *Runtime) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	slog.Info("[runtime] All tasks stopped")
}

// TaskCount returns the number of registered tasks.
func (r *Runtime) TaskCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.tasks)
}
