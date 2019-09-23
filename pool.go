package gpool

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Gorouting instance which can accept client jobs
type worker struct {
	workerPool chan *worker
	jobChannel chan Job
	stop       chan struct{}
}

func (w *worker) start() {
	go func() {
		var job Job
		for {
			// worker free, add it to pool

			select {
			case job = <-w.jobChannel:
				job()
			case <-w.stop:
				w.stop <- struct{}{}
				return
			}
			w.workerPool <- w

		}
	}()
}

func newWorker(pool chan *worker) *worker {
	return &worker{
		workerPool: pool,
		jobChannel: make(chan Job),
		stop:       make(chan struct{}),
	}
}

// Accepts jobs from clients, and waits for first free worker to deliver job
type dispatcher struct {
	workerPool chan *worker
	jobQueue   chan Job
	stop       chan struct{}
	pool       *Pool
	once       sync.Once
}

func (d *dispatcher) dispatch() {
	timer := time.NewTicker(10* time.Second)
	for {
		select {
		case job := <-d.jobQueue:
			select {
			case worker := <-d.workerPool:
				worker.jobChannel <- job
			default:
				runningWorker := atomic.LoadInt64(&d.pool.runningWorker)
				if runningWorker < d.pool.maxWorkerSize {
					atomic.AddInt64(&d.pool.runningWorker, 1)
					newWorker := newWorker(d.workerPool)
					newWorker.start()
					newWorker.jobChannel <- job

				} else {
					select {
					case worker := <-d.workerPool:
						worker.jobChannel <- job
					case <-d.stop:
						for i := 0; i < len(d.workerPool); i++ {
							worker := <-d.workerPool
							worker.stop <- struct{}{}
							<-worker.stop
						}
						d.stop <- struct{}{}
						return

					}
				}
			}

		case <-timer.C:
			if len(d.jobQueue) < len(d.workerPool) {
				select {
				case w := <-d.workerPool:
					atomic.AddInt64(&d.pool.runningWorker, -1)
					w.stop <- struct{}{}
					<-w.stop
					w.jobChannel = nil
					w.stop = nil
					w = nil

				default:

				}
			}
		case <-d.stop:
			d.once.Do(func() {
				for {
					runningWork:=atomic.LoadInt64(&d.pool.runningWorker)
					if len(d.workerPool) > 0 || runningWork > 0 {
						for i := 0; i < len(d.workerPool); i++ {
							worker := <-d.workerPool
							worker.stop <- struct{}{}
							<-worker.stop
							atomic.AddInt64(&d.pool.runningWorker,-1)
						}
					}else{
						return
					}
				}
			})
			d.stop<- struct{}{}
			return
		}
	}
}

func newDispatcher(workerPool chan *worker, jobQueue chan Job, pool *Pool) *dispatcher {
	d := &dispatcher{
		workerPool: workerPool,
		jobQueue:   jobQueue,
		stop:       make(chan struct{}),
		pool:       pool,
	}
	go d.dispatch()
	return d
}
func (pool *Pool) Running() int64 {
	runningWorker := atomic.LoadInt64(&pool.runningWorker)
	return runningWorker - int64(len(pool.dispatcher.workerPool))
}

// Represents user request, function which should be executed in some worker.
type Job func()

type Pool struct {
	JobQueue      chan Job
	dispatcher    *dispatcher
	wg            sync.WaitGroup
	runningWorker int64
	maxWorkerSize int64
	maxJobSize    int64
}

// Will make pool of gorouting workers.
// numWorkers - how many workers will be created for this pool
// queueLen - how many jobs can we accept until we block
//
// Returned object contains JobQueue reference, which you can use to send job to pool.
func NewPool(numWorkers int, jobQueueLen int) *Pool {
	maxWorkerSize := 100000
	maxJobSize := 100000
	if numWorkers < 0 {
		panic(errors.New("numWorkers is negative"))
	}
	if numWorkers < maxWorkerSize {
		maxWorkerSize = numWorkers
	}
	if jobQueueLen < 0 {
		panic(errors.New("jobQueueLen is negative"))
	}
	if jobQueueLen < maxJobSize {
		maxJobSize = jobQueueLen
	}

	jobQueue := make(chan Job, maxJobSize)
	workerPool := make(chan *worker, maxWorkerSize)
	pool := &Pool{
		JobQueue:      jobQueue,
		maxWorkerSize: int64(maxWorkerSize),
		maxJobSize:    int64(maxJobSize),
	}
	pool.dispatcher = newDispatcher(workerPool, jobQueue, pool)
	return pool
}

// In case you are using WaitAll fn, you should call this method
// every time your job is done.
//
// If you are not using WaitAll then we assume you have your own way of synchronizing.
func (p *Pool) JobDone() {
	p.wg.Done()
}

// How many jobs we should wait when calling WaitAll.
// It is using WaitGroup Add/Done/Wait
func (p *Pool) WaitCount(count int) {
	p.wg.Add(count)
}

// Will wait for all jobs to finish.
func (p *Pool) WaitAll() {
	p.wg.Wait()
}

// Will release resources used by pool
func (p *Pool) Release() {
	p.dispatcher.stop <- struct{}{}
	<-p.dispatcher.stop
}
