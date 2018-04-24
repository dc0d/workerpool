# workerpool

[![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT) [![GoDoc](https://godoc.org/github.com/dc0d/workerpool?status.svg)](http://godoc.org/github.com/dc0d/workerpool) [![Go Report Card](https://goreportcard.com/badge/github.com/dc0d/workerpool)](https://goreportcard.com/report/github.com/dc0d/workerpool) [![Build Status](https://travis-ci.org/dc0d/workerpool.svg?branch=master)](http://travis-ci.org/dc0d/workerpool) [![codecov](https://codecov.io/gh/dc0d/workerpool/branch/master/graph/badge.svg)](https://codecov.io/gh/dc0d/workerpool)

This is an implementation of a workerpool which can get expanded &amp; shrink dynamically. Workers can get added when needed and get dismissed when no longer are needed. Of-course this workerpool can be used just as a simple one with a fixed size:

```go
func main() {
	jobs := make(chan func(), 10)
	// for demonstration purpose
	myAppCtx, myAppCancel := context.WithCancel(context.Background())
	// for example: could get called on SIGINT
	_ = myAppCancel

	pool, _ := workerpool.WithContext(myAppCtx, -1, jobs)

	for i := 0; i < 10; i++ {
		lc := i
		jobs <- func() {
			log.Printf("doing job #%d", lc)
		}
	}

	pool.StopWait()
}
```

If a negative value is passed as the minimum number of workers, then the number of CPUs would be used as minimum number. We can stop the worker pool either by calling `pool.StopWait()` or closing the input job channel (if we want to wait for all the worker goroutines to stop, we should call `pool.StopWait()`).

When a temporary burst comes, we can add workers to the pool with different strategies. We can quit them explicitly or let them work until there are no more jobs to do and they will get timed-out in a sliding (timeout) manner and would just do their last job, eventually, like this:

```go
func main() {
	jobs := make(chan func(), 10)
	// for demonstration purpose
	myAppCtx, myAppCancel := context.WithCancel(context.Background())
	// for example: could get called on SIGINT
	_ = myAppCancel

	pool, _ := workerpool.WithContext(myAppCtx, -1, jobs)

	// a WaitGroup for our jobs (workerpool use it's own WaitGroup for it's
	// workers)
	wgJobs := &sync.WaitGroup{}
	wgJobs.Add(1)
	go func() {
		defer wgJobs.Done()

		for i := 0; i < 10000; i++ {
			wgJobs.Add(1)
			lc := i
			jobs <- func() {
				defer wgJobs.Done()
				log.Printf("doing job #%d", lc)
			}
		}
	}()

	pool.Expand(1000, time.Second*10, nil)
	wgJobs.Wait()
	pool.StopWait()
}
```

An absolute timeout is simply a Go idiomatic pattern: closing a channel after a specific time period - using a go-routine.

## Note
One firend noted that there might be a *temporary* goroutine leak, when expanding the worker pool, using timeouts. Actually is's not a goroutine leak because it's always bound to the size of pool and has a deterministic behavior. Assuming we have a worker pool of size 10, and we expand it with a 1000 extra workers, that would timeout after 100 mili-seconds, we may see (after some 100 mili-second) there remains twice the initial size of the pool (10 * 2) number of goroutines - which of-cource would get timedout after doing some extra jobs and the pool will shrink to it's initial size. The reason for this temporary expanded lives of some extra workers is, the initial workers may fail to register before those extra workers. So we will have 10 registered extra workers, plus 10 unregistered initial workers. And the rest of extra workers will get timedout because they fail to register. So we have 20 goroutines in the pool at max, in this very specific situation, which will evantually get timed out. That's not a goroutine leak (it described as *temporary* in the first place) but it was entertaining to find out why and how that happens! A test named `TestTimeoutNoGoroutineLeak(...)` is added to descibe this in code.