package value

import (
	"fmt"
	"iter"
	"maps"
	"slices"
)

// Mapper is an interface used to represent objects types.
type Mapper interface {
	Attr(string) (Value, error)
	All() iter.Seq2[Value, Value]
	Values() iter.Seq[Value]
	Keys() iter.Seq[Value]
	Len() int
	WithMarks(marks uint16) Mapper
}

// Map implements Mapper and is the default for the object type.
type Map struct {
	m    map[string]Value
	keys []Value
}

// MustMap creates a new ordered map by passing (key, value, key, value)
//
// This function panics if there's an odd number of elements passed, or
// if a key value is not a String.
func MustMap(kvs ...Value) Value {
	m, err := NewMap(kvs...)
	if err != nil {
		panic(err)
	}

	return m
}

// NewMap creates a new ordered map by passing (key, value, key, value)
func NewMap(kvs ...Value) (Value, error) {
	if len(kvs)%2 != 0 {
		return null, fmt.Errorf("mismatching keys and value element count")
	}

	keys := make([]Value, len(kvs)/2)
	vals := make(map[string]Value, len(keys))

	for i := 0; i < len(kvs); i += 2 {
		key := kvs[i]
		if key.kind != StringKind {
			return null, fmt.Errorf("key was not of kind String")
		}

		keys[i/2] = kvs[i]
		vals[key.v.(string)] = kvs[i+1]
	}

	return Object(&Map{m: vals, keys: keys}), nil
}

func (m *Map) Attr(key string) (Value, error) {
	val, ok := m.m[key]
	if !ok {
		return null, ErrAttributeNotFound
	}

	return val, nil
}

func (m *Map) All() iter.Seq2[Value, Value] {
	return func(yield func(Value, Value) bool) {
		for _, key := range m.keys {
			if !yield(key, m.m[key.v.(string)]) {
				return
			}
		}
	}
}

func (m *Map) Values() iter.Seq[Value] {
	return maps.Values(m.m)
}

func (m *Map) Keys() iter.Seq[Value] {
	return slices.Values(m.keys)
}

func (m *Map) Len() int {
	return len(m.keys)
}

func (m *Map) WithMarks(marks uint16) Mapper {
	vals := make(map[string]Value, len(m.keys))
	keys := make([]Value, len(m.keys))

	for idx, key := range m.keys {
		keys[idx] = key.WithMarks(marks)
		vals[key.v.(string)] = m.m[key.v.(string)].WithMarks(marks)
	}

	return &Map{m: vals, keys: keys}
}
