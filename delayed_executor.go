package temporal

import (
	"delayed-task-exec/future"
	"delayed-task-exec/internal/executor"
	"delayed-task-exec/task"
	"sync"
	"time"
)

// CompletedTask represents a completed task. The Resultch field is always closed.
type CompletedTask struct {
	// We re-export internal type TaskEntry as CompletedTask to avoid exposing task queue internals.
	executor.TaskEntry
}

// ResultListener implementations can be registered with the TaskExecutor to provide a callback mechanism on completion.
type ResultListener interface {
	TaskCompleted(CompletedTask)
}

// MetricsListener implements the ResultListener interface to calculate some basic stats about
// the TaskExecutor performance.
type MetricsListener struct {
	completedCount  int
	avgSchedulerLag int
	avgExecutorLag  int
}

func (m *MetricsListener) TaskCompleted(entry CompletedTask) {
	m.completedCount += 1
	// time spent waiting to get picked up by the scheduler
	schedulerLag := int(entry.EnqueuedTime.UnixMilli() - entry.ScheduledTime.UnixMilli())
	// time spent waiting fo get picked up by a worker
	executorLag := int(entry.ExecutedTime.UnixMilli() - entry.EnqueuedTime.UnixMilli())
	// update averages
	m.avgSchedulerLag += (schedulerLag - m.avgSchedulerLag) / m.completedCount
	m.avgExecutorLag += (executorLag - m.avgExecutorLag) / m.completedCount
}

func (m *MetricsListener) Stats() map[string]int {
	return map[string]int{
		"completed":            m.completedCount,
		"avg_scheduler_lag_ms": m.avgSchedulerLag,
		"avg_executor_lag_ms":  m.avgExecutorLag,
	}
}

type TaskExecutor struct {
	queue            *executor.TaskQueue
	workch           chan executor.TaskEntry
	workchBufferSize int
	schedulerPeriod  time.Duration
	cond             *sync.Cond
	resultch         chan executor.TaskEntry
	resultListener   ResultListener
}

// WithResultListener registers the given ResultListener with the TaskExecutor.
func WithResultListener(resultListener ResultListener) func(*TaskExecutor) {
	return func(exec *TaskExecutor) {
		exec.resultListener = resultListener
	}
}

// WithBuffer sets the internal buffer size for the work queue. By default this is 0, blocking on worker availability. Setting a higher buffer
// may improve performance (or rather smooth it out) if task execution is intermittently bottlenecking.
func WithBuffer(size int) func(*TaskExecutor) {
	return func(exec *TaskExecutor) {
		exec.workchBufferSize = size
	}
}

// WithSchedulerPeriod sets the wake timer duration to poll for new tasks. In ideal conditions, the task queue is uniformly dense but when sparse,
// this will act as a lower bound for scheduler latency. Default value is 15ms.
func WithSchedulerPeriod(period time.Duration) func(*TaskExecutor) {
	return func(exec *TaskExecutor) {
		exec.schedulerPeriod = period
	}
}

// NewTaskExecutor returns a new TaskExecutor with the number of workers and schedulers running.
func NewTaskExecutor(workers int, schedulers int, options ...func(*TaskExecutor)) TaskExecutor {
	exec := TaskExecutor{
		queue:           executor.NewTaskQueue(),
		cond:            sync.NewCond(&sync.Mutex{}),
		resultListener:  &MetricsListener{},
		schedulerPeriod: time.Millisecond * 15,
	}

	// init options
	for _, opt := range options {
		opt(&exec)
	}

	// construct work/result channels
	exec.workch = make(chan executor.TaskEntry, exec.workchBufferSize) // Optional, we can buffer tasks to avoid schedulers from blocking on long-running tasks
	exec.resultch = make(chan executor.TaskEntry, 10000)               // We buffer results to avoid workers waiting on reporter

	// start result listener
	go exec.reporter()

	// start N schedulers
	for i := 0; i < schedulers; i++ {
		go exec.scheduler()
	}

	// set lower bound for waking up scheduler using signal
	go func(signaler *sync.Cond) {
		ticker := time.NewTicker(exec.schedulerPeriod)
		for {
			select {
			case <-ticker.C:
				// Note, this only wakes up a single scheduler to avoid contention. We could call Broadcast to wake all but in testing it did not improve performance.
				exec.cond.Signal()
			}
		}
	}(exec.cond)

	// start N workers
	for i := 0; i < workers; i++ {
		go exec.worker()
	}
	return exec
}

// Submit schedules a Task to be executed at time T+delay. Submit returns immediately with a Future that will contain the result of Task#Run.
func (exec *TaskExecutor) Submit(delayDuration time.Duration, task task.Task) future.Future {
	entry := executor.NewTaskEntry(time.Now(), delayDuration, task)
	exec.queue.Push(entry)
	exec.cond.Signal() // Wake up scheduler if it's not busy
	return entry.Result
}

func (exec *TaskExecutor) scheduler() {
	for {
		exec.cond.L.Lock()
		next := exec.queue.Peek()
		now := time.Now()

		if next == nil || next.ScheduledTime.After(now) {
			// No tasks or we're early: Wait until we get signaled from submit
			exec.cond.Wait()
		}

		task := exec.queue.Pop()
		if task == nil {
			exec.cond.L.Unlock()
			continue
		}
		if task.ScheduledTime.Before(now) {
			task.EnqueuedTime = now
			exec.workch <- *task
		} else {
			exec.queue.Push(task)
		}
		exec.cond.L.Unlock()
	}
}

func (exec *TaskExecutor) reporter() {
	for {
		select {
		case task := <-exec.resultch:
			exec.resultListener.TaskCompleted(CompletedTask{task})
		}
	}
}

func (exec *TaskExecutor) worker() {
	for {
		select {
		case task := <-exec.workch:
			task.ExecutedTime = time.Now()
			res, err := task.Task.Run()
			task.Resultch <- future.FutureResult{Result: res, Error: err}
			close(task.Resultch)
			task.CompletedTime = time.Now()
			exec.resultch <- task
		}
	}
}
