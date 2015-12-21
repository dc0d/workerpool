package workerpool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	jobChannel := make(chan Job, 2)

	pool := New(1, jobChannel)
	pool.Run()

	<-time.After(time.Millisecond * 500)

	var conf WorkerConfig
	conf.Timeout = time.Millisecond * 100
	pool.GrowExtra(1, conf)

	<-time.After(time.Millisecond * 500)

	taskList := make(chan int, 2)
	taskList <- 1
	taskList <- 1

	startedAt := time.Now()

	jobChannel <- func() {
		<-time.After(time.Second * 2)

		<-taskList
	}
	jobChannel <- func() {
		<-time.After(time.Second * 2)

		<-taskList
	}

	taskList <- 1
	taskList <- 1

	elapsed := time.Since(startedAt)

	if elapsed < time.Millisecond*4000 {
		t.Log(elapsed)
		t.Fail()
	}
}

func TestQuit(t *testing.T) {
	jobChannel := make(chan Job, 2)

	pool := New(1, jobChannel)
	pool.Run()

	<-time.After(time.Millisecond * 500)

	var conf WorkerConfig
	conf.Quit = make(chan bool)
	pool.GrowExtra(10, conf)

	<-time.After(time.Millisecond * 500)

	close(conf.Quit)

	<-time.After(time.Millisecond * 500)

	taskList := make(chan int, 2)
	taskList <- 1
	taskList <- 1

	startedAt := time.Now()

	jobChannel <- func() {
		<-time.After(time.Second * 2)

		<-taskList
	}
	jobChannel <- func() {
		<-time.After(time.Second * 2)

		<-taskList
	}

	taskList <- 1
	taskList <- 1

	elapsed := time.Since(startedAt)

	if elapsed < time.Millisecond*4000 {
		t.Log(elapsed)
		t.Fail()
	}
}

func TestSmokeTest(t *testing.T) {
	jobChannel := make(chan Job, 50)

	pool := New(0, jobChannel)
	pool.Run()

	pool.GrowExtra(512, WorkerConfig{Timeout: time.Millisecond * 1000})

	var count int64
	const N = 10000

	wg := new(sync.WaitGroup)
	for i := 0; i < N; i++ {
		jobChannel <- func() {
			wg.Add(1)
			<-time.After(time.Millisecond * 50)
			atomic.AddInt64(&count, 1)
			wg.Done()
		}
	}

	<-time.After(time.Millisecond * 500)
	wg.Wait()

	if count != N {
		t.Fail()
	}
}
