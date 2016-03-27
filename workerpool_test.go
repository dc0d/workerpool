package workerpool

import (
	"math"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestNegWorkers(t *testing.T) {
	jobChannel := make(chan Job)
	pool := New(-1, jobChannel)
	pool.Run()

	n := int64(runtime.NumCPU())
	var backSlot int64
	var job Job = func() {
		atomic.AddInt64(&backSlot, 1)
		select {}
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
}

func TestZeroWorkers(t *testing.T) {
	jobChannel := make(chan Job)
	pool := New(0, jobChannel)
	pool.Run()

	var backSlot int64 = 10
	var job Job = func() {
		atomic.StoreInt64(&backSlot, 110)
	}
	select {
	case jobChannel <- job:
	case <-time.After(time.Millisecond * 300):
	}
	if atomic.LoadInt64(&backSlot) != 10 {
		t.Fail()
	}

	pool.GrowExtra(1, NewWorkerConfig(0, make(chan bool)))

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
}

func TestAbsoluteTimeout(t *testing.T) {
	dispatcherGoroutine := 1
	initialWorkers := 1
	extraWorkers := 10
	startedWith := runtime.NumGoroutine()

	jobChannel := make(chan Job, 2)

	pool := New(initialWorkers, jobChannel)
	pool.Run()

	var conf WorkerConfig
	conf.Quit = make(chan bool)
	pool.GrowExtra(extraWorkers, conf)

	afterGoroutines := runtime.NumGoroutine()
	thenGoroutines := startedWith + extraWorkers + initialWorkers + dispatcherGoroutine
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}

	done := make(chan bool)
	absoluteTimeout := func() {
		defer close(done)
		<-time.After(time.Millisecond * 100)
		close(conf.Quit)
	}

	go absoluteTimeout()
	<-done
	<-time.After(time.Millisecond * 400)
	runtime.GC()

	afterGoroutines = runtime.NumGoroutine()
	thenGoroutines = startedWith + initialWorkers + dispatcherGoroutine // no extraWorkers
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}
}

func TestTimeout(t *testing.T) {
	dispatcherGoroutine := 1
	initialWorkers := 1
	extraWorkers := 10
	startedWith := runtime.NumGoroutine()

	jobChannel := make(chan Job, 2)

	pool := New(initialWorkers, jobChannel)
	pool.Run()

	var conf WorkerConfig
	conf.Timeout = time.Millisecond * 10
	pool.GrowExtra(extraWorkers, conf)

	<-time.After(time.Millisecond * 100)

	afterGoroutines := runtime.NumGoroutine()
	thenGoroutines := startedWith + initialWorkers + dispatcherGoroutine // no extraWorkers
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}
}

func TestQuit(t *testing.T) {
	dispatcherGoroutine := 1
	initialWorkers := 1
	extraWorkers := 10
	startedWith := runtime.NumGoroutine()

	jobChannel := make(chan Job, 2)

	pool := New(initialWorkers, jobChannel)
	pool.Run()

	var conf WorkerConfig
	conf.Quit = make(chan bool)
	pool.GrowExtra(extraWorkers, conf)

	afterGoroutines := runtime.NumGoroutine()
	thenGoroutines := startedWith + extraWorkers + initialWorkers + dispatcherGoroutine
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}

	close(conf.Quit)
	<-time.After(time.Millisecond * 100)

	afterGoroutines = runtime.NumGoroutine()
	thenGoroutines = startedWith + initialWorkers + dispatcherGoroutine // no extraWorkers
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}
}

func maxDiff(fst, snd, diff int) bool {
	return math.Abs(float64(fst)-float64(snd)) > float64(diff)
}
