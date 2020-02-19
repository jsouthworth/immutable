// Package list implements a persistent linked list.
package list // import "jsouthworth.net/go/immutable/list"

import (
	"errors"
	"reflect"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/seq"
)

var errRangeSig = errors.New("Range requires a function: func(v vT) bool or func(v vT)")

// List is a persistent linked list.
type List struct {
	first interface{}
	next  *List
	len   int
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

// From will convert many go types to an immutable list.
// Converting some types is more efficient than others and the
// mechanisms are described below.
//
// *List:
//    Returned directly as it is already immutable.
// []interface{}:
//    New is called with the elements.
// seq.Sequable:
//    Seq is called on the value and the list is built from the resulting sequence. This will reverse the sequence.
// seq.Sequence:
//    The list is built from the sequence. Care should be taken to provide finite sequences or the list will grow without bound. This will reverse the sequence.
// []T:
//    The slice is converted to a list using reflection.
func From(value interface{}) *List {
	switch v := value.(type) {
	case *List:
		return v
	case []interface{}:
		return New(v...)
	case seq.Seqable:
		return listFromSequence(seq.Seq(v))
	case seq.Sequence:
		return listFromSequence(v)
	default:
		return listFromReflection(v)
	}
}

func listFromSequence(coll seq.Sequence) *List {
	return seq.Reduce(func(result *List, input interface{}) *List {
		return Cons(input, result)
	}, Empty(), coll).(*List)
}

func listFromReflection(value interface{}) *List {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Slice:
		out := Empty()
		for i := v.Len() - 1; i >= 0; i-- {
			out = Cons(v.Index(i).Interface(), out)
		}
		return out
	default:
		return Empty()
	}
}

// Cons constructs a new list from the element and another list.
func Cons(elem interface{}, list *List) *List {
	return list.Cons(elem)
}

// Cons constructs a new list from the element and another list.
func (l *List) Cons(elem interface{}) *List {
	return &List{
		first: elem,
		next:  l,
		len:   l.Length() + 1,
	}
}

// Conj constructs a new list from the element and another list.
// Conj implements a generic mechanism for building collections.
func (l *List) Conj(elem interface{}) interface{} {
	return l.Cons(elem)
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

// Length returns the number of members of the list
func (l *List) Length() int {
	if l == nil {
		return 0
	}
	return l.len
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
	var f func(value interface{}) bool
	switch fn := do.(type) {
	case func(value interface{}) bool:
		f = fn
	case func(value interface{}):
		f = func(value interface{}) bool {
			fn(value)
			return true
		}
	default:
		f = genRangeFunc(do)
	}
	cont := true
	for list := l; list != nil && cont; list = list.Next() {
		cont = f(list.First())
	}
}

func genRangeFunc(do interface{}) func(value interface{}) bool {
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

// Seq returns a representation of the list as a sequence
// corresponding to the elements of the list.
func (l *List) Seq() seq.Sequence {
	return &listSequence{l: l}
}

// String returns a string representation of the list.
func (l *List) String() string {
	return seq.ConvertToString(l.Seq())
}

// Equal returns whether the other value is a list, and all the values
// are equal to their corresponding partner in the other list.
func (l *List) Equal(other interface{}) bool {
	ol, isList := other.(*List)
	return isList &&
		ol.Length() == l.Length() &&
		l.elementsAreEqual(ol)
}

func (l *List) elementsAreEqual(ol *List) bool {
	allEqual := true
	l.Range(func(val interface{}) bool {
		oval := ol.First()
		ol = ol.Next()
		if !dyn.Equal(val, oval) {
			allEqual = false
			return false
		}
		return true
	})
	return allEqual
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
