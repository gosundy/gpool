package gpool

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	RunTimes      = 1000000
	benchParam    = 10
	benchAntsSize = 200000
)

func init() {
	println("using MAXPROC")
	numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPUs)
}

func demoPoolFunc(args interface{}) {
	n := args.(int)
	time.Sleep(time.Duration(n) * time.Millisecond)
}
func TestNewWorker(t *testing.T) {
	pool := make(chan *worker)
	worker := newWorker(pool)
	worker.start()
	worker.jobChannel <- func() {}
	assert.NotNil(t, worker)

	worker = <-pool
	assert.NotNil(t, worker, "Worker should register itself to the pool")

	called := false
	done := make(chan bool)

	job := func() {
		called = true
		done <- true
	}

	worker.jobChannel <- job
	<-done
	assert.Equal(t, true, called)
}

func TestNewPool(t *testing.T) {
	pool := NewPool(1000, 10000)
	defer pool.Release()

	iterations := 1000000
	pool.WaitCount(iterations)
	var counter uint64 = 0

	for i := 0; i < iterations; i++ {
		arg := uint64(1)

		job := func() {
			defer pool.JobDone()
			atomic.AddUint64(&counter, arg)
			assert.Equal(t, uint64(1), arg)
		}

		pool.JobQueue <- job
	}

	pool.WaitAll()

	counterFinal := atomic.LoadUint64(&counter)
	assert.Equal(t, uint64(iterations), counterFinal)
}

func TestRelease(t *testing.T) {
	grNum := runtime.NumGoroutine()
	pool := NewPool(1000, 1000)
	defer func() {
		pool.Release()
		// give some time for all goroutines to quit
		assert.Equal(t, grNum, runtime.NumGoroutine(), "All goroutines should be released after Release() call")
	}()

	pool.WaitCount(1000000)

	for i := 0; i < 1000000; i++ {
		job := func() {
			defer pool.JobDone()
		}

		pool.JobQueue <- job
	}

	pool.WaitAll()
}
func BenchmarkPool(b *testing.B) {
	// Testing with just 1 goroutine
	// to benchmark the non-parallel part of the code
	pool := NewPool(1, 10)
	defer pool.Release()

	log.SetOutput(ioutil.Discard)

	for n := 0; n < b.N; n++ {
		pool.JobQueue <- func() {
			log.Printf("I am worker! Number %d\n", n)
		}
	}
}
func BenchmarkGoroutineWithFunc(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			go func() {
				demoPoolFunc(benchParam)
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkSemaphoreWithFunc(b *testing.B) {
	var wg sync.WaitGroup
	sema := make(chan struct{}, benchAntsSize)

	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			sema <- struct{}{}
			go func() {
				demoPoolFunc(benchParam)
				<-sema
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkGoPoolWithFunc(b *testing.B) {
	var wg sync.WaitGroup

	pool := NewPool(100000, 100000)
	defer pool.Release()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		pool.WaitCount(RunTimes)
		for j := 0; j < RunTimes; j++ {
			pool.JobQueue <- func() {
				defer pool.JobDone()
				demoPoolFunc(benchParam)
			}
		}
		wg.Wait()
	}
	b.StopTimer()
}
