package treeset

import (
	"fmt"
	"reflect"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/internal/btree"
)

type TSet struct {
	root *btree.TBTree
	orig *Set
	eq   eqFunc
}

// Add adds an element to the set and a new set is returned.
func (s *TSet) Add(elem interface{}) *TSet {
	s.root = s.root.Add(elem)
	return s
}

// Conj adds an element to the set. Conj implements
// a generic mechanism for building collections.
func (s *TSet) Conj(elem interface{}) interface{} {
	return s.Add(elem)
}

// At returns the elem if it exists in the set otherwise it returns nil.
func (s *TSet) At(elem interface{}) interface{} {
	v, ok := s.root.Find(elem)
	if !ok {
		return nil
	}
	return v
}

// Contains returns true if the element is in the set, false otherwise.
func (s *TSet) Contains(elem interface{}) bool {
	return s.root.Contains(elem)
}

// Find will return the key if it exists in the set and whether the
// key exists in the set. If the key is not in the set, (nil, false) is
// returned.
func (s *TSet) Find(elem interface{}) (interface{}, bool) {
	return s.root.Find(elem)
}

// Delete removes an element from the set returning a new Set without
// the element.
func (s *TSet) Delete(elem interface{}) *TSet {
	s.root = s.root.Delete(elem)
	return s
}

// Length returns the elements in the set.
func (s *TSet) Length() int {
	return s.root.Length()
}

// String returns a string serialization of the set.
func (s *TSet) String() string {
	var b strings.Builder
	fmt.Fprint(&b, "{ ")
	iter := s.Iterator()
	for iter.HasNext() {
		elem := iter.Next()
		fmt.Fprintf(&b, "%v ", elem)
	}
	fmt.Fprint(&b, "}")
	return b.String()
}

// Iterator provides a mutable iterator over the set. This allows
// efficient, heap allocation-less access to the contents. Iterators
// are not safe for concurrent access so they may not be shared
// by reference between goroutines.
func (s *TSet) Iterator() Iterator {
	return Iterator{
		impl: s.root.Iterator(),
	}
}

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows set to be called
// as a function by the 'dyn' library.
func (s *TSet) Apply(args ...interface{}) interface{} {
	k := args[0]
	return s.At(k)
}

// Equal tests if two sets are Equal by comparing the entries of each.
// Equal implements the Equaler which allows for deep
// comparisons when there are sets of sets
func (s *TSet) Equal(o interface{}) bool {
	other, ok := o.(*TSet)
	if !ok {
		return ok
	}
	if s.Length() != other.Length() {
		return false
	}
	iter := s.Iterator()
	for iter.HasNext() {
		elem := iter.Next()
		if !s.eq(other.At(elem), elem) {
			return false
		}
	}
	return true
}

// AsPersistent will transform this transient map into a persistent map.
// Once this occurs any additional actions on the transient map will fail.
func (m *TSet) AsPersistent() *Set {
	newRoot := m.root.AsPersistent()
	if newRoot == m.orig.root {
		return m.orig
	}
	return &Set{
		root: newRoot,
		eq:   m.eq,
	}
}

// MakePersistent is a generic version of AsPersistent.
func (m *TSet) MakePersistent() interface{} {
	return m.AsPersistent()
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
func (s *TSet) Range(do interface{}) {
	var rangefn func(interface{}) bool
	switch fn := do.(type) {
	case func(value interface{}) bool:
		rangefn = fn
	case func(value interface{}):
		rangefn = func(val interface{}) bool {
			fn(val)
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
		rangefn = func(val interface{}) bool {
			cont := true
			out := dyn.Apply(do, val)
			if out != nil {
				cont = out.(bool)
			}
			return cont
		}
	}
	iter := s.Iterator()
	var cont = true
	for iter.HasNext() && cont {
		elem := iter.Next()
		cont = rangefn(elem)
	}
}
