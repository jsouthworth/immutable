// Package list implements a persistent linked list.
package list // import "jsouthworth.net/go/immutable/list"

import (
	"errors"
	"reflect"

	"jsouthworth.net/go/seq"
)

var errRangeSig = errors.New("Range requires a function: func(v vT) bool or func(v vT)")

// List is a persistent linked list.
type List struct {
	first interface{}
	next  *List
}

// Empty returns the empty list (nil).
func Empty() *List {
	return nil
}

// New converts as list of elements to a persistent list.
func New(elems ...interface{}) *List {
	out := Empty()
	for i := len(elems) - 1; i >= 0; i-- {
		out = Cons(elems[i], out)
	}
	return out
}

// Cons constructs a new list from the element and another list.
func Cons(elem interface{}, list *List) *List {
	return &List{
		first: elem,
		next:  list,
	}
}

// First returns the first element of a list.
func (l *List) First() interface{} {
	return l.first
}

// Next returns the next list in the chain.
func (l *List) Next() *List {
	return l.next
}

// Find whether the value exists in the list by walking every value.
// Returns the value and whether or not it was found.
func (l *List) Find(value interface{}) (interface{}, bool) {
	var out interface{}
	var found bool
	l.Range(func(v interface{}) bool {
		if v == value {
			out = v
			found = true
			return false
		}
		return true
	})
	return out, found
}

// Range calls the passed in function on each element of the list.
// The function passed in may be of many types:
//
// func(value interface{}) bool:
//    Takes a value of any type and returns if the loop should continue.
//    Useful to avoid reflection where not needed and to support
//    heterogenous lists.
// func(value interface{})
//    Takes a value of any type.
//    Useful to avoid reflection where not needed and to support
//    heterogenous lists.
// func(value T) bool:
//    Takes a value of the type of element stored in the list and
//    returns if the loop should continue. Useful for homogeneous lists.
//    Is called with reflection and will panic if the type is incorrect.
// func(value T)
//    Takes a value of the type of element stored in the list and
//    returns if the loop should continue. Useful for homogeneous lists.
//    Is called with reflection and will panic if the type is incorrect.
// Range will panic if passed anything that doesn't match one of these signatures
func (l *List) Range(do interface{}) {
	cont := true
	for list := l; list != nil && cont; list = list.Next() {
		switch fn := do.(type) {
		case func(value interface{}) bool:
			cont = fn(list.First())
		case func(value interface{}):
			fn(list.First())
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
			outs := rv.Call([]reflect.Value{
				reflect.ValueOf(list.First())})
			if len(outs) != 0 {
				cont = outs[0].Interface().(bool)
			}
		}
	}
}

// Seq returns a representation of the list as a sequence
// corresponding to the elements of the list.
func (l *List) Seq() seq.Sequence {
	return &listSequence{l: l}
}

// String returns a string representation of the list.
func (l *List) String() string {
	return seq.ConvertToString(l.Seq())
}

type listSequence struct {
	l *List
}

func (l *listSequence) First() interface{} {
	return l.l.First()
}
func (l *listSequence) Next() seq.Sequence {
	if l.l.next == nil {
		return nil
	}
	return l.l.next.Seq()
}

func (l *listSequence) String() string {
	return seq.ConvertToString(l)
}
