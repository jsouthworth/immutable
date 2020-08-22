// Package treeset implements an immutable Set datastructure on top of treemap
//
// A note about Value comparability, by default, go's comparison operators
// will be used for any comparable type. Any type may implement the
// Compare(other interface{}) int interface to override this requirement.
package treeset // import "jsouthworth.net/go/immutable/treeset"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/internal/btree"
	"jsouthworth.net/go/seq"
)

var errRangeSig = errors.New("Range requires a function: func(v vT) bool or func(v vT)")

// Set is a persistent ordered set implementation.
type Set struct {
	root *btree.BTree
	eq   eqFunc
}

type cmpFunc func(k1, k2 interface{}) int
type eqFunc func(k1, k2 interface{}) bool

func defaultCompare(a, b interface{}) int {
	return dyn.Compare(a, b)
}

func defaultEqual(a, b interface{}) bool {
	return dyn.Compare(a, b) == 0
}

var empty = Set{
	root: btree.Empty(
		btree.Compare(defaultCompare),
		btree.Equal(defaultEqual),
	),
	eq: defaultEqual,
}

type setOptions struct {
	compare cmpFunc
}

// Option is a type that allows changes to pluggable parts of the
// Map implementation.
type Option func(*setOptions)

// Compare is an option to the Empty function that will allow
// one to specify a different comparison operator instead
// of the default which is from the dyn library. This is used
// for keys.
func Compare(cmp func(k1, k2 interface{}) int) Option {
	return func(o *setOptions) {
		o.compare = cmp
	}
}

// Empty returns a new empty persistent set, one may supply options
// for the set by using one of the option generating functions and
// providing that to Empty.
func Empty(options ...Option) *Set {
	if len(options) == 0 {
		return &empty
	}

	opts := setOptions{
		compare: defaultCompare,
	}
	for _, opt := range options {
		opt(&opts)
	}

	eq := func(a, b interface{}) bool {
		return opts.compare(a, b) == 0
	}

	return &Set{
		root: btree.Empty(
			btree.Compare(opts.compare),
			btree.Equal(eq),
		),
		eq: eq,
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

func newWithOptions(elems []interface{}, options ...Option) *Set {
	s := Empty(options...).AsTransient()
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
func From(value interface{}, options ...Option) *Set {
	switch v := value.(type) {
	case *Set:
		return v
	case *TSet:
		return v.AsPersistent()
	case map[interface{}]struct{}:
		s := Empty(options...).AsTransient()
		for k := range v {
			s = s.Add(k)
		}
		return s.AsPersistent()
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

func setFromSequence(coll seq.Sequence, options ...Option) *Set {
	if coll == nil {
		return Empty(options...)
	}
	return seq.Reduce(func(result *TSet, input interface{}) *TSet {
		return result.Add(input)
	}, Empty().AsTransient(), coll).(*TSet).AsPersistent()
}

func setFromReflection(value interface{}, options ...Option) *Set {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		out := Empty(options...).AsTransient()
		for _, key := range v.MapKeys() {
			out = out.Add(key.Interface())
		}
		return out.AsPersistent()
	case reflect.Slice:
		out := Empty(options...).AsTransient()
		for i := 0; i < v.Len(); i++ {
			out = out.Add(v.Index(i).Interface())
		}
		return out.AsPersistent()
	default:
		if value == nil {
			return Empty(options...)
		}
		return newWithOptions([]interface{}{value}, options...)
	}
}

// Add adds an element to the set and a new set is returned.
func (s *Set) Add(elem interface{}) *Set {
	root := s.root.Add(elem)
	if root == s.root {
		return s
	}
	return &Set{
		root: root,
		eq:   s.eq,
	}
}

// Conj adds an element to the set. Conj implements
// a generic mechanism for building collections.
func (s *Set) Conj(elem interface{}) interface{} {
	return s.Add(elem)
}

// At returns the elem if it exists in the set otherwise it returns nil.
func (s *Set) At(elem interface{}) interface{} {
	v, ok := s.root.Find(elem)
	if !ok {
		return nil
	}
	return v
}

// Contains returns true if the element is in the set, false otherwise.
func (s *Set) Contains(elem interface{}) bool {
	return s.root.Contains(elem)
}

// Find will return the key if it exists in the set and whether the
// key exists in the set. If the key is not in the set, (nil, false) is
// returned.
func (s *Set) Find(elem interface{}) (interface{}, bool) {
	return s.root.Find(elem)
}

// Delete removes an element from the set returning a new Set without
// the element.
func (s *Set) Delete(elem interface{}) *Set {
	root := s.root.Delete(elem)
	if root == s.root {
		return s
	}
	return &Set{
		root: root,
		eq:   s.eq,
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

// Length returns the elements in the set.
func (s *Set) Length() int {
	return s.root.Length()
}

// String returns a string serialization of the set.
func (s *Set) String() string {
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

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows set to be called
// as a function by the 'dyn' library.
func (s *Set) Apply(args ...interface{}) interface{} {
	k := args[0]
	return s.At(k)
}

// Seq returns a seralized sequence of interface{}
// corresponding to the sets entries.
func (s *Set) Seq() seq.Sequence {
	iter := s.root.Iterator()
	if !iter.HasNext() {
		return nil
	}
	return sequenceNew(iter)
}

// Equal tests if two sets are Equal by comparing the entries of each.
// Equal implements the Equaler which allows for deep
// comparisons when there are sets of sets
func (s *Set) Equal(o interface{}) bool {
	other, ok := o.(*Set)
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

// Iterator provides a mutable iterator over the set. This allows
// efficient, heap allocation-less access to the contents. Iterators
// are not safe for concurrent access so they may not be shared
// by reference between goroutines.
func (s *Set) Iterator() Iterator {
	return Iterator{
		impl: s.root.Iterator(),
	}
}

// AsTransient will return a transient map that shares
// structure with the persistent set.
func (s *Set) AsTransient() *TSet {
	return &TSet{
		root: s.root.AsTransient(),
		eq:   s.eq,
		orig: s,
	}
}

// MakeTransient is a generic version of AsTransient.
func (s *Set) MakeTransient() interface{} {
	return s.AsTransient()
}

// Transform takes a set of actions and performs them
// on the persistent set. It does this by making a transient
// set and calling each action on it, then converting it back
// to a persistent set.
func (m *Set) Transform(actions ...func(*TSet)) *Set {
	out := m.AsTransient()
	for _, action := range actions {
		action(out)
	}
	return out.AsPersistent()
}

// Iterator is a mutable iterator for a set. It has a fixed size
// stack, the size of which is computed from the maximum number of
// nested nodes possible based on the branching factor.
type Iterator struct {
	impl btree.Iterator
}

// Next provides the next key value pair and increments the cursor.
func (i *Iterator) Next() interface{} {
	return i.impl.Next()
}

// HasNext is true when there are more elements to be iterated over.
func (i *Iterator) HasNext() bool {
	return i.impl.HasNext()
}

type sequence struct {
	iter btree.Iterator
}

func sequenceNew(iter btree.Iterator) *sequence {
	return &sequence{
		iter: iter,
	}
}

func (s *sequence) First() interface{} {
	return s.iter.Next()
}

func (s *sequence) Next() seq.Sequence {
	new := &(*s)
	hasNext := new.iter.HasNext()
	if !hasNext {
		return nil
	}
	return new
}

func (s *sequence) String() string {
	return seq.ConvertToString(s)
}
