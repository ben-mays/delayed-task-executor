# Design
Submissions are pushed onto a min priority queue using `now() + delay` for ordering. A configurable pool of `scheduler` coroutines pulls from the priority queue and pushes events onto a worker channel, which is listened to by a pool of configurable number of `worker` coroutines. Task execution results are returned via a `Future` interface. Alternatively, results are also published via a `ResultListener` interface, which includes some additional execution metrics.

The scheduler is a for-loop that blocks using `sync.Cond` to wake up. On new task submission, or every 15ms, it wakes up and re-checks the top of the priority queue. I considered exploring some type of uniform sharding to improve throughput but stopped myself. I ran some benchmarks and found that the scheduler lag was dominated by queue thrashing when the queue is sparse -- increasing scheduling density reduced scheduler lag to ~zero.

Files:
- `future/future.go` - the future implementation, exposing `Future`, `FutureResult` and two functions create futures `NewFuture(resultch chan FutureResult)` and `NewFutureFn(fn func() (any, error))`
- `task/task.go` - just contains the `Task` interface
- `internal/executor/task_entry.go` - internal value type for task queue entry
- `internal/executor/task_queue.go` - an internal queue impl that wraps `container/heap` with a thread-safe push/pop interface.
- `delayed_executor.go` 
  - The public entrypoint for the library, containg the implementation of the scheduler and worker coroutines. Exposes `TaskExecutor` via the `NewTaskExecutor` constructor. `CompletedTask`, `ResultListener` and `MetricsListener` are all exposed to serve as a callback for results

# Assumptions

- This project makes the assumption that `submit` is going to work for all callers, and does not need additional features like backpressure or sharding to distribute load. 
- We assume that all scheduled tasks can fit in memory.
- There are no per-task SLAs or task-level concurrency controls.
- The task executor is not durable.
- I did not implement a shutdown hook to clean up state.

# Running / Testing

There is no driver or binary, rather you can run via `go test`. `TestExecutor` creates 10 random batches of jittered tasks and then waits on the futures. It produces some basic stats to judge performance (these are slightly off due to the process not draining the listener channel before termination).

```
❯ go test -count=1 -run ^TestExecutor$ -v
=== RUN   TestExecutor
submitting 498081 tasks
submitting 604162 tasks
submitting 614352 tasks
submitting 794478 tasks
submitting 847746 tasks
submitting 264228 tasks
submitting 689070 tasks
submitting 949352 tasks
submitting 279234 tasks
submitting 524743 tasks
waiting on 6065446 futures
processed 0 futures
processed 1000000 futures
processed 2000000 futures
processed 3000000 futures
processed 4000000 futures
processed 5000000 futures
processed 6000000 futures
✅ map[avg_executor_lag_ms:1 avg_scheduler_lag_ms:0 completed:6065446] 
--- PASS: TestExecutor (20.64s)
PASS
ok      delayed-task-exec       20.952s
```