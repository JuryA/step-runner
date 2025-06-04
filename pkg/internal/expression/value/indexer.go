package value

import (
	"iter"
	"slices"
)

// Indexer is an interface used to represent array types.
type Indexer interface {
	Index(int) (Value, error)
	All() iter.Seq2[int, Value]
	Values() iter.Seq[Value]
	Len() int
	WithMarks(marks uint16) Indexer
}

// NewList returns a new array value backed by List.
func NewList(v ...Value) Value {
	return Array(List(v))
}

// List implements Indexer and is the default for the array type.
type List []Value

func (l List) Index(i int) (Value, error) {
	if i < 0 || i >= len(l) {
		return null, ErrArrayIndexOutOfRange
	}

	return l[int(i)], nil
}

func (l List) All() iter.Seq2[int, Value] {
	return slices.All(l)
}

func (l List) Values() iter.Seq[Value] {
	return slices.Values(l)
}

func (l List) Len() int {
	return len(l)
}

func (l List) WithMarks(marks uint16) Indexer {
	arr := make(List, 0, len(l))
	for _, val := range l {
		arr = append(arr, val.WithMarks(marks))
	}

	return arr
}
