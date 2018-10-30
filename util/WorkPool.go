package util

import (
	"fmt"
	"sync"
)

type WorkPool struct {
	allowLevel     bool
	jobpipe_level1 chan *Job //三个优先级
	jobpipe_level2 chan *Job
	jobpipe_level3 chan *Job
	workers        []*Worker
	locker         *sync.Mutex
	//checkrun       bool  //检测协程是否工作
	initSize int32 //初始化指定的工人大小
	delSize  int32 //异常销毁的个人大小
}
type Worker struct {
	code   int32     //编号
	status byte      //工作状态 0待命  1工作中 2 deal
	delcmd chan bool //销毁命令
}
type Job struct {
	fun  func(...interface{})
	args []interface{}
}

func (self *Job) proc() {
	self.fun(self.args...)
}
func NewJob(fun func(...interface{}), args ...interface{}) *Job {
	return &Job{
		fun:  fun,
		args: args,
	}
}
func (self *WorkPool) Toplen() int {
	return len(self.jobpipe_level1)
}
func (self *WorkPool) AddTopJob(fun func(...interface{}), args ...interface{}) {
	job := NewJob(fun, args...)
	self.jobpipe_level1 <- job
}
func (self *WorkPool) AddCenterJob(fun func(...interface{}), args ...interface{}) {
	job := NewJob(fun, args...)
	if self.allowLevel {
		self.jobpipe_level2 <- job
	} else {
		self.jobpipe_level1 <- job
	}
}
func (self *WorkPool) AddFootJob(fun func(...interface{}), args ...interface{}) {
	job := NewJob(fun, args...)
	if self.allowLevel {
		self.jobpipe_level3 <- job
	} else {
		self.jobpipe_level1 <- job
	}
}

/**
 *添加JOB
 **/
func (self *WorkPool) AddJob(level byte, fun func(...interface{}), args ...interface{}) {
	job := NewJob(fun, args...)
	if self.allowLevel {
		switch level {
		case 1:
			self.jobpipe_level1 <- job
		case 2:
			self.jobpipe_level2 <- job
		default:
			self.jobpipe_level3 <- job
		}
	} else {
		self.jobpipe_level1 <- job
	}
}
func NewWorker(code int32) *Worker {
	return &Worker{code: code, delcmd: make(chan bool)}
}

//只有一个级别的pool
func NewWorkPool(queuenum, worknum int32) *WorkPool {
	return newWorkPool(false, queuenum, worknum)
}

//支持优先级的pool
func NewLevelWorkPool(queuenum, worknum int32) *WorkPool {
	return newWorkPool(true, queuenum, worknum)
}

func newWorkPool(allowlevel bool, queuenum, worknum int32) (pool *WorkPool) {
	if allowlevel {
		pool = &WorkPool{
			jobpipe_level1: make(chan *Job, queuenum),
			jobpipe_level2: make(chan *Job, queuenum),
			jobpipe_level3: make(chan *Job, queuenum),
			workers:        make([]*Worker, 0, worknum),
			locker:         new(sync.Mutex),
			allowLevel:     allowlevel,
			initSize:       worknum,
		}
	} else {
		pool = &WorkPool{
			jobpipe_level1: make(chan *Job, queuenum),
			workers:        make([]*Worker, 0, worknum),
			locker:         new(sync.Mutex),
			allowLevel:     allowlevel,
			initSize:       worknum,
		}
	}
	for i := int32(0); i < worknum; i++ {
		pool.workers = append(pool.workers, NewWorker(i))
	}
	return
}

func (self *WorkPool) Run() {
	self.locker.Lock()
	defer self.locker.Unlock()
	for _, worker := range self.workers {
		go worker.work(self)
	}
	//self.checkrun = true
	//go self.checker()
}

func (self *WorkPool) GetDelSize() int32 {
	return self.delSize
}
func (self *WorkPool) Stop() {
	self.locker.Lock()
	defer self.locker.Unlock()
	//self.checkrun = false
	for _, worker := range self.workers {
		worker.delcmd <- true
	}
	close(self.jobpipe_level1)
	if self.allowLevel {
		close(self.jobpipe_level2)
		close(self.jobpipe_level3)
	}
}

/*
func (self *WorkPool) checker() {
	for self.checkrun {
		for i, worker := range self.workers {
			if worker.status == 2 {
				self.delSize++
				worker := NewWorker(int32(i))
				self.workers[i] = worker
				go worker.work(self)
			}
		}
	}
}
*/
func (self *Worker) work(pool *WorkPool) {
	self.status = 1
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("工人异常结束！")
			self.status = 2
			//重新恢复工人
			worker := NewWorker(self.code)
			pool.workers[self.code] = worker
			go worker.work(pool)
		}
	}()
	if pool.allowLevel {
		for {
			select {
			case job := <-pool.jobpipe_level1:
				job.proc()
			case job := <-pool.jobpipe_level2:
				job.proc()
			case job := <-pool.jobpipe_level3:
				job.proc()
			case <-self.delcmd:
				break
			}
		}
	} else {
		for {
			select {
			case job := <-pool.jobpipe_level1:
				job.proc()
			case <-self.delcmd:
				break
			}
		}
	}
	self.status = 2
}
