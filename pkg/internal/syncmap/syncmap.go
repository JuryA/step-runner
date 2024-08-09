// syncmap implements a simple generic, synchronized map.
package syncmap

import (
	"sync"

	"golang.org/x/exp/maps"
)

type SyncMap[K comparable, V any] struct {
	data map[K]V
	lock sync.RWMutex
}

func New[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		data: make(map[K]V),
	}
}

func (c *SyncMap[K, V]) Put(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.data[key] = value
}

func (c *SyncMap[K, V]) Get(key K) (V, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *SyncMap[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.data, key)
}

func (c *SyncMap[K, V]) Keys() []K {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return maps.Keys(c.data)
}

func (c *SyncMap[K, V]) ForEach(f func(k K, v V)) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for k, v := range c.data {
		f(k, v)
	}
}
