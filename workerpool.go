// Package workerpool provides a workerpool. It also can expand and shrink dynamically.
package workerpool

import (
	"runtime"
	"sync"
	"time"
)

// License: See the LICENSE File.

//-----------------------------------------------------------------------------

// WorkerPool provides a pool of workers.
type WorkerPool struct {
	pool chan chan func()
	jobs chan func()

	quit     chan struct{}
	quitOnce sync.Once
	wg       sync.WaitGroup
}

// New makes a new *WorkerPool.
func New(workers int, jobQueue ...int) *WorkerPool {
	q := 0
	if len(jobQueue) > 0 && jobQueue[0] > 0 {
		q = jobQueue[0]
	}
	if workers < 0 {
		workers = runtime.NumCPU()
	}
	pool := WorkerPool{
		pool: make(chan chan func(), workers),
		jobs: make(chan func(), q),
		quit: make(chan struct{}),
		wg:   sync.WaitGroup{},
	}
	for i := 0; i < workers; i++ {
		initWorker(pool.pool, 0, nil, pool.quit, &pool.wg)
	}
	go pool.dispatch()
	return &pool
}

// Queue queues a job to be run by a worker.
func (pool *WorkerPool) Queue(job func(), timeout ...time.Duration) bool {
	if pool.stopped() {
		return false
	}
	var t <-chan time.Time
	if len(timeout) > 0 && timeout[0] > 0 {
		t = time.After(timeout[0])
	}
	select {
	case pool.jobs <- job:
	case <-t:
		return false
	case <-pool.quit:
		return false
	}
	return true
}

// Stop stops the pool and waits for all workers to return.
func (pool *WorkerPool) Stop() {
	pool.quitOnce.Do(func() { close(pool.quit) })
	pool.wg.Wait()
}

// Expand is for putting more 'Worker's into work. If there is'nt any job to do,
// and a timeout is set, they will simply get timed-out.
// Default behaviour is they will timeout in a sliding manner.
// A quit channel can be used too, to explicitly stop extra workers.
//
// One firend noted that there might be a *temporary* goroutine leak, when expanding
// the worker pool, using timeouts. Actually is's not a goroutine leak because
// it's always bound to the size of pool and has a deterministic behavior.
// Assuming we have a worker pool of size 10, and we expand it with a 1000 extra workers,
// that would timeout after 100 mili-seconds, we may see (after some 100 mili-second)
// there remains twice the initial size of the pool (10 * 2) number of goroutines - which
// of-cource would get timedout after doing some extra jobs and the pool will shrink
// to it's initial size. The reason for this temporary expanded lives of some extra
// workers is, the initial workers may fail to register before those extra workers.
// So we will have 10 registered extra workers, plus 10 unregistered initial workers.
// And the rest of extra workers will get timedout because they fail to register.
// So we have 20 goroutines in the pool at max, in this very specific situation,
// which will evantually get timed out. That's not a goroutine leak (it described
// as *temporary* in the first place) but it was entertaining to find out why and
// how that happens! A test named `TestTimeoutNoGoroutineLeak(...)` is added to
// descibe this in code.
func (pool *WorkerPool) Expand(n int, timeout time.Duration, quit <-chan struct{}) bool {
	if pool.stopped() {
		return false
	}
	for i := 0; i < n; i++ {
		initWorker(pool.pool, timeout, quit, pool.quit, &pool.wg)
	}
	return true
}

func (pool *WorkerPool) stopped() bool {
	return stopped(pool.quit)
}

func (pool *WorkerPool) dispatch() {
	for {
		select {
		case job := <-pool.jobs:
			//handle job
			todo := <-pool.pool
			todo <- job
		case <-pool.quit:
			return
		}
	}
}

//-----------------------------------------------------------------------------

type worker struct {
	pool     chan chan func()
	poolQuit <-chan struct{}
	todo     chan func()
	timeout  time.Duration
	quit     <-chan struct{}
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
		case w.pool <- w.todo:
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
		case job, ok := <-w.todo:
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
	pool chan chan func(),
	timeout time.Duration,
	quit <-chan struct{},
	poolQuit <-chan struct{},
	wg *sync.WaitGroup) *worker {
	if stopped(poolQuit) {
		return nil
	}
	w := &worker{
		pool:     pool,
		todo:     make(chan func()),
		timeout:  timeout,
		quit:     quit,
		poolQuit: poolQuit,
	}

	wg.Add(1)
	go w.begin(wg)

	return w
}

//-----------------------------------------------------------------------------

func stopped(c <-chan struct{}) bool {
	ok := true
	select {
	case _, ok = <-c:
	default:
	}
	return !ok
}

//-----------------------------------------------------------------------------

