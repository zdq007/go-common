package util

import (
	"sync"
)

type SafeMap struct {
	lock *sync.RWMutex
	bm   map[interface{}]interface{}
}

func NewSafeMap(caps int) *SafeMap {
	return &SafeMap{
		lock: new(sync.RWMutex),
		bm:   make(map[interface{}]interface{}, caps),
	}
}

//Get from maps return the k's value
func (m *SafeMap) Get(k interface{}) interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if val, ok := m.bm[k]; ok {
		return val
	}
	return nil
}

//Get from maps return the k's value
func (m *SafeMap) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.bm)
}

// Maps the given key and value. Returns false
// if the key is already in the map and changes nothing.
func (m *SafeMap) Set(k interface{}, v interface{}) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if val, ok := m.bm[k]; !ok {
		m.bm[k] = v
	} else if val != v {
		m.bm[k] = v
	} else {
		return false
	}
	return true
}
func (m *SafeMap) SetSet(k1 interface{}, k2 interface{}, v interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.bm[k1]; !ok {
		tm := NewSafeMap(10)
		tm.bm[k2] = v
		m.bm[k1] = tm
	} else {
		m.bm[k1].(*SafeMap).bm[k2] = v
	}
}

func (m *SafeMap) Foreach(backfn func(k interface{}, v interface{})) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for key, val := range m.bm {
		backfn(key, val)
	}
}

// Returns true if k is exist in the map.
func (m *SafeMap) Check(k interface{}) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if _, ok := m.bm[k]; !ok {
		return false
	}
	return true
}

func (m *SafeMap) Delete(k interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.bm, k)
}

func (m *SafeMap) DelDel(k1 interface{}, k2 interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if v := m.bm[k1]; v!=nil {
		childmap := v.(*SafeMap)
		delete(childmap.bm, k2)
		if len(childmap.bm) == 0 {
			delete(m.bm, k1)
		}
	}
}
