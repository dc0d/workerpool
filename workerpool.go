//Package workerpool provides a workerpool that can get expanded & shrink dynamically.
package workerpool

//License: See the LICENSE File.

import (
	"runtime"
	"time"
)

//WorkerPool sample usage:
//  func main() {
//      jobs := make(chan workerpool.Job, 10)
//      workerpool.InitNewPool(-1, jobs)
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
	JobChannel chan Job
}

//Expand is for putting more 'Worker's into work. If there is'nt any job to do, and a timeout is set,
//they will simply get timed-out and worker pool will shrink to it's minimum size.
//Default behaviour is they will timeout on conf.Timeout in a sliding manner.
//A quit channel can be used too, to explicitly stop extra workers.
func (pool *WorkerPool) Expand(n int, timeout time.Duration, quit chan bool) {
	for i := 0; i < n; i++ {
		initWorker(pool.workerPool, timeout, quit)
	}
}

func (pool *WorkerPool) dispatch() {
	for {
		select {
		case job, ok := <-pool.JobChannel:
			if !ok {
				//it means this dispatcher has stopped
				return
			}

			//handle job
			todo := <-pool.workerPool
			todo <- job
		}
	}
}

//InitNewPool creates a new worker pool. If minWorkers is negative, the default number of workers would be runtime.NumCPU().
func InitNewPool(minWorkers int, jobChannel chan Job) *WorkerPool {
	if minWorkers < 0 {
		minWorkers = runtime.NumCPU()
	}

	var _workerPool chan chan Job
	if minWorkers == 0 {
		_workerPool = make(chan chan Job)
	} else {
		_workerPool = make(chan chan Job, minWorkers)
	}

	pool := &WorkerPool{
		workerPool: _workerPool,
		JobChannel: jobChannel,
	}

	for i := 0; i < minWorkers; i++ {
		initWorker(pool.workerPool, 0, nil)
	}

	go pool.dispatch()

	return pool
}

//Job is a job to do, by the workers in the pool
type Job func()

type worker struct {
	workerPool  chan chan Job
	todoChannel chan Job
	timeout     time.Duration
	quit        chan bool
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
		default:
		}

		//register this worker in the pool
		select {
		case w.workerPool <- w.todoChannel:
		case <-timeout:
			//failed to register; means WorkerPool is full == there are
			//enough workers with not enough work!
			return
		case <-w.quit:
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
			//we do not check for timeout or quit here because a registered worker
			//is meant to do his job
			//(& implementing unregistering would be complicated, inefficiet & unnecessary)
		}
	}
}

func initWorker(workerPool chan chan Job, timeout time.Duration, quit chan bool) *worker {
	w := &worker{
		workerPool:  workerPool,
		todoChannel: make(chan Job),
		timeout:     timeout,
		quit:        quit,
	}

	go w.begin()

	return w
}
