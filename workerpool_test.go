package workerpool

import (
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
