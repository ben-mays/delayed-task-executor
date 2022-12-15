package future

import (
	"sync"
)

type Future interface {
	WaitAndGetResult() (any, error)
}

type FutureResult struct {
	Result any
	Error  error
}

type future struct {
	resultch    chan FutureResult
	readBarrier *sync.WaitGroup // we use a waitgroup initialized to 1 as a memory barrier
	result      *FutureResult
}

func (f *future) handler() {
	res := <-f.resultch
	f.result = &res
	f.readBarrier.Done()
}

// NewFuture returns a new Future that listens for a FutureResult on the channel. Note, the Future will not close the
// channel but only the first value will be used.
func NewFuture(resultch chan FutureResult) Future {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	f := &future{resultch, wg, nil}
	go f.handler()
	return f
}

// NewFutureFn returns a Future that executes the given function immediately.
func NewFutureFn(fn func() (any, error)) Future {
	resultch := make(chan FutureResult)
	go func(f func() (any, error), ch chan FutureResult) {
		res, err := f()
		ch <- FutureResult{res, err}
		close(ch)
	}(fn, resultch)
	return NewFuture(resultch)
}

// WaitAndGetResults will return immediately if a value is present or block until it is available.
func (f *future) WaitAndGetResult() (any, error) {
	f.readBarrier.Wait()
	return f.result.Result, f.result.Error
}
