package workerpool

import (
	"context"
	"errors"
	"math"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestNegWorkers(t *testing.T) {
	jobChannel := make(chan func())
	pool, ctx := WithContext(context.Background(), -1, jobChannel)

	n := int64(runtime.NumCPU())
	var backSlot int64
	var job = func() {
		atomic.AddInt64(&backSlot, 1)
		select {
		case <-ctx.Done():
		}
	}
OUT1:
	for {
		select {
		case jobChannel <- job:
		case <-time.After(time.Millisecond * 300):
			break OUT1
		}
	}

	actual := atomic.LoadInt64(&backSlot)
	if actual != n {
		t.Log(actual)
		t.Fail()
	}

	close(jobChannel)
	err := waitFunc(pool.StopWait, _timeout)
	if err != nil {
		t.Fail()
	}
}

func TestZeroWorkers(t *testing.T) {
	jobChannel := make(chan func())
	pool, _ := WithContext(context.Background(), 0, jobChannel)

	var backSlot int64 = 10
	var job = func() {
		atomic.StoreInt64(&backSlot, 110)
	}

	select {
	case jobChannel <- job:
	case <-time.After(time.Millisecond * 300):
	}
	if atomic.LoadInt64(&backSlot) != 10 {
		t.Fail()
	}

	pool.Expand(1, 0, nil)

	done := make(chan bool)
	job = func() {
		defer close(done)
		atomic.StoreInt64(&backSlot, 73)
	}
	select {
	case jobChannel <- job:
	case <-time.After(time.Millisecond * 300):
		t.Fail()
	}
	<-done

	if atomic.LoadInt64(&backSlot) != 73 {
		t.Fail()
	}

	err := waitFunc(pool.StopWait, _timeout)
	if err != nil {
		t.Fail()
	}
}

func TestAbsoluteTimeout(t *testing.T) {
	initialWorkers := 1
	extraWorkers := 10

	jobChannel := make(chan func(), 2)

	pool, _ := WithContext(context.Background(), initialWorkers, jobChannel)

	quit1 := make(chan struct{})
	pool.Expand(extraWorkers, 0, quit1)

	done := make(chan bool)
	absoluteTimeout := func() {
		defer close(done)
		<-time.After(time.Millisecond * 100)
		close(quit1)
	}

	go absoluteTimeout()
	<-done
	<-time.After(time.Millisecond * 400)

	err := waitFunc(pool.StopWait, _timeout)
	if err != nil {
		t.Fail()
	}
}

func TestTimeout(t *testing.T) {
	initialWorkers := 1
	extraWorkers := 10

	jobChannel := make(chan func(), 2)

	pool, _ := WithContext(context.Background(), initialWorkers, jobChannel)

	pool.Expand(extraWorkers, time.Millisecond*10, nil)

	err := waitFunc(pool.StopWait, _timeout)
	if err != nil {
		t.Fail()
	}
}

func TestQuit(t *testing.T) {
	initialWorkers := 1
	extraWorkers := 10

	jobChannel := make(chan func(), 2)
	pool, _ := WithContext(context.Background(), initialWorkers, jobChannel)

	quit1 := make(chan struct{})
	pool.Expand(extraWorkers, 0, quit1)
	close(quit1)

	err := waitFunc(pool.StopWait, _timeout)
	if err != nil {
		t.Fail()
	}
}

func maxDiff(fst, snd, diff int) bool {
	return math.Abs(float64(fst)-float64(snd)) > float64(diff)
}

func TestWorkerPoolQuit(t *testing.T) {
	initialWorkers := 10
	extraWorkers := 10

	jobChannel := make(chan func(), 2)
	pool, _ := WithContext(context.Background(), initialWorkers, jobChannel)

	quit1 := make(chan struct{})
	pool.Expand(extraWorkers, 0, quit1)
	close(quit1)

	err := waitFunc(pool.StopWait, _timeout)
	if err != nil {
		t.Fail()
	}
}

func TestWorkerPoolQuitByClosingJobChannel(t *testing.T) {
	initialWorkers := 10
	extraWorkers := 10

	jobChannel := make(chan func(), 2)
	pool, _ := WithContext(context.Background(), initialWorkers, jobChannel)

	quit1 := make(chan struct{})
	pool.Expand(extraWorkers, 0, quit1)

	// should use pool.StopWait() instead, but this might come in handy too
	close(jobChannel)
	err := waitFunc(pool.StopWait, _timeout)
	if err != nil {
		t.Fail()
	}
}

func TestWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	initialWorkers := 10
	extraWorkers := 10

	jobChannel := make(chan func(), 2)
	pool, _ := WithContext(ctx, initialWorkers, jobChannel)

	pool.Expand(extraWorkers, 100*time.Millisecond, nil)

	cancel()
	err := waitFunc(pool.StopWait, _timeout)
	if err != nil {
		t.Fail()
	}
}

func waitFunc(f func(), exitDelay time.Duration) error {
	funcDone := make(chan struct{})
	go func() {
		defer close(funcDone)
		f()
	}()

	if exitDelay <= 0 {
		<-funcDone

		return nil
	}

	select {
	case <-time.After(exitDelay):
		return ErrTimeout
	case <-funcDone:
	}

	return nil
}

var (
	ErrTimeout = errors.New(`TIMEOUT`)
)

const (
	_timeout = time.Second * 5
)
