# workerpool
This is an implementation of a workerpool which can get expanded &amp; shrink dynamically. Workers can get added when needed and get dismissed when no longer are needed. Of-course this workerpool can be used just as a simple one:

```go
func main() {
	jobs := make(chan workerpool.Job, 10)
	workerpool.InitNewPool(-1, jobs)

	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		lc := i
		jobs <- workerpool.JobFunc(func() {
			defer wg.Done()
			log.Infof("doing job #%d", lc)
		})
	}
	wg.Wait()
}
```

If a negative value is passed as the minimum number of workers, then the number of CPUs would be used as minimum number.

When a temporary burst comes, we can add workers to the pool with different strategies. We can quit them explicitly or let them work until there are no more jobs to do and they will get timed-out using a sliding timeout, like this:

```go
func main() {
	jobs := make(chan workerpool.Job, 50)
	pool := workerpool.InitNewPool(-1, jobs)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 10000; i++ {
			wg.Add(1)
			lc := i
			jobs <- workerpool.JobFunc(func() {
				defer wg.Done()
				log.Infof("doing job #%d", lc)
			})
		}
	}()

	pool.Expand(450, time.Second*10, nil)

	wg.Wait()
}
```

An absolute timeout is simply a Go idiomatic pattern: closing a channel after a specific time period - using a go-routine.
