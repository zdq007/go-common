package concurrent

import (
	"sync/atomic"
	"unsafe"
)

// private structure
type node struct {
	value interface{}
	next  *node
}

type CASQueue struct {
	dummy *node
	tail  *node
}

func newCASQueue() *CASQueue {
	q := new(CASQueue)
	q.dummy = new(node)
	q.tail = q.dummy

	return q
}

func (q *CASQueue) enqueue(v interface{}) {
	var oldTail, oldTailNext *node

	newNode := new(node)
	newNode.value = v

	newNodeAdded := false

	for !newNodeAdded {
		oldTail = q.tail
		oldTailNext = oldTail.next

		if q.tail != oldTail {
			continue
		}
		//可能有多个线程在同时写，且已经有一个写入进去了
		if oldTailNext != nil {
			atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(oldTail), unsafe.Pointer(oldTailNext))
			continue
		}

		newNodeAdded = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&oldTail.next)), unsafe.Pointer(oldTailNext), unsafe.Pointer(newNode))
	}

	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(oldTail), unsafe.Pointer(newNode))
}

func (q *CASQueue) dequeue() (interface{}, bool) {
	var temp interface{}
	var oldDummy, oldHead *node

	removed := false

	for !removed {
		oldDummy = q.dummy
		oldHead = oldDummy.next
		oldTail := q.tail

		if q.dummy != oldDummy {
			continue
		}
		//如果取出head为空则直接返回
		if oldHead == nil {
			return nil, false
		}
		//Dummy和尾在一起 且写入方已经放入一个元素，帮写入方设置tail 后继续尝试读取
		if oldTail == oldDummy {
			atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.tail)), unsafe.Pointer(oldTail), unsafe.Pointer(oldHead))
			continue
		}
		//Dummy和尾不在一起，就剩下考虑多个线程同时读取的问题了，先取出节点值，然后再看有没有被其他线程读取走，如果读取了则继续尝试
		temp = oldHead.value
		removed = atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.dummy)), unsafe.Pointer(oldDummy), unsafe.Pointer(oldHead))
	}

	return temp, true
}


