// Package queue implements a persistent FIFO queue.
package queue

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/vector"
	"jsouthworth.net/go/seq"
)

var errRangeSig = errors.New("Range requires a function: func(v vT) bool or func(v vT)")

// Queue represents a persistent immutable queue structure.
type Queue struct {
	bv *vector.Slice
}

var empty = Queue{
	bv: vector.Empty().Slice(0, 0),
}

// Empty returns an empty queue.
func Empty() *Queue {
	return &empty
}

// New returns a queue populated with elems.
func New(elems ...interface{}) *Queue {
	q := Empty()
	for _, elem := range elems {
		q = q.Push(elem)
	}
	return q
}

// From returns a queue created from one of several go types:
//
// *Queue:
//    The queue unmodified
// []interface{}:
//    A queue with the elements of the slice passed to New.
// seq.Seqable:
//    A queue populated with the sequence returned by Seq.
// seq.Sequence:
//    A queue populated with the elements of the sequence.
//    Care should be taken to provide finite sequences or the
//    queue will grow without bound.
func From(value interface{}) *Queue {
	if value == nil {
		return Empty()
	}
	switch v := value.(type) {
	case *Queue:
		return v
	case []interface{}:
		return New(v...)
	case seq.Seqable:
		return queueFromSequence(seq.Seq(v))
	case seq.Sequence:
		return queueFromSequence(v)
	default:
		panic(errors.New("cannot convert supplied value to a queue"))
	}
}

func queueFromSequence(coll seq.Sequence) *Queue {
	return seq.Reduce(func(result, input interface{}) interface{} {
		return result.(*Queue).Push(input)
	}, Empty(), coll).(*Queue)
}

// Push returns a Queue with the element added to the end.
func (q *Queue) Push(elem interface{}) *Queue {
	return &Queue{
		bv: q.bv.Append(elem),
	}
}

// Pop returns a queue with the first element removed.
func (q *Queue) Pop() *Queue {
	new := q.bv.Slice(1, q.bv.Length())
	if new.Length() == 0 {
		return Empty()
	}
	return &Queue{
		bv: new,
	}
}

// First returns the first element of the queue.
func (q *Queue) First() interface{} {
	elem, _ := q.bv.Find(0)
	return elem
}

// Range calls the passed in function on each element of the queue.
// The function passed in may be of many types:
//
// func(value interface{}) bool:
//    Takes a value of any type and returns if the loop should continue.
//    Useful to avoid reflection where not needed and to support
//    heterogenous queues.
// func(value interface{})
//    Takes a value of any type.
//    Useful to avoid reflection where not needed and to support
//    heterogenous queues.
// func(value T) bool:
//    Takes a value of the type of element stored in the queue and
//    returns if the loop should continue. Useful for homogeneous queues.
//    Is called with reflection and will panic if the type is incorrect.
// func(value T)
//    Takes a value of the type of element stored in the queue and
//    returns if the loop should continue. Useful for homogeneous queues.
//    Is called with reflection and will panic if the type is incorrect.
// Range will panic if passed anything that doesn't match one of these signatures
func (q *Queue) Range(do interface{}) {
	cont := true
	fn := genRangeFunc(do)
	for queue := q; queue != Empty() && cont; queue = queue.Pop() {
		value := queue.First()
		cont = fn(value)
	}
}

func genRangeFunc(do interface{}) func(value interface{}) bool {
	switch fn := do.(type) {
	case func(value interface{}) bool:
		return fn
	case func(value interface{}):
		return func(value interface{}) bool {
			fn(value)
			return true
		}
	default:
		rv := reflect.ValueOf(do)
		if rv.Kind() != reflect.Func {
			panic(errRangeSig)
		}
		rt := rv.Type()
		if rt.NumIn() != 1 || rt.NumOut() > 1 {
			panic(errRangeSig)
		}
		if rt.NumOut() == 1 &&
			rt.Out(0).Kind() != reflect.Bool {
			panic(errRangeSig)
		}
		return func(value interface{}) bool {
			out := dyn.Apply(do, value)
			if out != nil {
				return out.(bool)
			}
			return true
		}
	}
}

// Seq returns the queue as a sequence.
func (q *Queue) Seq() seq.Sequence {
	if q.bv.Length() == 0 {
		return nil
	}
	return &queueSeq{
		queue: q,
	}
}

// String returns a representation of the queue as a string.
func (q *Queue) String() string {
	b := new(strings.Builder)
	fmt.Fprint(b, "[ ")
	q.Range(func(item interface{}) {
		fmt.Fprintf(b, "%v ", item)
	})
	fmt.Fprint(b, "]")
	return b.String()
}

// Length returns the number of elements currently in the queue.
func (q *Queue) Length() int {
	return q.bv.Length()
}

type queueSeq struct {
	queue *Queue
}

func (q *queueSeq) First() interface{} {
	return q.queue.First()
}

func (q *queueSeq) Next() seq.Sequence {
	new := q.queue.Pop()
	if new.bv.Length() == 0 {
		return nil
	}
	return &queueSeq{
		queue: new,
	}
}

func (q *queueSeq) String() string {
	return seq.ConvertToString(q)
}
