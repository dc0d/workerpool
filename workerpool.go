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

//Starts the worker pool. Initial workers never timeout and never quit.
func (d *WorkerPool) Run() {
	var defaultConf WorkerConfig

	for i := 0; i < d.minWorkers; i++ {
		worker := newWorker(d.workerPool, defaultConf)
		worker.Start()
	}

	go d.dispatch()
}

//Putting more 'Worker's into work. If there is'nt any job to do, and a timeout is set,
//they will simply get timed-out and worker pool will shrink to it's minimum size.
//Default behaviour is they will timeout on conf.Timeout in a sliding manner.
//A quit channel can be used too, to explicitly stop extra workers.
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
	//registration timeout
	Timeout time.Duration
	Quit    chan bool
}

func NewWorkerConfig(timeout time.Duration, quit chan bool) WorkerConfig {
	var conf WorkerConfig
	conf.Timeout = timeout
	conf.Quit = quit

	return conf
}

type worker struct {
	workerPool  chan chan Job
	todoChannel chan Job
	config      WorkerConfig
}

func (w *worker) Start() {
	//TODO: should not start twice (sync.Once)
	go w.startWorking()
}

func (w *worker) startWorking() {
	for {
		var timeout <-chan time.Time
		if w.config.Timeout > 0 {
			timeout = time.After(w.config.Timeout)
		}

		//register this worker in the pool
		select {
		case w.workerPool <- w.todoChannel:
		case <-timeout:
			//failed to register; means WorkerPool is full == there are
			//enough workers with not enough work!
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
