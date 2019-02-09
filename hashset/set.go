// Package hashset implements an immutable Set datastructure on top of hashmap
package hashset // import "jsouthworth.net/go/immutable/hashset"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"jsouthworth.net/go/immutable/hashmap"
	"jsouthworth.net/go/seq"
)

var errRangeSig = errors.New("Range requires a function: func(v vT) bool or func(v vT)")

// Set is a persistent unordered set implementation.
type Set struct {
	backingMap *hashmap.Map
}

// Empty returns the empty set.
func Empty() *Set {
	return &Set{
		backingMap: hashmap.Empty(),
	}
}

// New returns a set containing the supplied elements.
func New(elems ...interface{}) *Set {
	s := Empty().AsTransient()
	for _, elem := range elems {
		s = s.Add(elem)
	}
	return s.AsPersistent()
}

// From will convert many different go types to an immutable map.
// Converting some types is more efficient than others and the mechanisms
// are described below.
//
// *Set:
//    Returned directly as it is already immutable.
// *TSet:
//    AsPersistent is called on it and the result is returned.
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
func From(value interface{}) *Set {
	switch v := value.(type) {
	case *Set:
		return v
	case *TSet:
		return v.AsPersistent()
	case map[interface{}]struct{}:
		s := Empty().AsTransient()
		for k := range v {
			s = s.Add(k)
		}
		return s.AsPersistent()
	case []interface{}:
		return New(v...)
	case seq.Seqable:
		return setFromSequence(v.Seq())
	case seq.Sequence:
		return setFromSequence(v)
	default:
		return setFromReflection(value)
	}
}

func setFromSequence(coll seq.Sequence) *Set {
	if coll == nil {
		return Empty()
	}
	return seq.Reduce(func(result *TSet, input interface{}) *TSet {
		return result.Add(input)
	}, Empty().AsTransient(), coll).(*TSet).AsPersistent()
}

func setFromReflection(value interface{}) *Set {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		out := Empty().AsTransient()
		for _, key := range v.MapKeys() {
			out.Add(key.Interface())
		}
		return out.AsPersistent()
	case reflect.Slice:
		out := Empty().AsTransient()
		for i := 0; i < v.Len(); i++ {
			out = out.Add(v.Index(i).Interface())
		}
		return out.AsPersistent()
	default:
		if value == nil {
			return Empty()
		}
		return New(value)
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
			outs := rv.Call([]reflect.Value{
				reflect.ValueOf(key)})
			if len(outs) != 0 {
				cont = outs[0].Interface().(bool)
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

// AsTransient returns a mutable copy on write version of the set.
func (s *Set) AsTransient() *TSet {
	return &TSet{
		backingMap: s.backingMap.AsTransient(),
	}
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

// TSet is a transient copy on write version of Set. Changes made to a
// transient set will not effect the original persistent
// structure. Changes to a transient set occur as mutations. These
// mutations are then made persistent when the transient is transformed
// into a persistent structure. These are useful when appling multiple
// transforms to a persistent set where the intermediate results will not
// be seen or stored anywhere.
type TSet struct {
	backingMap *hashmap.TMap
}

// Add adds an element to the set as a mutation and original TSet is returned.
func (s *TSet) Add(elem interface{}) *TSet {
	m := s.backingMap.Assoc(elem, nil)
	s.backingMap = m
	return s
}

// At returns the elem if it exists in the set otherwise it returns nil.
func (s *TSet) At(elem interface{}) interface{} {
	if s.backingMap.Contains(elem) {
		return elem
	}
	return nil
}

// Contains returns true if the element is in the set, false otherwise.
func (s *TSet) Contains(elem interface{}) bool {
	return s.backingMap.Contains(elem)
}

// Delete removes an element from the set as a mutation returning the
// original TSet.
func (s *TSet) Delete(elem interface{}) *TSet {
	m := s.backingMap.Delete(elem)
	s.backingMap = m
	return s
}

// Length returns the elements in the set.
func (s *TSet) Length() int {
	return s.backingMap.Length()
}

// AsPersistent will transform this transient set into a persistent set.
// Once this occurs any additional actions on the transient set will fail.
func (s *TSet) AsPersistent() *Set {
	return &Set{
		backingMap: s.backingMap.AsPersistent(),
	}
}
