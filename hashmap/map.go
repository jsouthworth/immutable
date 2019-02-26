package hashmap // import "jsouthworth.net/go/immutable/hashmap"

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"sync/atomic"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/hash"
	"jsouthworth.net/go/seq"
)

const (
	shiftBits = 5
	width     = 1 << shiftBits
	maskValue = width - 1
)

var errTafterP = errors.New("transient used after persistent call")
var errOddElements = errors.New("must supply an even number elements")
var errRangeSig = errors.New("Range requires a function: func(k kT, v vT) bool or func(k kT, v vT)")

var zero = atomicZero()

// Entry is a map entry. Each entry consists of a key and value.
type Entry interface {
	Key() interface{}
	Value() interface{}
}

// Map is a persistent immutable map. Operations on
// map returns a new map that shares much of the
// structure with the original map.
type Map struct {
	hashSeed uintptr
	count    int
	root     node
}

// Empty returns a new empty persistent map with a random hashSeed.
func Empty() *Map {
	seed := uintptr(rand.Uint64())
	return &Map{
		hashSeed: seed,
		root:     emptySeededBitmapNode(seed),
	}
}

// New converts a list of elements to a persistent map
// by associating them pairwise. New will panic if the
// number of elements is not even.
func New(elems ...interface{}) *Map {
	if len(elems)%2 != 0 {
		panic(errOddElements)
	}
	out := Empty().AsTransient()
	for i := 0; i < len(elems); i += 2 {
		out = out.Assoc(elems[i], elems[i+1])
	}
	return out.AsPersistent()
}

// From will convert many different go types to an immutable map.
// Converting some types is more efficient than others and the mechanisms
// are described below.
//
// *Map:
//    Returned directly as it is already immutable.
// *TMap:
//    AsPersistent is called on it and the result is returned.
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
func From(value interface{}) *Map {
	switch v := value.(type) {
	case *Map:
		return v
	case *TMap:
		return v.AsPersistent()
	case map[interface{}]interface{}:
		out := Empty().AsTransient()
		for key, val := range v {
			out = out.Assoc(key, val)
		}
		return out.AsPersistent()
	case []Entry:
		out := Empty().AsTransient()
		for _, entry := range v {
			out = out.Assoc(entry.Key(), entry.Value())
		}
		return out.AsPersistent()
	case []interface{}:
		return New(v...)
	default:
		return mapFromReflection(value)
	}
}

func mapFromReflection(value interface{}) *Map {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		out := Empty().AsTransient()
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			out.Assoc(key.Interface(), val.Interface())
		}
		return out.AsPersistent()
	case reflect.Slice:
		sl := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			sl[i] = elem.Interface()
		}
		return New(sl...)
	default:
		return Empty()
	}
}

// At returns the value associated with the key.
// If one is not found, nil is returned.
func (m *Map) At(key interface{}) interface{} {
	v, ok := m.root.find(0, hash.Any(key, m.hashSeed), key)
	if !ok {
		return nil
	}
	return v
}

// EntryAt returns the entry (key, value pair) of the key.
// If one is not found, nil is returned.
func (m *Map) EntryAt(key interface{}) Entry {
	v, ok := m.root.find(0, hash.Any(key, m.hashSeed), key)
	if !ok {
		return nil
	}
	return entry{k: key, v: v}
}

// Assoc associates a value with a key in the map.
// A new persistent map is returned if the key and value
// are different from one already in the map, if the entry
// is already in the map the original map is returned.
func (m *Map) Assoc(key, value interface{}) *Map {
	root, added := m.root.assoc(zero, 0,
		hash.Any(key, m.hashSeed), key, value)
	switch {
	case root == m.root:
		return m
	case added:
		return &Map{
			hashSeed: m.hashSeed,
			count:    m.count + 1,
			root:     root,
		}
	default: //replaced key
		return &Map{
			hashSeed: m.hashSeed,
			count:    m.count,
			root:     root,
		}
	}
}

// AsNative returns the map converted to a go native map type.
func (m *Map) AsNative() map[interface{}]interface{} {
	out := make(map[interface{}]interface{})
	m.Range(func(key, val interface{}) {
		out[key] = val
	})
	return out
}

// AsTransient will return a transient map that shares
// structure with the persistent map.
func (m *Map) AsTransient() *TMap {
	return &TMap{
		hashSeed: m.hashSeed,
		count:    m.count,
		root:     m.root,
		edit:     atomicOne(),
	}
}

// Contains will test if the key exists in the map.
func (m *Map) Contains(key interface{}) bool {
	_, ok := m.root.find(0, hash.Any(key, m.hashSeed), key)
	return ok
}

// Find will return the value for a key if it exists in the map and
// whether the key exists in the map. For non-nil values, exists will
// always be true.
func (m *Map) Find(key interface{}) (value interface{}, exists bool) {
	return m.root.find(0, hash.Any(key, m.hashSeed), key)
}