// // WorkerPool sample usage:
// //	func main() {
// // 		jobs := make(chan func(), 10)
// // 		// for demonstration purpose
// // 		myAppCtx, myAppCancel := context.WithCancel(context.Background())
// // 		// for example: could get called on SIGINT
// // 		_ = myAppCancel
// //
// // 		pool, _ := workerpool.WithContext(myAppCtx, -1, jobs)
// //
// // 		for i := 0; i < 10; i++ {
// // 			lc := i
// // 			jobs <- func() {
// // 				log.Printf("doing job #%d", lc)
// // 			}
// // 		}
// //
// // 		pool.StopWait()
// //	}
// type WorkerPool struct {
// 	workerPool chan chan func()
// 	jobChannel chan func()

// 	cancel  func()
// 	stopped chan struct{}
// 	wg      sync.WaitGroup
// }

// // WithContext creates a new worker pool. If minWorkers is negative, the default
// // number of workers would be runtime.NumCPU(). When minWorkers is zero, you can
// // only use WorkerPool by expanding it.
// func WithContext(
// 	ctx context.Context,
// 	minWorkers int,
// 	jobChannel chan func()) (*WorkerPool, context.Context) {
// 	ctx, cancel := context.WithCancel(ctx)
// 	if minWorkers < 0 {
// 		minWorkers = runtime.NumCPU()
// 	}
// 	pool := &WorkerPool{
// 		workerPool: make(chan chan func(), minWorkers),
// 		jobChannel: jobChannel,
// 		cancel:     cancel,
// 		stopped:    make(chan struct{}),
// 		wg:         sync.WaitGroup{},
// 	}
// 	for i := 0; i < minWorkers; i++ {
// 		initWorker(pool.workerPool, 0, nil, pool.stopped, &pool.wg)
// 	}
// 	go func() {
// 		<-ctx.Done()
// 		close(pool.stopped)
// 	}()
// 	go pool.dispatch()
// 	return pool, ctx
// }

// // StopWait stops the pool (cancels the context), then waits for
// // all workers to quit.
// func (pool *WorkerPool) StopWait() {
// 	pool.cancel()
// 	pool.wg.Wait()
// }

// // Expand is for putting more 'Worker's into work. If there is'nt any job to do,
// // and a timeout is set, they will simply get timed-out.
// // Default behaviour is they will timeout in a sliding manner.
// // A quit channel can be used too, to explicitly stop extra workers.
// func (pool *WorkerPool) Expand(n int, timeout time.Duration, quit <-chan struct{}) {
// 	for i := 0; i < n; i++ {
// 		initWorker(pool.workerPool, timeout, quit, pool.stopped, &pool.wg)
// 	}
// }

// func (pool *WorkerPool) dispatch() {
// 	for {
// 		select {
// 		case job, ok := <-pool.jobChannel:
// 			if !ok {
// 				// it means this WorkerPool should stop
// 				pool.cancel()
// 				return
// 			}

// 			//handle job
// 			todo := <-pool.workerPool
// 			todo <- job
// 		case <-pool.stopped:
// 			return
// 		}
// 	}
// }

// //-----------------------------------------------------------------------------

// type worker struct {
// 	workerPool  chan chan func()
// 	todoChannel chan func()
// 	timeout     time.Duration
// 	quit        <-chan struct{}
// 	poolQuit    <-chan struct{}
// }

// func (w *worker) begin(wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	var timeout <-chan time.Time

// 	for {
// 		select {
// 		case <-w.quit:
// 			return
// 		case <-w.poolQuit:
// 			return
// 		case <-timeout:
// 			return
// 		default:
// 		}

// 		if w.timeout > 0 {
// 			timeout = time.After(w.timeout)
// 		}

// 		// register this worker in the pool
// 		select {
// 		case w.workerPool <- w.todoChannel:
// 		case <-timeout:
// 			//failed to register; means WorkerPool is full == there are
// 			//enough workers with not enough work!
// 			return
// 		case <-w.quit:
// 			return
// 		case <-w.poolQuit:
// 			return
// 		}

// 		select {
// 		case job, ok := <-w.todoChannel:
// 			if !ok {
// 				return
// 			}

// 			if job != nil {
// 				job()
// 			}
// 			// we do not check for timeout or quit here because a registered worker
// 			// is meant to do his job
// 			// (& implementing unregistering would be complicated, inefficiet & unnecessary)
// 			// unless the whole pool is quit (a prototype implemented using a priority queue
// 			// - a heap - but it was just more complicated and did not add much; should
// 			// investigate it more deeply; but this just works fine; after the burst,
// 			// the expanded workers would just do their last job, eventually).
// 		case <-w.poolQuit:
// 			return
// 		}
// 	}
// }

// func initWorker(
// 	workerPool chan chan func(),
// 	timeout time.Duration,
// 	quit <-chan struct{},
// 	poolQuit <-chan struct{},
// 	wg *sync.WaitGroup) *worker {
// 	w := &worker{
// 		workerPool:  workerPool,
// 		todoChannel: make(chan func()),
// 		timeout:     timeout,
// 		quit:        quit,
// 		poolQuit:    poolQuit,
// 	}

// 	wg.Add(1)
// 	go w.begin(wg)

// 	return w
// }

// //-----------------------------------------------------------------------------
