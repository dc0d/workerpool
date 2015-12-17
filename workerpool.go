//A workerpool that can get expanded & shrink dynamically.
package workerpool

//License: See the LICENSE File.

import (
	"runtime"
	"time"
)

//Sample usage:
//	func main() {
//		jobChannel := make(chan workerpool.Job)
//
//		pool := workerpool.New(0, jobChannel)
//		pool.Run()
//
//		done := make(chan bool)
//		jobChannel <- func() {
//			log.Info("Job done!")
//			done <- true
//		}
//		<-done
//	}
type WorkerPool struct {
	workerPool chan chan Job
	JobChannel chan Job
	minWorkers int
}

//Starts the worker pool. Initial workers never timeout (well; they do; after a year of unemployment!) and never quit.
func (d *WorkerPool) Run() {
	var defaultConf WorkerConfig
	defaultConf.Registration.Timeout = time.Hour * 24 * 365

	for i := 0; i < d.minWorkers; i++ {
		worker := newWorker(d.workerPool, defaultConf)
		worker.Start()
	}

	go d.dispatch()
}

//Putting more 'Worker's into work. If there is'nt any job to do,
//they will simply get timed-out and worker pool will shrink to it's minimum size.
//Default behaviour is they will timeout on conf.Registration.Timeout in a sliding manner.
func (d *WorkerPool) GrowExtra(n int, conf WorkerConfig) {
	for i := 0; i < n; i++ {
		worker := newWorker(d.workerPool, conf)
		worker.Start()
	}
}

func (d *WorkerPool) dispatch() {
	for {
		select {
		case job, ok := <-d.JobChannel:
			if !ok {
				//it means this dispatcher has stopped
				return
			}

			//memory hungry version
			//go d.handleJob(job)

			d.handleJob(job)
		}
	}
}

func (d *WorkerPool) handleJob(j Job) {
	todo := <-d.workerPool
	todo <- j
}

//Creates a new worker pool. If minWorkers is 0 or negative, the default number of workers would be runtime.NumCPU().
func New(minWorkers int, jobChannel chan Job) *WorkerPool {
	if minWorkers <= 0 {
		minWorkers = runtime.NumCPU()
	}

	return &WorkerPool{
		workerPool: make(chan chan Job, minWorkers),
		JobChannel: jobChannel,
		minWorkers: minWorkers,
	}
}

type Job func()

//Configuration for extra workers.
type WorkerConfig struct {
	Registration struct {
		Timeout    time.Duration
		IsAbsolute bool
	}
	Quit chan bool
}

func NewWorkerConfig(timeout time.Duration, isAbsolute bool, quit chan bool) WorkerConfig {
	var conf WorkerConfig
	conf.Registration.Timeout = timeout
	conf.Registration.IsAbsolute = isAbsolute
	conf.Quit = quit

	return conf
}

type worker struct {
	workerPool  chan chan Job
	todoChannel chan Job
	config      WorkerConfig
}

func (w *worker) Start() {
	go w.startWorking()
}

func (w *worker) startWorking() {
	var absoluteTimeout <-chan time.Time
	if w.config.Registration.IsAbsolute {
		absoluteTimeout = time.After(w.config.Registration.Timeout)
	}

	for {
		//register this worker in the pool
		select {
		case w.workerPool <- w.todoChannel:
		case <-time.After(w.config.Registration.Timeout):
			//failed to register; means WorkerPool is full == there are
			//enough workers with not enough work!
			return
		case <-absoluteTimeout:
			return
		case <-w.config.Quit:
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
		case <-w.config.Quit:
			return
		}
	}
}

func newWorker(workerPool chan chan Job, conf WorkerConfig) *worker {
	return &worker{
		workerPool:  workerPool,
		todoChannel: make(chan Job),
		config:      conf,
	}
}
