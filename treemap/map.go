package treemap // import "jsouthworth.net/go/immutable/treemap"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/seq"
)

var errOddElements = errors.New("must supply an even number elements")
var errRangeSig = errors.New("Range requires a function: func(k kT, v vT) bool or func(k kT, v vT)")

// Entry is a map entry. Each entry consists of a key and value.
type Entry interface {
	Key() interface{}
	Value() interface{}
}

// Map is a persistent immutable map based on Red/Black
// trees. Operations on map returns a new map that shares much of the
// structure with the original map.
type Map struct {
	compare cmpFunc
	root    tree
	count   int
}

var empty = Map{
	compare: dyn.Compare,
	root:    &leaf{cmp: dyn.Compare},
	count:   0,
}

type mapOpts struct {
	compare cmpFunc
}
type mapOpt func(*mapOpts)

// Compare is an option to the Empty function that will allow
// one to specify a different comparison operator instead
// of the default which is from the dyn library.
func Compare(cmp func(k1, k2 interface{}) int) mapOpt {
	return func(o *mapOpts) {
		o.compare = cmp
	}
}

// Empty returns a new empty persistent map, one may supply options
// for the map by using one of the option generating functions and
// providing that to Empty.
func Empty(options ...mapOpt) *Map {
	var opts mapOpts
	for _, opt := range options {
		opt(&opts)
	}
	if opts.compare == nil {
		return &empty
	}
	return &Map{
		compare: opts.compare,
		root:    &leaf{cmp: opts.compare},
		count:   0,
	}
}

// New converts a list of elements to a persistent map
// by associating them pairwise. New will panic if the
// number of elements is not even.
func New(elems ...interface{}) *Map {
	return newWithOptions(elems)
}

func newWithOptions(elems []interface{}, options ...mapOpt) *Map {
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
func From(value interface{}, options ...mapOpt) *Map {
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

func mapFromReflection(value interface{}, options ...mapOpt) *Map {
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
	ent, ok := get(m.root, key)
	if !ok {
		return nil
	}
	return ent.value
}

// EntryAt returns the entry (key, value pair) of the key.
// If one is not found, nil is returned.
func (m *Map) EntryAt(key interface{}) Entry {
	v, ok := get(m.root, key)
	if !ok {
		return nil
	}
	return v
}

// Contains will test if the key exists in the map.
func (m *Map) Contains(key interface{}) bool {
	_, ok := get(m.root, key)
	return ok
}

// Find will return the value for a key if it exists in the map and
// whether the key exists in the map. For non-nil values, exists will
// always be true.
func (m *Map) Find(key interface{}) (value interface{}, exists bool) {
	return get(m.root, key)
}

// Assoc associates a value with a key in the map.
// A new persistent map is returned if the key and value
// are different from one already in the map, if the entry
// is already in the map the original map is returned.
func (m *Map) Assoc(key, value interface{}) *Map {
	root, added := insert(m.root, key, value)
	switch {
	case root == m.root:
		return m
	case added:
		return &Map{
			compare: m.compare,
			root:    root,
			count:   m.count + 1,
		}
	default:
		return &Map{
			compare: m.compare,
			root:    root,
			count:   m.count,
		}
	}
}

// Delete removes a key and associated value from the map.
func (m *Map) Delete(key interface{}) *Map {
	root := _delete(m.root, key)
	if root == m.root {
		return m
	}
	return &Map{
		compare: m.compare,
		root:    root,
		count:   m.count - 1,
	}
}

// Length returns the number of entries in the map.
func (m *Map) Length() int {
	return m.count
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
	s := seq.Seq(m)
	if s == nil {
		return
	}
	var cont = true
	for s != nil && cont {
		entry := seq.First(s).(Entry)
		switch fn := do.(type) {
		case func(key, value interface{}) bool:
			cont = fn(entry.Key(), entry.Value())
		case func(key, value interface{}):
			fn(entry.Key(), entry.Value())
		case func(e Entry) bool:
			cont = fn(entry)
		case func(e Entry):
			fn(entry)
		default:
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
			out := dyn.Apply(do, entry.Key(), entry.Value())
			if out != nil {
				cont = out.(bool)
			}
		}
		s = seq.Seq(seq.Next(s))
	}
}

// Seq returns a seralized sequence of Entry
// corresponding to the maps entries.
func (m *Map) Seq() seq.Sequence {
	if _, isLeaf := m.root.(*leaf); isLeaf {
		return nil
	}
	return sequenceNew(m.root)
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
