// Package vector implements a Radix Balanced trie based vector.
package vector // import "jsouthworth.net/go/immutable/vector"

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/seq"
)

var errOutOfBounds = errors.New("out of bounds")
var errEmptyVector = errors.New("empty vector")
var errTafterP = errors.New("transient used after persistent call")
var errRangeSig = errors.New("Range requires a function: func(v vT) bool or func(v vT)")
var errReduceSig = errors.New("Reduce requires a function: func(init iT, v vT) oT")

const (
	bits  = 5
	width = 1 << bits
	mask  = width - 1
)

// Vector is a persistent immutable vector.
// Operations on this structure will return
// modified copies of the original vector sharing
// much of the structure with the original.
type Vector struct {
	count int
	shift uint
	root  *vnode
	tail  *array
}

var emptyNode = vnodeNew(atomicZero())

var empty = Vector{
	count: 0,
	shift: bits,
	root:  emptyNode,
	tail:  new(array),
}

// Empty returns the empty vector
func Empty() *Vector {
	return &empty
}

// New converts as list of elements to a persistent vector.
func New(elems ...interface{}) *Vector {
	v := Empty().AsTransient()
	for _, elem := range elems {
		v = v.Append(elem)
	}
	return v.AsPersistent()
}

// From will convert many go types to an immutable vector.
// Converting some types is more efficient than others and the
// mechanisms are described below.
//
// *Vector:
//    Returned directly as it is already immutable.
// *TVector:
//    AsPersistent is called on the value and the result returned.
// []interface{}:
//    New is called with the elements.
// seq.Sequable:
//    Seq is called on the value and the vector is built from the resulting sequence.
// seq.Sequence:
//    The vector is built from the sequence. Care should be taken to provide finite sequences or the vector will grow without bound.
// []T:
//    The slice is converted to a vector using reflection.
func From(value interface{}) *Vector {
	switch v := value.(type) {
	case *Vector:
		return v
	case *TVector:
		return v.AsPersistent()
	case []interface{}:
		return New(v...)
	case seq.Seqable:
		return vectorFromSequence(v.Seq())
	case seq.Sequence:
		return vectorFromSequence(v)
	default:
		return vectorFromReflection(value)
	}
}

func vectorFromSequence(coll seq.Sequence) *Vector {
	if coll == nil {
		return Empty()
	}
	return seq.Reduce(func(result *Vector, input interface{}) *Vector {
		return result.Append(input)
	}, Empty(), coll).(*Vector)
}

func vectorFromReflection(value interface{}) *Vector {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Slice:
		out := Empty().AsTransient()
		for i := 0; i < v.Len(); i++ {
			out = out.Append(v.Index(i).Interface())
		}
		return out.AsPersistent()
	default:
		return Empty()
	}
}

// At returns the element at the supplied index. It will panic if out of bounds.
func (v *Vector) At(i int) interface{} {
	arr := v.arrayFor(i)
	return arr[i&mask]
}

// Find returns the value at the supplied index and if that index was
// in bounds for the vector. Out of bounds access does not panic but
// returns (nil, false). idx must be an int.
func (v *Vector) Find(idx interface{}) (interface{}, bool) {
	i := idx.(int)
	if i < 0 || i >= v.Length() {
		return nil, false
	}
	return v.At(i), true
}

// Assoc associates the value with the index in an immutable copy of the vector
// sharing structure with the original vector.
func (v *Vector) Assoc(i int, value interface{}) *Vector {
	switch {
	case i < 0 || i >= v.count:
		panic(errOutOfBounds)
	case i >= v.tailOffset():
		return &Vector{
			count: v.count,
			shift: v.shift,
			root:  v.root,
			tail:  v.tail.copy().assoc(i&mask, value),
		}
	default:
		return &Vector{
			count: v.count,
			shift: v.shift,
			root:  v.doAssoc(v.shift, v.root, i, value),
			tail:  v.tail,
		}
	}
}

