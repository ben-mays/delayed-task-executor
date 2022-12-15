package future

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func assert(t *testing.T, expected, actual any) {
	if expected != actual {
		t.Fatal(fmt.Sprintf("%v does not equal %v\n", expected, actual))
	}
}

func TestFuture(t *testing.T) {
	f1 := NewFutureFn(func() (any, error) {
		return 1, nil
	})
	f2 := NewFutureFn(func() (any, error) {
		return 2, nil
	})
	f3 := NewFutureFn(func() (any, error) {
		return 3, errors.New("test")
	})
	res1, err1 := f1.WaitAndGetResult()
	res2, err2 := f2.WaitAndGetResult()
	res3, err3 := f3.WaitAndGetResult()
	assert(t, res1, 1)
	assert(t, err1, nil)
	assert(t, res2, 2)
	assert(t, err2, nil)
	assert(t, res3, 3)
	assert(t, err3.Error(), "test")
}

// TODO: Refactor into test matrix
func TestFutureConcurrently(t *testing.T) {
	wg := &sync.WaitGroup{}
	fn := func(f Future, wg *sync.WaitGroup) {
		res, err := f.WaitAndGetResult()
		assert(t, res, 1)
		assert(t, err, nil)
		wg.Done()
	}
	future := NewFutureFn(func() (any, error) {
		jitter := time.Millisecond * time.Duration(rand.Intn(1000))
		time.Sleep(time.Second + jitter)
		return 1, nil
	})
	for i := 0; i < 5000; i++ {
		wg.Add(1)
		go fn(future, wg)
	}
	wg.Wait()
}

func TestFutureConcurrentlyWithIntermittentBlocking(t *testing.T) {
	wg := &sync.WaitGroup{}
	fn := func(f Future, wg *sync.WaitGroup) {
		// only call Wait on 50% of futures
		if rand.Float32() < .5 {
			res, err := f.WaitAndGetResult()
			assert(t, res, 1)
			assert(t, err, nil)
		}
		wg.Done()
	}
	future := NewFutureFn(func() (any, error) {
		jitter := time.Millisecond * time.Duration(rand.Intn(1000))
		time.Sleep(time.Second + jitter)
		return 1, nil
	})
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		go fn(future, wg)
	}
	wg.Wait()
}
