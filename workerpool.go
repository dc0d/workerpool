// Package workerpool provides a workerpool. It also can expand and shrink dynamically.
//
// Jobs can be queued using the Queue() method which also accepts a timeout parameter for
// timing out queuing and if all workers are too busy.
//
// For expanding the queue, Expand() method can be used, which increases the number of workers.
// If a timeout is provided, these extra workers will stop, if there are not enough jobs to do.
// It is also possible to explicitly stop extra workers by providing a quit channel.
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
func New(workerCount, jobQueueSize int) *WorkerPool {
	if jobQueueSize < 0 {
		jobQueueSize = 0
	}
	if workerCount < 0 {
		workerCount = runtime.NumCPU()
	}
	pool := WorkerPool{
		pool: make(chan chan func(), workerCount),
		jobs: make(chan func(), jobQueueSize),
		quit: make(chan struct{}),
		wg:   sync.WaitGroup{},
	}
	for i := 0; i < workerCount; i++ {
		var builder workerBuilder
		w := builder.
			withPool(pool.pool).
			withPoolQuit(pool.quit).
			withTimeout(0).
			withQuit(pool.quit).
			build()
		w.initWorker(&pool.wg)
	}
	go pool.dispatch()
	return &pool
}

// Queue queues a job to be run by a worker.
func (pool *WorkerPool) Queue(job func(), timeout time.Duration) bool {
	if pool.stopped() {
		return false
	}
	var t <-chan time.Time
	if timeout > 0 {
		t = time.After(timeout)
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
// Default behavior is they will timeout in a sliding manner.
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
		var builder workerBuilder
		w := builder.
			withPool(pool.pool).
			withPoolQuit(pool.quit).
			withTimeout(timeout).
			withQuit(pool.quit).
			build()
		w.initWorker(&pool.wg)
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
			// handle job
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		var timeout <-chan time.Time

		for {
			if w.shouldQuit(timeout) {
				return
			}

			if w.timeout > 0 {
				timeout = time.After(w.timeout)
			}

			if !w.registerInPool(timeout) {
				return
			}

			if !w.executeJob() {
				return
			}
		}
	}()
}

func (w *worker) shouldQuit(timeout <-chan time.Time) (ok bool) {
	select {
	case <-w.quit:
		return true
	case <-w.poolQuit:
		return true
	case <-timeout:
		return true
	default:
	}

	return false
}

func (w *worker) registerInPool(timeout <-chan time.Time) (ok bool) {
	// register this worker in the pool
	select {
	case w.pool <- w.todo:
		return true
	case <-timeout:
		// failed to register; means WorkerPool is full == there are
		// enough workers with not enough work!
		return false
	case <-w.quit:
		return false
	case <-w.poolQuit:
		return false
	}
}

func (w *worker) executeJob() (ok bool) {
	select {
	case job, ok := <-w.todo:
		if !ok {
			return false
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
		return false
	}

	return true
}

func (w *worker) initWorker(wg *sync.WaitGroup) {
	if stopped(w.poolQuit) {
		return
	}

	w.begin(wg)
}

type workerBuilder worker

func (builder workerBuilder) withPool(pool chan chan func()) workerBuilder {
	builder.pool = pool
	return builder
}
func (builder workerBuilder) withPoolQuit(poolQuit <-chan struct{}) workerBuilder {
	builder.poolQuit = poolQuit
	return builder
}
func (builder workerBuilder) withTimeout(timeout time.Duration) workerBuilder {
	builder.timeout = timeout
	return builder
}
func (builder workerBuilder) withQuit(quit <-chan struct{}) workerBuilder {
	builder.quit = quit
	return builder
}
func (builder workerBuilder) build() *worker {
	builder.todo = make(chan func())
	w := worker(builder)
	return &w
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
