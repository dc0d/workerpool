# workerpool
This is an implementation of a workerpool which can get expanded &amp; shrink dynamically. Workers can get added when needed and get dismissed when no longer are needed. Of-course this workerpool can be used just as a simple one:

```go
func main() {
	jobChannel := make(chan workerpool.Job)

	pool := workerpool.New(0, jobChannel)
	pool.Run()

	done := make(chan bool)
	jobChannel <- func() {
		log.Info("Job done!")
		done <- true
	}
	<-done
}
```

If a negative value passes to `New` as the minimum number of workers, then the number of CPUs would be used as minimum number.

When a temporary burst comes, we can add workers to the pool with different strategies. We can quit them explicitly or let them work until there are no more jobs to do and they will get timed-out using a sliding timeout, like this:

```go
func main() {
	jobChannel := make(chan workerpool.Job)

	pool := workerpool.New(0, jobChannel)
	pool.Run()

	done := new(sync.WaitGroup)

	for i := 0; i < 100; i++ {
		jobChannel <- func() {
			done.Add(1)
			defer done.Done()
			//Oh! There is too much work!
			//We need more workers (or less? depending on what you are doing
			//and based on what strategy or algorithm you are using)!
			log.Info("working ...")
		}
	}

	var conf workerpool.WorkerConfig
	//These workers will get dismissed if there are not
	//enough jobs for 30 milliseconds (they are so impatient!).
	conf.Timeout = time.Millisecond * 30
	pool.GrowExtra(10, conf)

	<-time.After(time.Millisecond * 100)

	done.Wait()
}
```

An absolute timeout is simply a Go idiomatic pattern: closing `conf.Quit` after a specific time period - using a go-routine.