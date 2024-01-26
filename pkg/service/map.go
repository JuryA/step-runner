package service

import (
	"sync"

	"golang.org/x/exp/maps"
)

type ConcurrentMap[K comparable, V any] struct {
	data map[K]V
	lock sync.RWMutex
}

func New[K comparable, V any]() *ConcurrentMap[K, V] {
	return &ConcurrentMap[K, V]{
		data: make(map[K]V),
	}
}

func (c *ConcurrentMap[K, V]) Put(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.data[key] = value
}

func (c *ConcurrentMap[K, V]) Get(key K) (V, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *ConcurrentMap[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.data, key)
}

func (c *ConcurrentMap[K, V]) Keys() []K {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return maps.Keys(c.data)
}
