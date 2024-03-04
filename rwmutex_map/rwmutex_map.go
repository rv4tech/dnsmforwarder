package rwmutex_map

import (
	"sync"
)

/*
Simple RwMutex concurrent Map

On high-concurrent systems better
use https://github.com/mhmtszr/concurrent-swiss-map
or https://github.com/orcaman/concurrent-map
*/

type Map[K comparable, V any] struct {
	data map[K]V
	// we need global lock for delete-by-value
	// so sync.Map is not suitable
	lock sync.RWMutex
}

func New[K comparable, V any]() *Map[K, V] {
	m := Map[K, V]{}
	m.data = make(map[K]V)
	return &m
}

func (m *Map[K, V]) Store(key K, value V) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.data[key] = value
}

func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	value, ok = m.data[key]
	return
}

func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	value, loaded = m.data[key]
	delete(m.data, key)
	return
}

func (m *Map[K, V]) Delete(key K) {
	m.LoadAndDelete(key)
}

func (m *Map[K, V]) DeleteFunc(del func(K, V) bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for k, v := range m.data {
		if del(k, v) {
			delete(m.data, k)
		}
	}
}

func (m *Map[K, V]) DeleteFuncMap(del func(K, V) bool) map[K]V {
	m.lock.Lock()
	defer m.lock.Unlock()

	d := make(map[K]V)
	for k, v := range m.data {
		if del(k, v) {
			delete(m.data, k)
			d[k] = v
		}
	}
	return d
}

func (m *Map[K, V]) Clone() map[K]V {
	m.lock.Lock()
	defer m.lock.Unlock()

	d := make(map[K]V, len(m.data))
	for k, v := range m.data {
		d[k] = v
	}
	return d
}