// Append will extend the vector and associates the value with new last
// element. This will return a new copy of the immutable vector sharing
// structure with the original vector.
func (v *Vector) Append(value interface{}) *Vector {
	if v == nil {
		return Empty().Append(value)
	}
	switch {
	case v.roomInTail():
		return &Vector{
			count: v.count + 1,
			shift: v.shift,
			root:  v.root,
			tail: v.tail.copy().
				assoc(v.count-v.tailOffset(), value),
		}
	case v.overflowsRoot():
		root := vnodeNew(v.root.edit)
		root.array[0] = v.root
		root.array[1] = newPath(v.root.edit, v.shift,
			vnodeNewFromArray(v.root.edit, v.tail))
		return &Vector{
			count: v.count + 1,
			shift: v.shift + bits,
			root:  root,
			tail:  new(array).assoc(0, value),
		}
	default:
		return &Vector{
			count: v.count + 1,
			shift: v.shift,
			root: v.pushTail(v.shift, v.root,
				vnodeNewFromArray(v.root.edit, v.tail)),
			tail: new(array).assoc(0, value),
		}
	}
}

// Conj will extend the vector and associates the value with new last
// element. Conj implements a generic mechanism for building collections.
func (v *Vector) Conj(elem interface{}) interface{} {
	return v.Append(elem)
}

// Delete removes the element at the current index, shifting the others
// down and yeilding a vector with one fewer elements.
func (v *Vector) Delete(idx int) *Vector {
	return v.Transform(func(t *TVector) *TVector {
		return t.Delete(idx)
	})
}

// Insert adds the value to the vector at the provided index shifting the
// other values down. This yeilds a vector with an additional value at the
// provided index.
func (v *Vector) Insert(idx int, val interface{}) *Vector {
	return v.Transform(func(t *TVector) *TVector {
		return t.Insert(idx, val)
	})
}

// Equal compares each value of the vector to determine if the vector is
// equal to the one passed in.
func (v *Vector) Equal(o interface{}) bool {
	other, ok := o.(*Vector)
	if !ok {
		return false
	}
	if v.Length() != other.Length() {
		return false
	}
	for i := 0; i < v.Length(); i++ {
		val := v.At(i)
		if !dyn.Equal(other.At(i), val) {
			return false
		}
	}
	return true
}

// Pop removes the last element of the vector,
// returning an immutable copy of the vector
// with one less element, sharing structure with
// the original vector.
func (v *Vector) Pop() *Vector {
	switch {
	case v.count == 0:
		panic(errEmptyVector)
	case v.count == 1:
		return Empty()
	case v.count-v.tailOffset() > 1:
		newtail := arrayNewFromSlice(
			v.tail[:(v.count-v.tailOffset())-1])
		return &Vector{
			count: v.count - 1,
			shift: v.shift,
			root:  v.root,
			tail:  newtail,
		}
	default:
		newTail := v.arrayFor(v.count - 2)
		newRoot := v.popTail(v.shift, v.root)
		if newRoot == nil {
			newRoot = emptyNode
		}
		if v.shift > bits && newRoot.array[1] == nil {
			root := emptyNode
			if newRoot.array[0] != nil {
				root = newRoot.array[0].(*vnode)
			}
			return &Vector{
				count: v.count - 1,
				shift: v.shift - bits,
				root:  root,
				tail:  newTail,
			}
		}
		return &Vector{
			count: v.count - 1,
			shift: v.shift,
			root:  newRoot,
			tail:  newTail,
		}
	}
}

// Length returns the number of elements in the vector.
func (v *Vector) Length() int {
	if v == nil {
		return 0
	}
	return v.count
}

// AsTransient will return a mutable version of the
// vector that may be used to perform mutations in
// a controlled way.
func (v *Vector) AsTransient() *TVector {
	return &TVector{
		count: v.count,
		shift: v.shift,
		root:  v.root.editable(),
		tail:  v.tail.copy(),
	}
}

