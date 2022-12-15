package executor

import (
	"container/heap"
	"sync"
)

// create a priority queue using container/heap, backed by an array
type taskQueue []*TaskEntry

// boiler plate from container/heap docs
func (h taskQueue) Len() int           { return len(h) }
func (h taskQueue) Less(i, j int) bool { return h[i].ScheduledTime.Before(h[j].ScheduledTime) }
func (h taskQueue) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *taskQueue) Push(x any) {
	*h = append(*h, x.(*TaskEntry))
}

func (h *taskQueue) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (h taskQueue) Peek() *TaskEntry {
	return h[len(h)-1]
}

// TaskQueue wraps the more complex heap/container implementation
// into a simpler interface that is thread-safe.
type TaskQueue struct {
	sync.RWMutex
	taskqueue *taskQueue
}

func NewTaskQueue() *TaskQueue {
	return &TaskQueue{taskqueue: &taskQueue{}}
}

func (tq *TaskQueue) Push(x *TaskEntry) {
	tq.Lock()
	defer tq.Unlock()
	heap.Push(tq.taskqueue, x)
}

func (tq *TaskQueue) Pop() *TaskEntry {
	tq.Lock()
	defer tq.Unlock()
	if tq.taskqueue.Len() > 0 {
		return heap.Pop(tq.taskqueue).(*TaskEntry)
	}
	return nil
}

func (tq *TaskQueue) Peek() *TaskEntry {
	tq.RLock()
	defer tq.RUnlock()
	if tq.taskqueue.Len() > 0 {
		return tq.taskqueue.Peek()
	}
	return nil
}

func (tq *TaskQueue) Len() int {
	tq.RLock()
	defer tq.RUnlock()
	return tq.taskqueue.Len()
}
