package test

import (
	"sync/atomic"
	"time"
)

type Test struct {
	count int64
}
func NewTest() *Test{
	return &Test{
		count:0,
	}
}
func (self *Test) timer(ms int, fun func(count int64)) {
	timer1 := time.NewTicker(time.Duration(ms) * time.Millisecond)
	for {
		select {
		case <-timer1.C:
			fun(self.count)
			self.resetCount()
		}
	}
}

func (self *Test) Add() {
	atomic.AddInt64(&self.count, 1)
}
func (self *Test) resetCount() {
	atomic.StoreInt64(&self.count, 0)
}

func (self *Test) Counter(ms int, fun func(count int64)) {
	go func() {
		self.timer(ms, fun)
	}()
}