// AsNative will traverse the vector and return a
// go native representation of the values contained within.
func (v *Vector) AsNative() []interface{} {
	out := make([]interface{}, v.Length())
	for i := 0; i < v.Length(); i++ {
		val := v.At(i)
		out[i] = val
	}
	return out
}

// String coverts the vector to a string representation.
func (v *Vector) String() string {
	return vectorString(v)
}

// Seq returns a seq.Sequence that will traverse the vector.
func (v *Vector) Seq() seq.Sequence {
	if v.Length() == 0 {
		return nil
	}
	return &vectorSequence{
		vec: v,
	}
}

// Slice returns a Slice structure that has the semantics of go slices
// over the immutable vector.
func (v *Vector) Slice(start, end int) *Slice {
	if start < 0 || end > v.Length() {
		panic(errOutOfBounds)
	}
	return &Slice{
		vector: v,
		start:  start,
		end:    end,
	}
}

// Range calls the passed in function on each element of the vector.
// The function passed in may be of many types:
//
// func(index int, value interface{}) bool:
//    Takes the index and a value of any type and returns if the loop should continue.
//    Useful to avoid reflection where not needed and to support
//    heterogenous vectors.
// func(index int, value interface{})
//    Takes the index and a value of any type.
//    Useful to avoid reflection where not needed and to support
//    heterogenous vectors.
// func(index int, value T) bool:
//    Takes the index and a value of the type of element stored in the vector and
//    returns if the loop should continue. Useful for homogeneous vectors.
//    Is called with reflection and will panic if the type is incorrect.
// func(index int, value T)
//    Takes the index and a value of the type of element stored in the vector and
//    returns if the loop should continue. Useful for homogeneous vectors.
//    Is called with reflection and will panic if the type is incorrect.
// Range will panic if passed anything that doesn't match one of these signatures
func (v *Vector) Range(do interface{}) {
	cont := true
	fn := genRangeFunc(do)
	for i := 0; i < v.Length() && cont; i++ {
		value := v.At(i)
		cont = fn(i, value)
	}
}