// Delete removes a key and associated value from the map.
func (m *Map) Delete(key interface{}) *Map {
	root, removed := m.root.without(zero, 0,
		hash.Any(key, m.hashSeed), key)
	switch {
	case root == nil:
		return &Map{
			hashSeed: m.hashSeed,
			count:    m.count - 1,
			root:     emptySeededBitmapNode(m.hashSeed),
		}
	case removed:
		return &Map{
			hashSeed: m.hashSeed,
			count:    m.count - 1,
			root:     root,
		}
	default:
		return m
	}
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
	fn := genRangeFunc(do)
	m.root.rnge(fn)
}

func genRangeFunc(do interface{}) func(Entry) bool {
	switch fn := do.(type) {
	case func(key, value interface{}) bool:
		return func(entry Entry) bool {
			return fn(entry.Key(), entry.Value())
		}
	case func(key, value interface{}):
		return func(entry Entry) bool {
			fn(entry.Key(), entry.Value())
			return true
		}
	case func(e Entry) bool:
		return fn
	case func(e Entry):
		return func(entry Entry) bool {
			fn(entry)
			return true
		}
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
		return func(entry Entry) bool {
			out := dyn.Apply(do, entry.Key(), entry.Value())
			if out != nil {
				return out.(bool)
			}
			return true
		}

	}
}

