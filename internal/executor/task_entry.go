package executor

import (
	"delayed-task-exec/future"
	"delayed-task-exec/task"
	"time"
)

// TaskEntry is a wrapper around a Task that captures it's execution details and result. It's exposed to
// external callers via the ResultListener interface.
type TaskEntry struct {
	SubmittedTime time.Time
	ScheduledTime time.Time
	EnqueuedTime  time.Time
	ExecutedTime  time.Time
	CompletedTime time.Time
	Delay         time.Duration
	Task          task.Task
	Result        future.Future

	// used by executor impl to set Result
	Resultch chan future.FutureResult
}

// NewTaskEntry creates a new TaskEntry instance
func NewTaskEntry(submissionTime time.Time, delayDuration time.Duration, task task.Task) *TaskEntry {
	resultch := make(chan future.FutureResult, 1)
	future := future.NewFuture(resultch)
	entry := TaskEntry{
		SubmittedTime: submissionTime,
		ScheduledTime: submissionTime.Add(delayDuration),
		Delay:         delayDuration,
		Task:          task,
		Result:        future,
		Resultch:      resultch,
	}
	return &entry
}
