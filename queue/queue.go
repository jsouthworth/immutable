package queue

import (
	"errors"

	"jsouthworth.net/go/immutable/vector"
	"jsouthworth.net/go/seq"
)

// Queue represents a persistent immutable queue structure.
type Queue struct {
	first seq.Sequence
	rest  *vector.Vector
	count int
}

var empty = Queue{
	first: nil,
	rest:  nil,
	count: 0,
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
	if q.first == nil {
		return &Queue{
			first: seq.Cons(elem, nil),
			rest:  nil,
			count: 1,
		}
	}
	return &Queue{
		first: q.first,
		rest:  q.rest.Append(elem),
		count: q.count + 1,
	}
}

// Pop returns a queue with the first element removed.
func (q *Queue) Pop() *Queue {
	if q.first == nil {
		return q
	}

	first := seq.Next(q.first)
	if first != nil {
		return &Queue{
			first: first,
			rest:  q.rest,
			count: q.count - 1,
		}
	}

	return &Queue{
		first: seq.Seq(q.rest),
		rest:  nil,
		count: q.count - 1,
	}
}

// First returns the first element of the queue.
func (q *Queue) First() interface{} {
	return seq.First(q.first)
}

// Seq returns the queue as a sequence.
func (q *Queue) Seq() seq.Sequence {
	if q.first == nil {
		return nil
	}
	return &queueSeq{
		first: q.first,
		rest:  seq.Seq(q.rest),
	}
}

// Length returns the number of elements currently in the queue.
func (q *Queue) Length() int {
	return q.count
}

type queueSeq struct {
	first seq.Sequence
	rest  seq.Sequence
}

func (q *queueSeq) First() interface{} {
	return seq.First(q.first)
}

func (q *queueSeq) Next() seq.Sequence {
	first := seq.Next(q.first)
	if first == nil {
		if q.rest == nil {
			return nil
		}
		return &queueSeq{
			first: q.rest,
			rest:  nil,
		}
	}
	return &queueSeq{
		first: first,
		rest:  q.rest,
	}
}
