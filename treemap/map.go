package treemap // import "jsouthworth.net/go/immutable/treemap"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/internal/btree"
	"jsouthworth.net/go/seq"
)

var errOddElements = errors.New("must supply an even number elements")
var errRangeSig = errors.New("Range requires a function: func(k kT, v vT) bool or func(k kT, v vT)")

// Entry is a map entry. Each entry consists of a key and value.
type Entry interface {
	Key() interface{}
	Value() interface{}
}

// EntryNew returns an Entry
func EntryNew(key, value interface{}) Entry {
	return entry{key, value}
}

type entry struct {
	key   interface{}
	value interface{}
}

func (e entry) Key() interface{} {
	return e.key
}

func (e entry) Value() interface{} {
	return e.value
}

func (e entry) String() string {
	return fmt.Sprintf("[%v %v]", e.key, e.value)
}

// Map is a persistent immutable map based on Red/Black
// trees. Operations on map returns a new map that shares much of the
// structure with the original map.
type Map struct {
	root *btree.BTree
}

type cmpFunc func(k1, k2 interface{}) int
type eqFunc func(k1, k2 interface{}) bool

func defaultCompare(a, b interface{}) int {
	ae := a.(entry)
	be := b.(entry)
	return dyn.Compare(ae.key, be.key)
}

func defaultEqual(a, b interface{}) bool {
	ae, aok := a.(entry)
	be, bok := b.(entry)
	return aok && bok &&
		dyn.Compare(ae.key, be.key) == 0 &&
		dyn.Equal(ae.value, be.value)
}

var empty = Map{
	root: btree.Empty(
		btree.Compare(defaultCompare),
		btree.Equal(defaultEqual),
	),
}

type mapOptions struct {
	compare cmpFunc
	equal   eqFunc
}

// Option is a type that allows changes to pluggable parts of the
// Map implementation.
type Option func(*mapOptions)

// Compare is an option to the Empty function that will allow
// one to specify a different comparison operator instead
// of the default which is from the dyn library. This is used
// for keys.
func Compare(cmp func(k1, k2 interface{}) int) Option {
	return func(o *mapOptions) {
		o.compare = cmp
	}
}

// Equal is an option to the Empty function that will allow
// one to specify a different equality operator instead
// of the default which is from the dyn library. This is used
// for values.
func Equal(eq func(v1, v2 interface{}) bool) Option {
	return func(o *mapOptions) {
		o.equal = eq
	}
}

// Empty returns a new empty persistent map, one may supply options
// for the map by using one of the option generating functions and
// providing that to Empty.
func Empty(options ...Option) *Map {
	if len(options) == 0 {
		return &empty
	}

	opts := mapOptions{
		compare: dyn.Compare,
		equal:   dyn.Equal,
	}
	for _, opt := range options {
		opt(&opts)
	}

	cmp := func(a, b interface{}) int {
		ae := a.(entry)
		be := b.(entry)
		return opts.compare(ae.key, be.key)
	}
	eq := func(a, b interface{}) bool {
		ae, aok := a.(entry)
		be, bok := b.(entry)
		return aok && bok &&
			opts.compare(ae.key, be.key) == 0 &&
			opts.equal(ae.value, be.value)
	}

	return &Map{
		root: btree.Empty(
			btree.Compare(cmp),
			btree.Equal(eq),
		),
	}
}

// New converts a list of elements to a persistent map
// by associating them pairwise. New will panic if the
// number of elements is not even.
func New(elems ...interface{}) *Map {
	return newWithOptions(elems)
}

func newWithOptions(elems []interface{}, options ...Option) *Map {
	if len(elems)%2 != 0 {
		panic(errOddElements)
	}
	out := Empty(options...)
	for i := 0; i < len(elems); i += 2 {
		out = out.Assoc(elems[i], elems[i+1])
	}
	return out
}

// From will convert many different go types to an immutable map.
// Converting some types is more efficient than others and the mechanisms
// are described below.
//
// *Map:
//    Returned directly as it is already immutable.
// map[interface{}]interface{}:
//    Converted directly by looping over the map and calling Assoc starting with an empty transient map. The transient map is the converted to a persistent one and returned.
// []Entry:
//    The entries are looped over and Assoc is called on an empty transient map. The transient map is converted to a persistent map and then returned.
// []interface{}:
//    The elements are passed to New.
// map[kT]vT:
//    Reflection is used to loop over the entries of the map and associate them with an empty transient map. The transient map is converted to a persistent map and then returned.
// []T:
//    Reflection is used to convert the slice to []interface{} and then passed to New.
func From(value interface{}, options ...Option) *Map {
	switch v := value.(type) {
	case *Map:
		return v
	case map[interface{}]interface{}:
		out := Empty(options...)
		for key, val := range v {
			out = out.Assoc(key, val)
		}
		return out
	case []Entry:
		out := Empty(options...)
		for _, entry := range v {
			out = out.Assoc(entry.Key(), entry.Value())
		}
		return out
	case []interface{}:
		return newWithOptions(v, options...)
	default:
		return mapFromReflection(value)
	}
}

func mapFromReflection(value interface{}, options ...Option) *Map {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		out := Empty(options...)
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			out = out.Assoc(key.Interface(), val.Interface())
		}
		return out
	case reflect.Slice:
		sl := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			sl[i] = elem.Interface()
		}
		return newWithOptions(sl, options...)
	default:
		return Empty(options...)
	}
}

// At returns the value associated with the key.
// If one is not found, nil is returned.
func (m *Map) At(key interface{}) interface{} {
	v, ok := m.root.Find(entry{key: key})
	if !ok {
		return nil
	}
	ent := v.(entry)
	return ent.value
}