// Seq returns a seralized sequence of Entry
// corresponding to the maps entries.
func (m *Map) Seq() seq.Sequence {
	return m.root.seq()
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

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows map to be called
// as a function by the 'dyn' library.
func (m *Map) Apply(args ...interface{}) interface{} {
	key := args[0]
	return m.At(key)
}

// Transform takes a set of actions and performs them
// on the persistent map. It does this by making a transient
// map and calling each action on it, then converting it back
// to a persistent map.
func (m *Map) Transform(actions ...func(*TMap) *TMap) *Map {
	out := m.AsTransient()
	for _, action := range actions {
		out = action(out)
	}
	return out.AsPersistent()
}

// TMap is a transient version of a map. Changes made to a transient
// map will not effect the original persistent structure. Changes to a
// transient map occur as mutations. These mutations are then made
// persistent when the transient is transformed into a persistent
// structure. These are useful when appling multiple transforms to a
// persistent map where the intermediate results will not be seen or
// stored anywhere.
type TMap struct {
	edit     *uint32
	hashSeed uintptr
	count    int
	root     node
}

// At returns the value associated with the key.
// If one is not found, nil is returned.
func (m *TMap) At(key interface{}) interface{} {
	m.ensureEditable()
	v, ok := m.root.find(0, hash.Any(key, m.hashSeed), key)
	if !ok {
		return nil
	}
	return v
}

// EntryAt returns the entry (key, value pair) of the key.
// If one is not found, nil is returned.
func (m *TMap) EntryAt(key interface{}) Entry {
	v, ok := m.root.find(0, hash.Any(key, m.hashSeed), key)
	if !ok {
		return nil
	}
	return entry{k: key, v: v}
}

// Assoc associates a value with a key in the map.
// The transient map is modified and then returned.
func (m *TMap) Assoc(key, value interface{}) *TMap {
	m.ensureEditable()
	root, added := m.root.assoc(m.edit, 0,
		hash.Any(key, m.hashSeed), key, value)
	if added {
		m.count++
	}
	m.root = root
	return m
}

// AsPersistent will transform this transient map into a persistent map.
// Once this occurs any additional actions on the transient map will fail.
func (m *TMap) AsPersistent() *Map {
	m.ensureEditable()
	atomic.StoreUint32(m.edit, 0)
	return &Map{
		hashSeed: m.hashSeed,
		count:    m.count,
		root:     m.root,
	}
}

// Contains will test if the key exists in the map.
func (m *TMap) Contains(key interface{}) bool {
	m.ensureEditable()
	_, ok := m.root.find(0, hash.Any(key, m.hashSeed), key)
	return ok
}

// Find will return the value for a key if it exists in the map and
// whether the key exists in the map. For non-nil values, exists will
// always be true.
func (m *TMap) Find(key interface{}) (value interface{}, exists bool) {
	return m.root.find(0, hash.Any(key, m.hashSeed), key)
}

// Delete removes a key and associated value from the map.
func (m *TMap) Delete(key interface{}) *TMap {
	m.ensureEditable()
	root, removed := m.root.without(m.edit, 0,
		hash.Any(key, m.hashSeed), key)
	if root == nil {
		root = emptySeededBitmapNode(m.hashSeed)
	}
	if removed {
		m.count--
	}
	m.root = root
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

// Length returns the number of entries in the map.
func (m *TMap) Length() int {
	return m.count
}

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows map to be called
// as a function by the 'dyn' library.
func (m *TMap) Apply(args ...interface{}) interface{} {
	key := args[0]
	return m.At(key)
}

func (m *TMap) ensureEditable() {
	if atomic.LoadUint32(m.edit) == 0 {
		panic(errTafterP)
	}
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
	fn := genRangeFunc(do)
	m.root.rnge(fn)
}

// String returns a string representation of the map.
func (m *TMap) String() string {
	var b strings.Builder
	fmt.Fprint(&b, "{ ")
	m.Range(func(entry Entry) {
		fmt.Fprintf(&b, "%s ", entry)
	})
	fmt.Fprint(&b, "}")
	return b.String()
}

type node interface {
	assoc(edit *uint32, shift uint, hash uintptr,
		k, v interface{}) (node, bool)
	without(edit *uint32, shift uint, hash uintptr,
		k interface{}) (node, bool)
	find(shift uint, hash uintptr, k interface{}) (interface{}, bool)
	seq() seq.Sequence
	rnge(func(Entry) bool) bool
}

type entry struct {
	k, v interface{}
}

func (e entry) Key() interface{} {
	return e.k
}

func (e entry) Value() interface{} {
	return e.v
}

func (e entry) String() string {
	return fmt.Sprintf("[%v %v]", e.k, e.v)
}

func (e entry) isLeaf() bool {
	return e.k != nil
}

func (e entry) matches(k interface{}) bool {
	return dyn.Equal(k, e.k)
}

type entries []entry

func (e entries) insert(idx int, ent entry) entries {
	if cap(e) >= len(e)+1 {
		// This accounts for the transient case where
		// we might pop elements but the whole backing
		// array still exists and may be larger than
		// the current slice.
		out := append(e, entry{})
		copy(out[idx+1:], e[idx:])
		out[idx] = ent
		return out
	}
	out := make([]entry, len(e)+1)
	copy(out, e[:idx])
	out[idx] = ent
	copy(out[idx+1:], e[idx:])
	return out
}

func (e entries) assoc(idx int, ent entry) entries {
	e[idx] = ent
	return e
}

func (e entries) append(ent entry) entries {
	if cap(e) >= len(e)+1 {
		return append(e, ent)
	}
	// We don't want Go's append semantics we just want to
	// increase by one entry at a time so we do it our selves
	out := make([]entry, len(e)+1)
	copy(out, e)
	out[len(e)] = ent
	return out
}

func (e entries) copy() entries {
	out := make([]entry, len(e))
	copy(out, e)
	return out
}

func (e entries) copyWithCap(cap int) entries {
	out := make([]entry, len(e), cap)
	copy(out, e)
	return out
}

func (e entries) remove(idx int) entries {
	out := e
	copy(out[idx:], out[idx+1:])
	out[len(out)-1] = entry{}
	out = out[:len(out)-1]
	return out
}

type entrySeq struct {
	es    entries
	index int
	s     seq.Sequence
}

func entrySeqNew(es entries, index int, s seq.Sequence) *entrySeq {
	if s != nil {
		return &entrySeq{
			es:    es,
			index: index,
			s:     s,
		}
	}
	for i := index; i < len(es); i++ {
		entry := es[i]
		if entry.isLeaf() {
			return &entrySeq{
				es:    es,
				index: i,
				s:     nil,
			}
		}
		if entry.v == nil {
			continue
		}
		n := entry.v.(node)
		if n == nil {
			continue
		}
		nodeSeq := n.seq()
		if nodeSeq == nil {
			continue
		}
		return &entrySeq{
			es:    es,
			index: i + 1,
			s:     nodeSeq,
		}
	}
	return nil
}

func (e *entrySeq) First() interface{} {
	if e.s != nil {
		return e.s.First()
	}
	return e.es[e.index]
}

func (e *entrySeq) Next() seq.Sequence {
	if e.s != nil {
		out := entrySeqNew(e.es, e.index, e.s.Next())
		if out == nil {
			return nil
		}
		return out
	}
	out := entrySeqNew(e.es, e.index+1, nil)
	if out == nil {
		return nil
	}
	return out
}

func (e *entrySeq) String() string {
	return seq.ConvertToString(e)
}

func mask(hash uintptr, shift uint) uint {
	return uint((hash >> shift) & maskValue)
}

func bitpos(hash uintptr, shift uint) uint32 {
	return 1 << mask(hash, shift)
}

func isEditable(nodeEdit, edit *uint32) bool {
	return atomic.LoadUint32(edit) == 1 && edit == nodeEdit
}

func atomicUint(i uint32) *uint32 {
	var atom = new(uint32)
	atomic.StoreUint32(atom, i)
	return atom
}

func atomicZero() *uint32 {
	return atomicUint(0)
}

func atomicOne() *uint32 {
	return atomicUint(1)
}