func genRangeFunc(do interface{}) func(int, interface{}) bool {
	switch fn := do.(type) {
	case func(idx int, value interface{}) bool:
		return fn
	case func(idx int, value interface{}):
		return func(idx int, value interface{}) bool {
			fn(idx, value)
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
		return func(idx int, value interface{}) bool {
			out := dyn.Apply(do, idx, value)
			if out != nil {
				return out.(bool)
			}
			return true
		}
	}
}

// Reduce is a fast mechanism for reducing a Vector. Reduce can take
// the following types as the fn:
//
// func(init interface{}, value interface{}) interface{}
// func(init iT, v vT) oT
//
// Reduce will panic if given any other function type.
func (v *Vector) Reduce(fn interface{}, init interface{}) interface{} {
	res := init
	rFn := genReduceFunc(fn)
	v.Range(func(_ int, e interface{}) {
		res = rFn(res, e)
	})
	return res
}

func genReduceFunc(fn interface{}) func(r, v interface{}) interface{} {
	switch f := fn.(type) {
	case func(res, val interface{}) interface{}:
		return func(r, v interface{}) interface{} {
			return f(r, v)
		}
	default:
		rv := reflect.ValueOf(fn)
		if rv.Kind() != reflect.Func {
			panic(errReduceSig)
		}
		rt := rv.Type()
		if rt.NumIn() != 2 {
			panic(errReduceSig)
		}
		if rt.NumOut() != 1 {
			panic(errReduceSig)
		}
		return func(r, v interface{}) interface{} {
			return dyn.Apply(f, r, v)
		}
	}
}

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows vector to be called
// as a function by the 'dyn' library.
func (v *Vector) Apply(args ...interface{}) interface{} {
	idx := args[0].(int)
	return v.At(idx)
}

// Transform takes a set of actions and performs them
// on the persistent vector. It does this by making a transient
// vector and calling each action on it, then converting it back
// to a persistent vector.
func (v *Vector) Transform(actions ...func(*TVector) *TVector) *Vector {
	out := v.AsTransient()
	for _, action := range actions {
		out = action(out)
	}
	return out.AsPersistent()
}

func (v *Vector) tailOffset() int {
	if v.count < width {
		return 0
	}
	return ((v.count - 1) >> bits) << bits
}

func (v *Vector) arrayFor(i int) *array {
	switch {
	case i < 0 || i >= v.count:
		panic(errOutOfBounds)
	case i >= v.tailOffset():
		return v.tail
	default:
		n := v.root
		for level := v.shift; level > 0; level -= bits {
			n = n.array[(i>>level)&mask].(*vnode)
		}
		return n.array
	}
}

func (v *Vector) roomInTail() bool {
	return (v.count - v.tailOffset()) < width
}

func (v *Vector) overflowsRoot() bool {
	return (v.count >> bits) > (1 << v.shift)
}

func (v *Vector) pushTail(
	level uint,
	parent *vnode,
	tailnode *vnode,
) *vnode {
	subidx := ((v.count - 1) >> level) & mask
	ret := parent.clone()
	if isLeaf(level) {
		//leaf of trie, insert the node passed in
		ret.array[subidx] = tailnode
	} else {
		child := parent.array[subidx]
		if child != nil {
			//maps to existing trie, keep walking
			ret.array[subidx] =
				v.pushTail(level-bits, child.(*vnode), tailnode)
		} else {
			//no child found, allocate a new path
			ret.array[subidx] =
				newPath(v.root.edit, level-bits, tailnode)
		}
	}
	return ret
}

func (v *Vector) popTail(level uint, n *vnode) *vnode {
	subidx := ((v.count - 2) >> level) & mask
	switch {
	case level > bits:
		newChild := v.popTail(level-bits, n.array[subidx].(*vnode))
		if newChild == nil && subidx == 0 {
			return nil
		}
		ret := vnodeNewFromArray(v.root.edit, n.array.copy())
		if newChild == nil {
			ret.array[subidx] = nil
		} else {
			ret.array[subidx] = newChild
		}
		return ret
	case subidx == 0:
		return nil
	default:
		ret := vnodeNewFromArray(v.root.edit, n.array.copy())
		ret.array[subidx] = nil
		return ret
	}
}

func (v *Vector) doAssoc(
	level uint,
	n *vnode,
	i int,
	value interface{},
) *vnode {
	ret := n.clone()
	if level == 0 {
		ret.array[i&mask] = value
	} else {
		subidx := (i >> level) & mask
		ret.array[subidx] =
			v.doAssoc(level-bits,
				n.array[subidx].(*vnode), i, value)
	}
	return ret
}

// TVector is a transient version of a Vector. Changes made to a
// transient vector will not effect the original persistent
// structure. Changes occur as mutation of the transient. The changes
// made will become immutable when AsPersistent is called. This structure
// is useful when making mulitple modifications to a persistent vector
// where the intermediate results will not be seen or stored anywhere.
type TVector struct {
	count int
	shift uint
	root  *vnode
	tail  *array
}

// At returns the element at the supplied index.
// It will panic if out of bounds or called after AsPersistent.
func (v *TVector) At(i int) interface{} {
	v.ensureEditable()
	node := v.arrayFor(i)
	return node[i&mask]
}

// Find returns the value at the supplied index and if that index was
// in bounds for the vector. Out of bounds access does not panic but
// returns (nil, false). idx must be an int.
func (v *TVector) Find(idx interface{}) (interface{}, bool) {
	i := idx.(int)
	if i < 0 || i >= v.Length() {
		return nil, false
	}
	return v.At(i), true
}

// Assoc associates the value with the index.
// It will panic if called after AsPersistent.
func (v *TVector) Assoc(i int, value interface{}) *TVector {
	v.ensureEditable()
	switch {
	case i < 0 || i >= v.count:
		panic(errOutOfBounds)
	case i >= v.tailOffset():
		v.tail[i&mask] = value
		return v
	default:
		v.root = v.doAssoc(v.shift, v.root, i, value)
		return v
	}
}

// Append will extend the vector and associates the value with new last
// element. It will panic if called after AsPersistent.
func (v *TVector) Append(value interface{}) *TVector {
	v.ensureEditable()
	switch {
	case v.roomInTail():
		v.tail.assoc(v.count&mask, value)
	case v.overflowsRoot():
		newroot := vnodeNew(v.root.edit)
		newroot.array[0] = v.root
		newroot.array[1] = newPath(v.root.edit, v.shift,
			vnodeNewFromArray(v.root.edit, v.tail))
		v.root = newroot
		v.shift = v.shift + bits
		v.tail = new(array).assoc(0, value)
	default:
		v.root = v.pushTail(v.shift, v.root,
			vnodeNewFromArray(v.root.edit, v.tail))
		v.tail = new(array).assoc(0, value)
	}

	v.count = v.count + 1
	return v
}

// Conj will extend the vector and associates the value with new last
// element. Conj implements a generic mechanism for building collections.
func (v *TVector) Conj(elem interface{}) interface{} {
	return v.Append(elem)
}

// Pop removes the last element of the vector.
// It will panic if called after AsPersistent.
func (v *TVector) Pop() *TVector {
	v.ensureEditable()
	switch {
	case v.count == 0:
		panic(errEmptyVector)
	case v.count == 1:
		v.count--
		return v
	case ((v.count - 1) & mask) > 0:
		v.count--
		return v
	default:
		newTail := v.editableArrayFor(v.count - 2)
		newRoot := v.popTail(v.shift, v.root)
		newShift := v.shift
		if newRoot == nil {
			newRoot = vnodeNew(v.root.edit)
		}
		if v.shift > bits && newRoot.array[1] == nil {
			if newRoot.array[0] != nil {
				newRoot = v.ensureEditableNode(
					newRoot.array[0].(*vnode))
			} else {
				newRoot = vnodeNew(v.root.edit)
			}
			newShift = newShift - bits
		}
		v.root = newRoot
		v.shift = newShift
		v.count = v.count - 1
		v.tail = newTail
		return v
	}
}

// Length returns the number of elements in the vector.
func (v *TVector) Length() int {
	return v.count
}

// AsPersistent will transform this transient vector into a persistent vector.
// Once this occurs any additional actions on the transient vector will panic.
func (v *TVector) AsPersistent() *Vector {
	v.ensureEditable()
	atomic.StoreInt32(v.root.edit, 0)
	trimmedTail := arrayNewFromSlice(v.tail[:v.count-v.tailOffset()])
	return &Vector{
		count: v.count,
		shift: v.shift,
		root:  v.root,
		tail:  trimmedTail,
	}
}

// String coverts the vector to a string representation.
func (v *TVector) String() string {
	return vectorString(v)
}

// Range calls the passed in function on each element of the vector.
// The function passed in may be of many types:
//
// func(index int, value interface{}) bool:
//    Takes the index and a value of any type and returns if the loop should continue.
//    Useful to avoid reflection where not needed and to support
//    heterogenous vectors.
// func(index int, value interface{})
//    Takes the index and a value of any type.
//    Useful to avoid reflection where not needed and to support
//    heterogenous vectors.
// func(index int, value T) bool:
//    Takes the index and a value of the type of element stored in the vector and
//    returns if the loop should continue. Useful for homogeneous vectors.
//    Is called with reflection and will panic if the type is incorrect.
// func(index int, value T)
//    Takes the index and a value of the type of element stored in the vector and
//    returns if the loop should continue. Useful for homogeneous vectors.
//    Is called with reflection and will panic if the type is incorrect.
// Range will panic if passed anything that doesn't match one of these signatures
func (v *TVector) Range(do interface{}) {
	cont := true
	fn := genRangeFunc(do)
	for i := 0; i < v.Length() && cont; i++ {
		value := v.At(i)
		cont = fn(i, value)
	}
}

// Reduce is a fast mechanism for reducing a Vector. Reduce can take
// the following types as the fn:
//
// func(init interface{}, value interface{}) interface{}
// func(init iT, v vT) oT
//
// Reduce will panic if given any other function type.
func (v *TVector) Reduce(fn interface{}, init interface{}) interface{} {
	res := init
	rFn := genReduceFunc(fn)
	v.Range(func(_ int, e interface{}) {
		res = rFn(res, e)
	})
	return res
}

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows vector to be called
// as a function by the 'dyn' library.
func (v *TVector) Apply(args ...interface{}) interface{} {
	idx := args[0].(int)
	return v.At(idx)
}

// Delete removes the element at the current index, shifting the others
// down and yeilding a vector with one fewer elements.
func (v *TVector) Delete(idx int) *TVector {
	if idx < 0 || idx >= v.count {
		panic(errOutOfBounds)
	}

	for i := idx; i < v.Length()-1; i++ {
		v = v.Assoc(i, v.At(i+1))
	}
	return v.Pop()

}

// Insert adds the value to the vector at the provided index shifting the
// other values down. This yeilds a vector with an additional value at the
// provided index.
func (v *TVector) Insert(idx int, val interface{}) *TVector {
	if idx < 0 || idx >= v.count {
		panic(errOutOfBounds)
	}
	v = v.Append(nil)
	for i := v.Length() - 1; i > idx; i-- {
		v = v.Assoc(i, v.At(i-1))
	}
	return v.Assoc(idx, val)
}

func (v *TVector) roomInTail() bool {
	return (v.count - v.tailOffset()) < width
}

func (v *TVector) overflowsRoot() bool {
	return (v.count >> bits) > (1 << v.shift)
}

func (v *TVector) pushTail(
	level uint,
	parent *vnode,
	tailnode *vnode,
) *vnode {
	subidx := ((v.count - 1) >> level) & mask
	ret := v.ensureEditableNode(parent)
	var nodeToInsert *vnode
	if isLeaf(level) {
		nodeToInsert = tailnode
	} else {
		child := parent.array[subidx]
		if child != nil {
			nodeToInsert =
				v.pushTail(level-bits, child.(*vnode), tailnode)
		} else {
			nodeToInsert =
				newPath(v.root.edit, level-bits, tailnode)
		}
	}
	ret.array[subidx] = nodeToInsert
	return ret
}

func (v *TVector) popTail(level uint, n *vnode) *vnode {
	n = v.ensureEditableNode(n)
	subidx := ((v.count - 2) >> level) & mask
	switch {
	case level > bits:
		newChild := v.popTail(level-bits, n.array[subidx].(*vnode))
		if newChild == nil && subidx == 0 {
			return nil
		}
		if newChild == nil {
			n.array[subidx] = nil
		} else {
			n.array[subidx] = newChild
		}
		return n
	case subidx == 0:
		return nil
	default:
		n.array[subidx] = nil
		return n
	}
}

func (v *TVector) arrayFor(i int) *array {
	switch {
	case i < 0 || i >= v.count:
		panic(errOutOfBounds)
	case i >= v.tailOffset():
		return v.tail
	default:
		n := v.root
		for level := v.shift; level > 0; level -= bits {
			n = n.array[(i>>level)&mask].(*vnode)
		}
		return n.array
	}
}

func (v *TVector) editableArrayFor(i int) *array {
	switch {
	case i < 0 || i >= v.count:
		panic(errOutOfBounds)
	case i >= v.tailOffset():
		return v.tail
	default:
		n := v.root
		for level := v.shift; level > 0; level -= bits {
			n = v.ensureEditableNode(
				n.array[(i>>level)&mask].(*vnode))
		}
		return n.array
	}
}

func (v *TVector) tailOffset() int {
	if v.count < width {
		return 0
	}
	return ((v.count - 1) >> bits) << bits
}

func (v *TVector) ensureEditable() {
	if atomic.LoadInt32(v.root.edit) == 0 {
		panic(errTafterP)
	}
}

func (v *TVector) ensureEditableNode(node *vnode) *vnode {
	if node.edit == v.root.edit {
		return node
	}
	return node.editable()
}

func (v *TVector) doAssoc(
	level uint,
	n *vnode,
	i int,
	value interface{},
) *vnode {
	ret := v.ensureEditableNode(n)
	if level == 0 {
		ret.array[i&mask] = value
	} else {
		subidx := (i >> level) & mask
		ret.array[subidx] =
			v.doAssoc(level-bits,
				n.array[subidx].(*vnode), i, value)
	}
	return ret
}

type vnode struct {
	array *array
	edit  *int32
}

func (n *vnode) clone() *vnode {
	return &vnode{
		edit:  n.edit,
		array: n.array.copy(),
	}
}

func (n *vnode) editable() *vnode {
	tmp := n.clone()
	tmp.edit = atomicOne()
	return tmp
}

func vnodeNew(edit *int32) *vnode {
	return &vnode{edit: edit, array: new(array)}
}

func vnodeNewFromArray(edit *int32, a *array) *vnode {
	n := &vnode{edit: edit, array: a}
	return n
}

type array [width]interface{}

func (a *array) copy() *array {
	var tmp array
	copy(tmp[:], a[:])
	return &tmp
}

func (a *array) assoc(i int, v interface{}) *array {
	a[i] = v
	return a
}

func arrayNewFromSlice(in []interface{}) *array {
	var out array
	copy(out[:], in)
	return &out
}

func isLeaf(level uint) bool {
	return level == 5
}

func newPath(edit *int32, level uint, node *vnode) *vnode {
	if level == 0 {
		return node
	}
	ret := vnodeNew(edit)
	ret.array[0] = newPath(edit, level-bits, node)
	return ret
}

func atomicInt(i int32) *int32 {
	var atom = new(int32)
	atomic.StoreInt32(atom, i)
	return atom
}

func atomicZero() *int32 {
	return atomicInt(0)
}

func atomicOne() *int32 {
	return atomicInt(1)
}

type vectorSequence struct {
	vec interface {
		At(int) interface{}
		Length() int
	}
	idx int
}

func (seq *vectorSequence) First() interface{} {
	return seq.vec.At(seq.idx)
}

func (seq *vectorSequence) Next() seq.Sequence {
	if seq.idx+1 == seq.vec.Length() {
		return nil
	}
	return &vectorSequence{
		vec: seq.vec,
		idx: seq.idx + 1,
	}
}

func (s *vectorSequence) String() string {
	return seq.ConvertToString(s)
}

// Slice is a view of an underlying persistent vector.
// For the most part a Slice shares semantics with a go slice,
// except that changes do not modify the underlying vector;
// instead returning a view of a new persistent vector that
// shares structure with the original vector.
type Slice struct {
	vector     *Vector
	start, end int
}

// At returns the element at the supplied index. It will panic if out of bounds.
func (s *Slice) At(i int) interface{} {
	if (s.start+i >= s.end) || (i < 0) {
		panic(errOutOfBounds)
	}
	return s.vector.At(s.start + i)
}

// Find returns the value at the supplied index and if that index was
// in bounds for the vector. Out of bounds access does not panic but
// returns (nil, false). idx must be an int.
func (s *Slice) Find(idx interface{}) (interface{}, bool) {
	i := idx.(int)
	if i < 0 || i >= s.Length() {
		return nil, false
	}
	return s.At(i), true
}

// Append will extend the vector and associates the value with new last
// element. This will return a new copy of the immutable vector sharing
// structure with the original vector.
func (s *Slice) Append(v interface{}) *Slice {
	if s.end == s.vector.Length() {
		return &Slice{
			vector: s.vector.Append(v),
			start:  s.start,
			end:    s.end + 1,
		}
	}
	return &Slice{
		vector: s.vector.Assoc(s.end, v),
		start:  s.start,
		end:    s.end + 1,
	}
}

// Conj will extend the vector and associates the value with new last
// element. Conj implements a generic mechanism for building collections.
func (s *Slice) Conj(elem interface{}) interface{} {
	return s.Append(elem)
}

// Assoc associates the value with the index in an immutable copy of the vector
// sharing structure with the original vector.
func (s *Slice) Assoc(i int, v interface{}) *Slice {
	if (s.start+i >= s.end) || (i < 0) {
		panic(errOutOfBounds)
	}
	return &Slice{
		vector: s.vector.Assoc(s.start+i, v),
		start:  s.start,
		end:    s.end,
	}
}

// Length returns the number of elements in the vector.
func (s *Slice) Length() int {
	return s.end - s.start
}

// Slice will further limit the view of this slice.
func (s *Slice) Slice(start, end int) *Slice {
	newEnd := s.start + start + (end - start)
	if start < 0 || newEnd > s.end {
		panic(errOutOfBounds)
	}
	return &Slice{
		vector: s.vector,
		start:  s.start + start,
		end:    newEnd,
	}
}

// Seq returns a seq.Sequence that will traverse the vector.
func (s *Slice) Seq() seq.Sequence {
	if s.Length() == 0 {
		return nil
	}
	return &vectorSequence{
		vec: s,
	}
}

// Equal compares each value of the slice to determine if the slice is
// equal to the one passed in.
func (s *Slice) Equal(o interface{}) bool {
	other, ok := o.(*Slice)
	if !ok {
		return false
	}
	if s.Length() != other.Length() {
		return false
	}
	for i := 0; i < s.Length(); i++ {
		val := s.At(i)
		if !dyn.Equal(other.At(i), val) {
			return false
		}
	}
	return true
}

// String coverts the vector to a string representation.
func (s *Slice) String() string {
	return vectorString(s)
}

// Range calls the passed in function on each element of the slice.
// The function passed in may be of many types:
//
// func(index int, value interface{}) bool:
//    Takes the index and a value of any type and returns if the loop should continue.
//    Useful to avoid reflection where not needed and to support
//    heterogenous slices.
// func(index int, value interface{})
//    Takes the index and a value of any type.
//    Useful to avoid reflection where not needed and to support
//    heterogenous slices.
// func(index int, value T) bool:
//    Takes the index and a value of the type of element stored in the slice and
//    returns if the loop should continue. Useful for homogeneous slices.
//    Is called with reflection and will panic if the type is incorrect.
// func(index int, value T)
//    Takes the index and a value of the type of element stored in the slice and
//    returns if the loop should continue. Useful for homogeneous slices.
//    Is called with reflection and will panic if the type is incorrect.
// Range will panic if passed anything that doesn't match one of these signatures
func (s *Slice) Range(do interface{}) {
	cont := true
	fn := genRangeFunc(do)
	for i := 0; i < s.Length() && cont; i++ {
		value := s.At(i)
		cont = fn(i, value)
	}
}

// Reduce is a fast mechanism for reducing a Vector. Reduce can take
// the following types as the fn:
//
// func(init interface{}, value interface{}) interface{}
// func(init iT, v vT) oT
//
// Reduce will panic if given any other function type.
func (s *Slice) Reduce(fn interface{}, init interface{}) interface{} {
	res := init
	rFn := genReduceFunc(fn)
	s.Range(func(_ int, e interface{}) {
		res = rFn(res, e)
	})
	return res
}

// Apply takes an arbitrary number of arguments and returns the
// value At the first argument.  Apply allows a slice to be called
// as a function by the 'dyn' library.
func (s *Slice) Apply(args ...interface{}) interface{} {
	idx := args[0].(int)
	return s.At(idx)
}

func vectorString(v interface {
	At(int) interface{}
	Length() int
}) string {
	buf := new(bytes.Buffer)
	fmt.Fprint(buf, "[")
	if v.Length() != 0 {
		fmt.Fprint(buf, v.At(0))
	}
	for i := 1; i < v.Length(); i++ {
		fmt.Fprintf(buf, " %v", v.At(i))
	}
	fmt.Fprint(buf, "]")
	return buf.String()
}
