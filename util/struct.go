package util

import "sync"

type SafeMap struct {
	*sync.RWMutex
	unsafeMap map[int]string
}

func NewSafeMap() *SafeMap {
	return &SafeMap{RWMutex: &sync.RWMutex{}, unsafeMap: map[int]string{}}
}

func (m *SafeMap) AddReverseMap(mymap map[string]int) {
	m.Lock()
	defer m.Unlock()

	for cid, index := range mymap {
		m.unsafeMap[index] = cid
	}
}

func (m *SafeMap) Add(key int, val string) {
	m.Lock()
	defer m.Unlock()

	m.unsafeMap[key] = val
}

func (m *SafeMap) Get(key int) (val string, ok bool) {
	m.RLock()
	defer m.RUnlock()

	val, ok = m.unsafeMap[key]
	return
}

func (m *SafeMap) GetAll() map[int]string {
	m.RLock()
	defer m.RUnlock()

	mymap := make(map[int]string)
	for key, value := range m.unsafeMap {
		mymap[key] = value
	}
	return mymap
}
