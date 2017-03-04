// Package workerpool provides a workerpool. It also  can get expanded & shrink dynamically.
package workerpool

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// License: See the LICENSE File.

// WorkerPool sample usage:
//	func main() {
// 		jobs := make(chan func(), 10)
// 		// for demonstration purpose
// 		myAppCtx, myAppCancel := context.WithCancel(context.Background())
// 		// for example: could get called on SIGINT
// 		_ = myAppCancel
//
// 		pool, _ := workerpool.WithContext(myAppCtx, -1, jobs)
//
// 		for i := 0; i < 10; i++ {
// 			lc := i
// 			jobs <- func() {
// 				log.Printf("doing job #%d", lc)
// 			}
// 		}
//
// 		pool.StopWait()
//	}
type WorkerPool struct {
	workerPool chan chan func()
	jobChannel chan func()

	cancel  func()
	stopped chan struct{}
	wg      sync.WaitGroup
}

// WithContext creates a new worker pool. If minWorkers is negative, the default
// number of workers would be runtime.NumCPU(). When minWorkers is zero, you can
// only use WorkerPool by expanding it.
func WithContext(
	ctx context.Context,
	minWorkers int,
	jobChannel chan func()) (*WorkerPool, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	if minWorkers < 0 {
		minWorkers = runtime.NumCPU()
	}
	pool := &WorkerPool{
		workerPool: make(chan chan func(), minWorkers),
		jobChannel: jobChannel,
		cancel:     cancel,
		stopped:    make(chan struct{}),
		wg:         sync.WaitGroup{},
	}
	for i := 0; i < minWorkers; i++ {
		initWorker(pool.workerPool, 0, nil, pool.stopped, &pool.wg)
	}
	go func() {
		<-ctx.Done()
		close(pool.stopped)
	}()
	go pool.dispatch()
	return pool, ctx
}

// StopWait stops the pool (cancels the context), then waits for
// all workers to quit.
func (pool *WorkerPool) StopWait() {
	pool.cancel()
	pool.wg.Wait()
}

// Expand is for putting more 'Worker's into work. If there is'nt any job to do,
// and a timeout is set, they will simply get timed-out.
// Default behaviour is they will timeout in a sliding manner.
// A quit channel can be used too, to explicitly stop extra workers.
func (pool *WorkerPool) Expand(n int, timeout time.Duration, quit <-chan struct{}) {
	for i := 0; i < n; i++ {
		initWorker(pool.workerPool, timeout, quit, pool.stopped, &pool.wg)
	}
}

func (pool *WorkerPool) dispatch() {
	for {
		select {
		case job, ok := <-pool.jobChannel:
			if !ok {
				// it means this WorkerPool should stop
				pool.cancel()
				return
			}

			//handle job
			todo := <-pool.workerPool
			todo <- job
		case <-pool.stopped:
			return
		}
	}
}

//-----------------------------------------------------------------------------

type worker struct {
	workerPool  chan chan func()
	todoChannel chan func()
	timeout     time.Duration
	quit        <-chan struct{}
	poolQuit    <-chan struct{}
}

func (w *worker) begin(wg *sync.WaitGroup) {
	defer wg.Done()
	var timeout <-chan time.Time

	for {
		select {
		case <-w.quit:
			return
		case <-w.poolQuit:
			return
		case <-timeout:
			return
		default:
		}

		if w.timeout > 0 {
			timeout = time.After(w.timeout)
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
				job()
			}
			// we do not check for timeout or quit here because a registered worker
			// is meant to do his job
			// (& implementing unregistering would be complicated, inefficiet & unnecessary)
			// unless the whole pool is quit (a prototype implemented using a priority queue
			// - a heap - but it was just more complicated and did not add much; should
			// investigate it more deeply; but this just works fine; after the burst,
			// the expanded workers would just do their last job, eventually).
		case <-w.poolQuit:
			return
		}
	}
}

func initWorker(
	workerPool chan chan func(),
	timeout time.Duration,
	quit <-chan struct{},
	poolQuit <-chan struct{},
	wg *sync.WaitGroup) *worker {
	w := &worker{
		workerPool:  workerPool,
		todoChannel: make(chan func()),
		timeout:     timeout,
		quit:        quit,
		poolQuit:    poolQuit,
	}

	wg.Add(1)
	go w.begin(wg)

	return w
}

//-----------------------------------------------------------------------------
