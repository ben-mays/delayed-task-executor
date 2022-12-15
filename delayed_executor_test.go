package temporal

import (
	"delayed-task-exec/future"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type testTask struct{}

func (testTask) Run() (int, error) {
	return rand.Int(), nil
}

func run(workers, schedulers, numBatches, batchSize, delayJitter int) {
	metrics := &MetricsListener{}
	executor := NewTaskExecutor(workers, schedulers, WithResultListener(metrics))
	futures := []future.Future{}
	for i := 0; i < numBatches; i++ {
		for j := 0; j < batchSize; j++ {
			delay := time.Millisecond * time.Duration(rand.Intn(delayJitter))
			futures = append(futures, executor.Submit(delay, testTask{}))
		}
	}
	for _, f := range futures {
		f.WaitAndGetResult()
	}
}

func TestExecutor(t *testing.T) {
	metrics := &MetricsListener{}
	executor := NewTaskExecutor(1000, 10, WithResultListener(metrics),
		WithBuffer(10000),
		WithSchedulerPeriod(time.Millisecond))
	futures := []future.Future{}
	batchSizeLimit := 1000000
	for i := 0; i < 10; i++ {
		batchSize := rand.Intn(batchSizeLimit)
		fmt.Printf("submitting %d tasks\n", batchSize)
		for j := 0; j < batchSize; j++ {
			delay := time.Millisecond * time.Duration(rand.Intn(500))
			futures = append(futures, executor.Submit(delay, testTask{}))
		}
	}
	fmt.Printf("waiting on %d futures\n", len(futures))
	for i, f := range futures {
		if i%batchSizeLimit == 0 {
			fmt.Printf("processed %d futures\n", i)
		}
		f.WaitAndGetResult()
	}
	fmt.Printf("✅ %v \n", metrics.Stats())
}

func BenchmarkExecutor(b *testing.B) {
	for n := 0; n < b.N; n++ {
		run(1000, 3, 100, 500000, 200)
	}
}
