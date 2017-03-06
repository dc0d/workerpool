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

## Update

`master` now points to `v2` branch which is a redesign of the `workerpool` package employing the standard context pattern (using `context.Context`).

You can get `v2` by:

```
go get gopkg.in/dc0d/workerpool.v2
```

Or from the `master` branch.