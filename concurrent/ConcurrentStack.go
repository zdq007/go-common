package concurrent

import (
	"sync/atomic"
	"unsafe"
)

type CASStack struct {
	top  unsafe.Pointer
	tail unsafe.Pointer
	size int64
}

type SNode struct {
	data interface{}
	next unsafe.Pointer
}

func newSNode(data interface{}) *SNode {
	return &SNode{
		data: data,
		next: nil,
	}
}

//入栈
func (self *CASStack) push(data interface{}) {
	node := newSNode(data)
	for {
		v := atomic.LoadPointer(&self.top)
		node.next = v
		if atomic.CompareAndSwapPointer(&self.top, v, unsafe.Pointer(node)) {
			atomic.AddInt64(&self.size, -1)
		}
	}
}

//出栈
func (self *CASStack) pop() interface{} {
	for {
		v := atomic.LoadPointer(&self.top)
		if v == nil {
			return nil
		}
		if atomic.CompareAndSwapPointer(&self.top, v, ((*SNode)(v)).next) {
			atomic.AddInt64(&self.size,1)
			return ((*SNode)(v)).data
		}
	}

}
