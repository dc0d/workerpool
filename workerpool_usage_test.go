package workerpool_test

import (
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dc0d/workerpool/v5"
	"github.com/stretchr/testify/assert"
)

func TestNegWorkers(t *testing.T) {
	pool := workerpool.New(-1, -1)

	quit := make(chan struct{})

	n := int64(runtime.NumCPU())
	var backSlot int64
	var job = func() {
		atomic.AddInt64(&backSlot, 1)
		<-quit
	}
	for pool.Queue(job, time.Millisecond*50) {
	}

	actual := atomic.LoadInt64(&backSlot)
	assert.Equal(t, n, actual)

	close(quit)

	assert.NoError(t, waitFunc(pool.Stop, _timeout))
}

func TestZeroWorkers(t *testing.T) {
	pool := workerpool.New(0, 0)

	var backSlot int64 = 10
	var job = func() {
		atomic.StoreInt64(&backSlot, 110)
	}
	pool.Queue(job, 0)

	assert.Equal(t, int64(10), atomic.LoadInt64(&backSlot))

	pool.Expand(1, 0, nil)

	done := make(chan bool)
	job = func() {
		defer close(done)
		atomic.StoreInt64(&backSlot, 73)
	}
	assert.True(t, pool.Queue(job, time.Millisecond*50))
	<-done

	assert.Equal(t, int64(73), atomic.LoadInt64(&backSlot))

	assert.NoError(t, waitFunc(pool.Stop, _timeout))
}

func TestAbsoluteTimeout(t *testing.T) {
	initialWorkers := 1
	extraWorkers := 10

	pool := workerpool.New(initialWorkers, 2)

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

	assert.NoError(t, waitFunc(pool.Stop, _timeout))
}

func TestTimeout(t *testing.T) {
	initialWorkers := 1
	extraWorkers := 10

	pool := workerpool.New(initialWorkers, 2)

	pool.Expand(extraWorkers, time.Millisecond*10, nil)

	assert.NoError(t, waitFunc(pool.Stop, _timeout))
}

func TestQuit(t *testing.T) {
	initialWorkers := 1
	extraWorkers := 10

	pool := workerpool.New(initialWorkers, 2)

	quit1 := make(chan struct{})
	pool.Expand(extraWorkers, 0, quit1)
	close(quit1)

	assert.NoError(t, waitFunc(pool.Stop, _timeout))
}

func TestWorkerPoolQuit(t *testing.T) {
	initialWorkers := 10
	extraWorkers := 10

	pool := workerpool.New(initialWorkers, 2)

	pool.Expand(extraWorkers, 0, nil)

	assert.NoError(t, waitFunc(pool.Stop, _timeout))
}

func TestTimeoutNoGoroutineLeak(t *testing.T) {
	initialWorkers := 10
	extraWorkers := 1000

	pool := workerpool.New(initialWorkers, 2)

	before := runtime.NumGoroutine()

	pool.Expand(extraWorkers, time.Millisecond*100, nil)

	go func() { // A
		for i := 0; i < extraWorkers; i++ {
			assert.True(t, pool.Queue(func() {
				time.Sleep(time.Millisecond * 10)
			}, 0))
		}
	}()

	<-time.After(time.Millisecond * 500)
	go func() {
		for i := 0; i < initialWorkers*2; i++ {
			assert.True(t, pool.Queue(func() {
				time.Sleep(time.Millisecond * 10)
			}, 0))
		}
	}()

	<-time.After(time.Millisecond * 500)
	after := runtime.NumGoroutine()
	if (after - before) > (initialWorkers * 2) {
		t.Fatal()
	}

	assert.NoError(t, waitFunc(pool.Stop, _timeout))
}

func ExampleWorkerPool() {
	pool := workerpool.New(-1, 0)

	var v int64
	go func() {
		for i := 0; i < 100; i++ {
			pool.Queue(func() {
				atomic.AddInt64(&v, 1)
			}, 0)
		}
	}()

	pool.Stop()

	if v != 100 {
		panic("BOOM!")
	}
}

func ExampleWorkerPool_Expand() {
	pool := workerpool.New(-1, 0)
	pool.Expand(1000, time.Millisecond, nil)

	var v int64
	go func() {
		for i := 0; i < 100; i++ {
			pool.Queue(func() {
				atomic.AddInt64(&v, 1)
			}, 0)
		}
	}()

	pool.Stop()

	if v != 100 {
		panic("BOOM!")
	}
}

//nolint
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
