// package memory implements a simple way to facilitate "live" steaming of data originating from a single source, to
// multiple clients. The Write() method is used by the producer side to capture and accumulate (i.e. buffer) new data.
// The Follow() method is used by the consumer side to subscribe to and receive data. Calling Follow() will write all
// data buffered so far, as well as all future data the producer side writes. Follow() will block until Stop() is
// called, which tells the consumer side to stop waiting for new data and return. Data is buffered in-memory. This type
// should only be used when the data being buffered is bounded and small (enough).

package memory

import (
	"sync"
)

type Streamer[T any] struct {
	cond *sync.Cond
	data []T
	stop bool
}

func New[T any]() *Streamer[T] {
	return &Streamer[T]{cond: sync.NewCond(&sync.Mutex{})}
}

func (s *Streamer[T]) Write(v T) {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	s.data = append(s.data, v)
	s.cond.Broadcast()
}

func (s *Streamer[T]) Stop() {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	s.stop = true
	s.cond.Broadcast()
}

func (s *Streamer[T]) Follow(offset int32, write func(T) error) error {
	i := int(offset)
	for {
		s.cond.L.Lock()
		for ; i < len(s.data); i++ {
			if err := write(s.data[i]); err != nil {
				s.cond.L.Unlock()
				return err
			}
		}
		if s.stop {
			s.cond.L.Unlock()
			return nil
		}
		s.cond.Wait()
		s.cond.L.Unlock()
	}
}
