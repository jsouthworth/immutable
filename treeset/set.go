// Package treeset implements an immutable Set datastructure on top of treemap
package treeset // import "jsouthworth.net/go/immutable/treeset"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/treemap"
	"jsouthworth.net/go/seq"
)

var errRangeSig = errors.New("Range requires a function: func(v vT) bool or func(v vT)")

// Set is a persistent unordered set implementation.
type Set struct {
	backingMap *treemap.Map
}

// Empty returns the empty set.
func Empty(options ...treemap.Option) *Set {
	return &Set{
		backingMap: treemap.Empty(options...),
	}
}

// New returns a set containing the supplied elements.
func New(elems ...interface{}) *Set {
	s := Empty()
	for _, elem := range elems {
		s = s.Add(elem)
	}
	return s
}

func newWithOptions(elems []interface{}, options ...treemap.Option) *Set {
	s := Empty(options...)
	for _, elem := range elems {
		s = s.Add(elem)
	}
	return s
}

// From will convert many different go types to an immutable map.
// Converting some types is more efficient than others and the mechanisms
// are described below.
//
// *Set:
//    Returned directly as it is already immutable.
// map[interface{}]struct{}:
//    Converted directly by looping over the map and calling Add starting with an empty transient set. The transient set is the converted to a persistent one and returned.
// []interface{}:
//    The elements are passed to New.
// map[kT]vT:
//    Reflection is used to loop over the keys of the map and add them to an empty transient set. The transient set is converted to a persistent map and then returned.
// []T:
//    Reflection is used to convert the slice to add the elements to the set.
// seq.Sequence:
//    The sequence is reduced into a transient set that is made persistent on return.
// seq.Sequable:
//    A sequence is obtained using Seq() and then the sequence is reduced into a transient set that is made persistent on return.
func From(value interface{}, options ...treemap.Option) *Set {
	switch v := value.(type) {
	case *Set:
		return v
	case map[interface{}]struct{}:
		s := Empty(options...)
		for k := range v {
			s = s.Add(k)
		}
		return s
	case []interface{}:
		return newWithOptions(v, options...)
	case seq.Seqable:
		return setFromSequence(v.Seq(), options...)
	case seq.Sequence:
		return setFromSequence(v, options...)
	default:
		return setFromReflection(value, options...)
	}
}

func setFromSequence(coll seq.Sequence, options ...treemap.Option) *Set {
	if coll == nil {
		return Empty(options...)
	}
	return seq.Reduce(func(result *Set, input interface{}) *Set {
		return result.Add(input)
	}, Empty(), coll).(*Set)
}

func setFromReflection(value interface{}, options ...treemap.Option) *Set {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		out := Empty(options...)
		for _, key := range v.MapKeys() {
			out = out.Add(key.Interface())
		}
		return out
	case reflect.Slice:
		out := Empty(options...)
		for i := 0; i < v.Len(); i++ {
			out = out.Add(v.Index(i).Interface())
		}
		return out
	default:
		if value == nil {
			return Empty(options...)
		}
		return newWithOptions([]interface{}{value}, options...)
	}
}

// Add adds an element to the set and a new set is returned.
func (s *Set) Add(elem interface{}) *Set {
	m := s.backingMap.Assoc(elem, nil)
	if m == s.backingMap {
		return s
	}
	return &Set{
		backingMap: m,
	}
}

// At returns the elem if it exists in the set otherwise it returns nil.
func (s *Set) At(elem interface{}) interface{} {
	if s.backingMap.Contains(elem) {
		return elem
	}
	return nil
}

// Contains returns true if the element is in the set, false otherwise.
func (s *Set) Contains(elem interface{}) bool {
	return s.backingMap.Contains(elem)
}

// Find will return the key if it exists in the set and whether the
// key exists in the set. If the key is not in the set, (nil, false) is
// returned.
func (s *Set) Find(elem interface{}) (interface{}, bool) {
	if s.backingMap.Contains(elem) {
		return elem, true
	}
	return nil, false
}

// Delete removes an element from the set returning a new Set without
// the element.
func (s *Set) Delete(elem interface{}) *Set {
	m := s.backingMap.Delete(elem)
	if m == s.backingMap {
		return s
	}
	return &Set{
		backingMap: m,
	}
}

// Range calls the passed in function on each element of the set.
// The function passed in may be of many types:
//
// func(value interface{}) bool:
//    Takes a value of any type and returns if the loop should continue.
//    Useful to avoid reflection where not needed and to support
//    heterogenous sets.
// func(value interface{})
//    Takes a value of any type.
//    Useful to avoid reflection where not needed and to support
//    heterogenous sets.
// func(value T) bool:
//    Takes a value of the type of element stored in the set and
//    returns if the loop should continue. Useful for homogeneous sets.
//    Is called with reflection and will panic if the type is incorrect.
// func(value T)
//    Takes a value of the type of element stored in the set and
//    returns if the loop should continue. Useful for homogeneous sets.
//    Is called with reflection and will panic if the type is incorrect.
// Range will panic if passed anything that doesn't match one of these signatures
func (s *Set) Range(do interface{}) {
	var rangefn func(interface{}, interface{}) bool
	switch fn := do.(type) {
	case func(value interface{}) bool:
		rangefn = func(key, _ interface{}) bool {
			return fn(key)
		}
	case func(value interface{}):
		rangefn = func(key, _ interface{}) bool {
			fn(key)
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
		rangefn = func(key, _ interface{}) bool {
			cont := true
			out := dyn.Apply(do, key)
			if out != nil {
				cont = out.(bool)
			}
			return cont
		}
	}
	s.backingMap.Range(rangefn)
}

// Length returns the elements in the set.
func (s *Set) Length() int {
	return s.backingMap.Length()
}

// String returns a string serialization of the set.
func (s *Set) String() string {
	var b strings.Builder
	fmt.Fprint(&b, "{ ")
	s.Range(func(elem interface{}) {
		fmt.Fprintf(&b, "%v ", elem)
	})
	fmt.Fprint(&b, "}")
	return b.String()
}

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows set to be called
// as a function by the 'dyn' library.
func (s *Set) Apply(args ...interface{}) interface{} {
	k := args[0]
	return s.At(k)
}

// Seq returns a seralized sequence of interface{}
// corresponding to the set's elements.
func (s *Set) Seq() seq.Sequence {
	mSeq := s.backingMap.Seq()
	if mSeq == nil {
		return nil
	}
	return &setSeq{mSeq: mSeq}
}

// Equal tests if two sets are Equal by comparing the entries of each.
// Equal implements the Equaler which allows for deep
// comparisons when there are sets of sets
func (s *Set) Equal(o interface{}) bool {
	other, ok := o.(*Set)
	if !ok {
		return ok
	}
	return s.backingMap.Equal(other.backingMap)
}

type setSeq struct {
	mSeq seq.Sequence
}

func (s *setSeq) First() interface{} {
	out := s.mSeq.First()
	return out.(treemap.Entry).Key()
}

func (s *setSeq) Next() seq.Sequence {
	next := s.mSeq.Next()
	if next == nil {
		return nil
	}
	return &setSeq{mSeq: next}
}
