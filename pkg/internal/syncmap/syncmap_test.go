package syncmap

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func blammo(t *testing.T, i int, sm *SyncMap[string, string]) {
	k := string(alphabet[i])
	v := k

	sm.Put(k, v)

	keys := sm.Keys()
	assert.Contains(t, keys, k, k)

	val, ok := sm.Get(k)
	assert.True(t, ok, k)
	assert.Equal(t, v, val, v)

	sm.Remove(k)
	keys = sm.Keys()
	assert.NotContains(t, keys, k, k)
}

func Test_SyncMap(t *testing.T) {
	sm := New[string, string]()

	wg := sync.WaitGroup{}
	wg.Add(len(alphabet))
	for i := range alphabet {
		go func(i int) {
			defer wg.Done()
			blammo(t, i, sm)
		}(i)
	}

	wg.Wait()
}
