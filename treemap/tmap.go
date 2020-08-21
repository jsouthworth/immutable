package treemap

import (
	"fmt"
	"strings"

	"jsouthworth.net/go/immutable/internal/btree"
)

// TMap is a transient version of a map. Changes made to a transient
// map will not effect the original persistent structure. Changes to a
// transient map occur as mutations. These mutations are then made
// persistent when the transient is transformed into a persistent
// structure. These are useful when appling multiple transforms to a
// persistent map where the intermediate results will not be seen or
// stored anywhere.
type TMap struct {
	root *btree.TBTree
	eq   eqFunc
}

// At returns the value associated with the key.
// If one is not found, nil is returned.
func (m *TMap) At(key interface{}) interface{} {
	v, ok := m.root.Find(entry{key: key})
	if !ok {
		return nil
	}
	ent := v.(entry)
	return ent.value
}

// EntryAt returns the entry (key, value pair) of the key.
// If one is not found, nil is returned.
func (m *TMap) EntryAt(key interface{}) Entry {
	v, ok := m.root.Find(entry{key: key})
	if !ok {
		return nil
	}
	ent := v.(entry)
	return ent
}

// Assoc associates a value with a key in the map.
// The transient map is modified and then returned.
func (m *TMap) Assoc(key, value interface{}) *TMap {
	m.root = m.root.Add(entry{key: key, value: value})
	return m
}

// Conj takes a value that must be an Entry. Conj implements
// a generic mechanism for building collections.
func (m *TMap) Conj(value interface{}) interface{} {
	entry := value.(Entry)
	return m.Assoc(entry.Key(), entry.Value())
}

// AsPersistent will transform this transient map into a persistent map.
// Once this occurs any additional actions on the transient map will fail.
func (m *TMap) AsPersistent() *Map {
	return &Map{
		root: m.root.AsPersistent(),
		eq:   m.eq,
	}
}

// MakePersistent is a generic version of AsPersistent.
func (m *TMap) MakePersistent() interface{} {
	return m.AsPersistent()
}

// Contains will test if the key exists in the map.
func (m *TMap) Contains(key interface{}) bool {
	return m.root.Contains(entry{key: key})
}

// Find will return the value for a key if it exists in the map and
// whether the key exists in the map. For non-nil values, exists will
// always be true.
func (m *TMap) Find(key interface{}) (value interface{}, exists bool) {
	v, ok := m.root.Find(entry{key: key})
	if !ok {
		return nil, ok
	}
	ent := v.(entry)
	return ent.value, ok
}

// Delete removes a key and associated value from the map.
func (m *TMap) Delete(key interface{}) *TMap {
	m.root = m.root.Delete(entry{key: key})
	return m
}

// Equal tests if two maps are Equal by comparing the entries of each.
// Equal implements the Equaler which allows for deep
// comparisons when there are maps of maps
func (m *TMap) Equal(o interface{}) bool {
	other, ok := o.(*TMap)
	if !ok {
		return ok
	}
	if m.Length() != other.Length() {
		return false
	}
	iter := m.Iterator()
	for iter.HasNext() {
		key, value := iter.Next()
		if !m.eq(other.At(key), value) {
			return false
		}
	}
	return true
}

// Length returns the number of entries in the map.
func (m *TMap) Length() int {
	return m.root.Length()
}

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows map to be called
// as a function by the 'dyn' library.
func (m *TMap) Apply(args ...interface{}) interface{} {
	key := args[0]
	return m.At(key)
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
func (m *TMap) Range(do interface{}) {
	// NOTE: Update other functions using the same pattern
	//       when modifying the below.
	//       This code is inlined to avoid heap allocation of
	//       the closure.
	var f func(Entry) bool
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

	iter := m.Iterator()
	cont := true
	for iter.HasNext() && cont {
		entry := iter.NextEntry().(entry)
		cont = f(entry)
	}
}

// Reduce is a fast mechanism for reducing a Map. Reduce can take
// the following types as the fn:
//
// func(init interface{}, entry Entry) interface{}
// func(init interface{}, key interface{}, value interface{}) interface{}
// func(init iT, e Entry) oT
// func(init iT, k kT, v vT) oT
// Reduce will panic if given any other function type.
func (m *TMap) Reduce(fn interface{}, init interface{}) interface{} {
	// NOTE: Update other functions using the same pattern
	//       when modifying the below.
	//       This code is inlined to avoid heap allocation of
	//       the closure.
	var rFn func(interface{}, Entry) interface{}
	switch v := fn.(type) {
	case func(interface{}, Entry) interface{}:
		rFn = v
	case func(interface{}, interface{}) interface{}:
		rFn = func(init interface{}, entry Entry) interface{} {
			return v(init, entry)
		}
	case func(interface{}, interface{}, interface{}) interface{}:
		rFn = func(init interface{}, entry Entry) interface{} {
			return v(init, entry.Key(), entry.Value())
		}
	default:
		rFn = genReduceFunc(fn)
	}
	res := init
	iter := m.Iterator()
	for iter.HasNext() {
		entry := iter.NextEntry()
		res = rFn(res, entry)
	}
	return res
}

// String returns a string representation of the map.
func (m *TMap) String() string {
	var b strings.Builder
	fmt.Fprint(&b, "{ ")
	iter := m.Iterator()
	for iter.HasNext() {
		entry := iter.NextEntry()
		fmt.Fprintf(&b, "%s ", entry)
	}
	fmt.Fprint(&b, "}")
	return b.String()
}

// Iterator provides a mutable iterator over the map. This allows
// efficient, heap allocation-less access to the contents. Iterators
// are not safe for concurrent access so they may not be shared
// by reference between goroutines.
func (m *TMap) Iterator() Iterator {
	return Iterator{
		impl: m.root.Iterator(),
	}
}
