// Package queue implements a persistent FIFO queue.
package queue

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/stack"
	"jsouthworth.net/go/immutable/vector"
	"jsouthworth.net/go/seq"
)

var errRangeSig = errors.New("Range requires a function: func(v vT) bool or func(v vT)")

// Queue represents a persistent immutable queue structure.
type Queue struct {
	bv    *vector.Slice
	stack *stack.Stack
}

var empty = Queue{
	bv:    vector.Empty().Slice(0, 0),
	stack: stack.Empty(),
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
// []int:
//    A queue with the elements of the slice is created.
// seq.Seqable:
//    A queue populated with the sequence returned by Seq.
// seq.Sequence:
//    A queue populated with the elements of the sequence.
//    Care should be taken to provide finite sequences or the
//    queue will grow without bound.
// Other:
//    Returns Empty()
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
		return queueFromReflection(value)
	}
}

func queueFromReflection(value interface{}) *Queue {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Slice:
		out := Empty()
		for i := 0; i < v.Len(); i++ {
			out = out.Push(v.Index(i).Interface())
		}
		return out
	default:
		return Empty()
	}
}

func queueFromSequence(coll seq.Sequence) *Queue {
	return seq.Reduce(func(result, input interface{}) interface{} {
		return result.(*Queue).Push(input)
	}, Empty(), coll).(*Queue)
}

// Push returns a Queue with the element added to the end.
func (q *Queue) Push(elem interface{}) *Queue {
	if q.Length() == 0 {
		return &Queue{
			bv:    q.bv.Append(elem),
			stack: q.stack,
		}
	}
	return &Queue{
		bv:    q.bv,
		stack: q.stack.Push(elem),
	}
}

// Conj returns a Queue with the element added to the end.
// Conj implements a generic mechanism for building collections.
func (q *Queue) Conj(elem interface{}) interface{} {
	return q.Push(elem)
}

// Pop returns a queue with the first element removed.
func (q *Queue) Pop() *Queue {
	new := q.bv.Slice(1, q.bv.Length())
	if new.Length() != 0 {
		return &Queue{
			bv:    new,
			stack: q.stack,
		}
	}
	if q.stack.Length() == 0 {
		return Empty()
	}
	return &Queue{
		bv:    q.stack.Reverse().Slice(0, q.stack.Length()),
		stack: stack.Empty(),
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

// Reduce is a fast mechanism for reducing a Queue. Reduce can take
// the following types as the fn:
//
// func(init interface{}, value interface{}) interface{}
// func(init iT, v vT) oT
//
// Reduce will panic if given any other function type.
func (q *Queue) Reduce(fn interface{}, init interface{}) interface{} {
	return q.stack.Reverse().Reduce(fn, q.bv.Reduce(fn, init))
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
	return q.bv.Length() + q.stack.Length()
}

// Equal returns whether the other value passed in is a queue and the
// values of that queue are equal to its values.
func (q *Queue) Equal(other interface{}) bool {
	oq, isQueue := other.(*Queue)
	return isQueue &&
		q.bv.Equal(oq.bv)
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
