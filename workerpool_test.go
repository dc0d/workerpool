package workerpool

import (
	"math"
	"runtime"
	"testing"
	"time"
)

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