// EntryAt returns the entry (key, value pair) of the key.
// If one is not found, nil is returned.
func (m *Map) EntryAt(key interface{}) Entry {
	v, ok := m.root.Find(entry{key: key})
	if !ok {
		return nil
	}
	ent := v.(entry)
	return ent
}

// Contains will test if the key exists in the map.
func (m *Map) Contains(key interface{}) bool {
	return m.root.Contains(entry{key: key})
}

// Find will return the value for a key if it exists in the map and
// whether the key exists in the map. For non-nil values, exists will
// always be true.
func (m *Map) Find(key interface{}) (value interface{}, exists bool) {
	v, ok := m.root.Find(entry{key: key})
	if !ok {
		return nil, ok
	}
	ent := v.(entry)
	return ent.value, ok
}

// Assoc associates a value with a key in the map.
// A new persistent map is returned if the key and value
// are different from one already in the map, if the entry
// is already in the map the original map is returned.
func (m *Map) Assoc(key, value interface{}) *Map {
	root := m.root.Add(entry{key: key, value: value})
	switch {
	case root == m.root:
		return m
	default:
		return &Map{
			root: root,
		}
	}
}

// Conj associates a value with a key in the map.
func (m *Map) Conj(elem interface{}) interface{} {
	entry := elem.(Entry)
	return m.Assoc(entry.Key(), entry.Value())
}

// Delete removes a key and associated value from the map.
func (m *Map) Delete(key interface{}) *Map {
	root := m.root.Delete(entry{key: key})
	if root == m.root {
		return m
	}
	return &Map{
		root: root,
	}
}

// Length returns the number of entries in the map.
func (m *Map) Length() int {
	return m.root.Length()
}

// Range will loop over the entries in the Map and call 'do' on each entry.
// The 'do' function may be of many types:
//
// func(key, value interface{}) bool:
//    Takes empty interfaces and returns if the loop should continue.
//    Useful to avoid reflection or for hetrogenous maps.
// func(key, value interface{}):
//    Takes empty interfaces.
//    Useful to avoid reflection or for hetrogenous maps.
// func(entry Entry) bool:
//    Takes the Entry type and returns if the loop should continue
//    Is called directly and avoids entry unpacking if not necessary.
// func(entry Entry):
//    Takes the Entry type.
//    Is called directly and avoids entry unpacking if not necessary.
// func(k kT, v vT) bool
//    Takes a key of key type and a value of value type and returns if the loop should contiune.
//    Is called with reflection and will panic if the kT and vT types are incorrect.
// func(k kT, v vT)
//    Takes a key of key type and a value of value type.
//    Is called with reflection and will panic if the kT and vT types are incorrect.
// Range will panic if passed anything not matching these signatures.
func (m *Map) Range(do interface{}) {
	// NOTE: Update other functions using the same pattern
	//       when modifying the below.
	//       This code is inlined to avoid heap allocation of
	//       the closure.
	var f func(e Entry) bool
	switch fn := do.(type) {
	case func(key, value interface{}) bool:
		f = func(entry Entry) bool {
			return fn(entry.Key(), entry.Value())
		}
	case func(key, value interface{}):
		f = func(entry Entry) bool {
			fn(entry.Key(), entry.Value())
			return true
		}
	case func(e Entry) bool:
		f = fn
	case func(e Entry):
		f = func(entry Entry) bool {
			fn(entry)
			return true
		}
	default:
		f = genRangeFunc(do)
	}

	iter := m.root.Iterator()
	var cont = true
	for iter.HasNext() && cont {
		entry := iter.Next().(Entry)
		cont = f(entry)
	}
}

func genRangeFunc(do interface{}) func(Entry) bool {
	rv := reflect.ValueOf(do)
	if rv.Kind() != reflect.Func {
		panic(errRangeSig)
	}
	rt := rv.Type()
	if rt.NumIn() != 2 || rt.NumOut() > 1 {
		panic(errRangeSig)
	}
	if rt.NumOut() == 1 &&
		rt.Out(0).Kind() != reflect.Bool {
		panic(errRangeSig)
	}
	return func(entry Entry) bool {
		out := dyn.Apply(do, entry.Key(), entry.Value())
		if out != nil {
			return out.(bool)
		}
		return true
	}
}

// Seq returns a seralized sequence of Entry
// corresponding to the maps entries.
func (m *Map) Seq() seq.Sequence {
	iter := m.root.Iterator()
	if !iter.HasNext() {
		return nil
	}
	return sequenceNew(iter)
}

// String returns a string representation of the map.
func (m *Map) String() string {
	var b strings.Builder
	fmt.Fprint(&b, "{ ")
	m.Range(func(entry Entry) {
		fmt.Fprintf(&b, "%s ", entry)
	})
	fmt.Fprint(&b, "}")
	return b.String()
}

// AsNative returns the map converted to a go native map type.
func (m *Map) AsNative() map[interface{}]interface{} {
	out := make(map[interface{}]interface{})
	m.Range(func(key, val interface{}) {
		out[key] = val
	})
	return out
}

// Equal tests if two maps are Equal by comparing the entries of each.
// Equal implements the Equaler which allows for deep
// comparisons when there are maps of maps
func (m *Map) Equal(o interface{}) bool {
	other, ok := o.(*Map)
	if !ok {
		return ok
	}
	if m.Length() != other.Length() {
		return false
	}
	foundAll := true
	m.Range(func(key, value interface{}) bool {
		if !dyn.Equal(other.At(key), value) {
			foundAll = false
			return false
		}
		return true
	})
	return foundAll
}

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows map to be called
// as a function by the 'dyn' library.
func (m *Map) Apply(args ...interface{}) interface{} {
	k := args[0]
	return m.At(k)
}
