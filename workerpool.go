// Package workerpool provides a workerpool that can get expanded & shrink dynamically.
package workerpool

// License: See the LICENSE File.

import (
	"runtime"
	"sync"
	"time"
)

//-----------------------------------------------------------------------------

// WorkerPool sample usage:
//  func main() {
//      jobs := make(chan workerpool.Job, 10)
//      workerpool.NewWorkerPool(-1, jobs)
//
//      wg := &sync.WaitGroup{}
//      for i := 0; i < 10; i++ {
//          wg.Add(1)
//          lc := i
//          jobs <- func() {
//              defer wg.Done()
//              log.Infof("doing job #%d", lc)
//          }
//      }
//      wg.Wait()
//  }
type WorkerPool struct {
	workerPool chan chan Job
	jobChannel chan Job

	quit     chan struct{}
	quitOnce *sync.Once
}

// NewWorkerPool creates a new worker pool. If minWorkers is negative, the default
// number of workers would be runtime.NumCPU(). When minWorkers is zero, you can
// only use WorkerPool by expanding it.
func NewWorkerPool(minWorkers int, jobChannel chan Job) *WorkerPool {
	if minWorkers < 0 {
		minWorkers = runtime.NumCPU()
	}

	pool := &WorkerPool{
		workerPool: make(chan chan Job, minWorkers),
		jobChannel: jobChannel,
		quit:       make(chan struct{}),
		quitOnce:   new(sync.Once),
	}

	for i := 0; i < minWorkers; i++ {
		initWorker(pool.workerPool, 0, nil, pool.quit)
	}

	go pool.dispatch()

	return pool
}

// Stop stops the pool
func (pool *WorkerPool) Stop() { pool.quitOnce.Do(func() { close(pool.quit) }) }

// Expand is for putting more 'Worker's into work. If there is'nt any job to do, and a timeout is set,
// they will simply get timed-out and worker pool will shrink to it's minimum size.
// Default behaviour is they will timeout on conf.Timeout in a sliding manner.
// A quit channel can be used too, to explicitly stop extra workers.
func (pool *WorkerPool) Expand(n int, timeout time.Duration, quit chan struct{}) {
	for i := 0; i < n; i++ {
		initWorker(pool.workerPool, timeout, quit, pool.quit)
	}
}

func (pool *WorkerPool) dispatch() {
	for {
		select {
		case job, ok := <-pool.jobChannel:
			if !ok {
				// it means this WorkerPool should stop
				pool.Stop()
				return
			}

			//handle job
			todo := <-pool.workerPool
			todo <- job
		case <-pool.quit:
			return
		}
	}
}

//-----------------------------------------------------------------------------

// Job is a job to do, by the workers in the pool
type Job interface {
	Do()
}

//-----------------------------------------------------------------------------

// JobFunc wraps a func() as Job
type JobFunc func()

// Do implements Job
func (wf JobFunc) Do() {
	wf()
}

//-----------------------------------------------------------------------------

type worker struct {
	workerPool  chan chan Job
	todoChannel chan Job
	timeout     time.Duration
	quit        chan struct{}
	poolQuit    chan struct{}
}

func (w *worker) begin() {
	for {
		var timeout <-chan time.Time
		if w.timeout > 0 {
			timeout = time.After(w.timeout)
		}

		select {
		case <-w.quit:
			return
		case <-w.poolQuit:
			return
		default:
		}

		// register this worker in the pool
		select {
		case w.workerPool <- w.todoChannel:
		case <-timeout:
			//failed to register; means WorkerPool is full == there are
			//enough workers with not enough work!
			return
		case <-w.quit:
			return
		case <-w.poolQuit:
			return
		}

		select {
		case job, ok := <-w.todoChannel:
			if !ok {
				return
			}

			if job != nil {
				job.Do()
			}
			// we do not check for timeout or quit here because a registered worker
			// is meant to do his job
			// (& implementing unregistering would be complicated, inefficiet & unnecessary)
			// unless the whole pool is quit.
		case <-w.poolQuit:
			return
		}
	}
}

func initWorker(workerPool chan chan Job, timeout time.Duration, quit chan struct{}, poolQuit chan struct{}) *worker {
	w := &worker{
		workerPool:  workerPool,
		todoChannel: make(chan Job),
		timeout:     timeout,
		quit:        quit,
		poolQuit:    poolQuit,
	}

	go w.begin()

	return w
}
