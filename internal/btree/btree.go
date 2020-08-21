// Package btree implements a persistent B+Tree
package btree

import (
	"fmt"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/internal/atomic"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

const ErrTafterP = Error("transient used after persistent call")

type BTree struct {
	root    node
	count   int
	version int
	edit    *atomic.Bool

	cmp compareFunc
	eq  eqFunc
}

var emptyEdit = atomic.NewBool(false)

var empty = &BTree{
	root: newLeaf(0, emptyEdit),
	edit: emptyEdit,
	cmp:  dyn.Compare,
	eq:   dyn.Equal,
}

type btreeOptions struct {
	cmp compareFunc
	eq  eqFunc
}

type Option func(*btreeOptions)

func Compare(cmp func(k1, k2 interface{}) int) Option {
	return func(opts *btreeOptions) {
		opts.cmp = cmp
	}
}

func Equal(eq func(k1, k2 interface{}) bool) Option {
	return func(opts *btreeOptions) {
		opts.eq = eq
	}
}

func Empty(options ...Option) *BTree {
	if len(options) == 0 {
		return empty
	}

	opts := btreeOptions{
		cmp: dyn.Compare,
		eq:  dyn.Equal,
	}
	for _, option := range options {
		option(&opts)
	}

	return &BTree{
		root: newLeaf(0, emptyEdit),
		edit: emptyEdit,
		cmp:  opts.cmp,
		eq:   opts.eq,
	}
}

func (t *BTree) Contains(key interface{}) bool {
	_, found := t.root.find(key, t.cmp)
	return found
}

func (t *BTree) At(key interface{}) interface{} {
	out, _ := t.root.find(key, t.cmp)
	return out
}

func (t *BTree) Find(key interface{}) (interface{}, bool) {
	return t.root.find(key, t.cmp)
}

func (t *BTree) Add(key interface{}) *BTree {
	ret := t.root.add(key, t.cmp, t.eq, t.edit)
	var newRoot node
	switch ret.status {
	case returnUnchanged:
		return t
	case returnOne:
		newRoot = ret.nodes[0]
	case returnReplaced:
		return &BTree{
			root:    ret.nodes[0],
			count:   t.count,
			version: t.version + 1,
			edit:    t.edit,
			cmp:     t.cmp,
			eq:      t.eq,
		}
	default:
		nr := newNode(2, t.edit)
		nr.keys[0] = ret.nodes[0].maxKey()
		nr.keys[1] = ret.nodes[1].maxKey()
		copy(nr.children, ret.nodes[:])
		newRoot = nr
	}
	return &BTree{
		root:    newRoot,
		count:   t.count + 1,
		version: t.version + 1,
		edit:    t.edit,
		cmp:     t.cmp,
		eq:      t.eq,
	}
}

func (t *BTree) Delete(key interface{}) *BTree {
	ret := t.root.remove(key, nil, nil, t.cmp, t.edit)
	if ret.status == returnUnchanged {
		return t
	}
	newRoot := ret.nodes[1] // center
	if nr, ok := newRoot.(*internalNode); ok && nr.len == 1 {
		newRoot = nr.children[0]
	}
	return &BTree{
		root:    newRoot,
		count:   t.count - 1,
		version: t.version + 1,
		edit:    t.edit,
		cmp:     t.cmp,
		eq:      t.eq,
	}
}

func (t *BTree) Length() int {
	return t.count
}

func (t *BTree) String() string {
	var b strings.Builder
	t.root.string(&b, 1)
	return b.String()
}

func (t *BTree) Iterator() Iterator {
	i := makeIterator(t.root)
	i.HasNext() // Make sure the initial iterator value is valid
	return i
}

type Iterator struct {
	depth int
	stack [maxIterDepth]struct {
		n   node
		cur int
	}
}

func makeIterator(n node) Iterator {
	var i Iterator
	i.stack[0].n = n
	return i
}

func (i *Iterator) Next() interface{} {
	state := i.stack[i.depth]
	n := state.n.(*leafNode)
	out := n.keys[state.cur]
	i.stack[i.depth].cur++
	return out
}

func (i *Iterator) HasNext() bool {
	state := i.stack[i.depth]
	switch n := state.n.(type) {
	case *leafNode:
		if state.cur < n.len {
			return true
		}
		if i.depth == 0 {
			return false
		}
		i.popNode()
		return i.HasNext()
	case *internalNode:
		if state.cur < n.len {
			child := n.children[state.cur]
			i.stack[i.depth].cur++
			i.pushNode(child)
			switch child.(type) {
			case *leafNode:
				return true
			case *internalNode:
				return i.HasNext()
			}
		}
		if i.depth == 0 {
			return false
		}
		i.popNode()
		return i.HasNext()
	default:
		return false
	}
}

func (i *Iterator) pushNode(n node) {
	i.depth = i.depth + 1
	state := i.stack[i.depth]
	state.n = n
	state.cur = 0
	i.stack[i.depth] = state
}

func (i *Iterator) popNode() {
	state := i.stack[i.depth]
	state.n = nil
	state.cur = 0
	i.stack[i.depth] = state
	i.depth = i.depth - 1
}

type TBTree struct {
	root    node
	count   int
	version int
	edit    *atomic.Bool

	cmp compareFunc
	eq  eqFunc
}

func (t *BTree) AsTransient() *TBTree {
	return &TBTree{
		root:    t.root,
		count:   t.count,
		version: t.version,
		edit:    atomic.NewBool(true),
		cmp:     t.cmp,
		eq:      t.eq,
	}
}

func (t *TBTree) Contains(key interface{}) bool {
	t.ensureEditable()
	_, found := t.root.find(key, t.cmp)
	return found
}

func (t *TBTree) At(key interface{}) interface{} {
	t.ensureEditable()
	out, _ := t.root.find(key, t.cmp)
	return out
}

func (t *TBTree) Find(key interface{}) (interface{}, bool) {
	t.ensureEditable()
	return t.root.find(key, t.cmp)
}

func (t *TBTree) Add(key interface{}) *TBTree {
	t.ensureEditable()
	ret := t.root.add(key, t.cmp, t.eq, t.edit)
	switch ret.status {
	case returnUnchanged:
		return t
	case returnEarly:
	case returnReplaced:
		t.root = ret.nodes[0]
		t.version++
		return t
	case returnOne:
		t.root = ret.nodes[0]
	default:
		nr := newNode(2, t.edit)
		nr.keys[0] = ret.nodes[0].maxKey()
		nr.keys[1] = ret.nodes[1].maxKey()
		copy(nr.children, ret.nodes[:])
		t.root = nr
	}
	t.count++
	t.version++
	return t
}

func (t *TBTree) Delete(key interface{}) *TBTree {
	t.ensureEditable()
	ret := t.root.remove(key, nil, nil, t.cmp, t.edit)
	switch ret.status {
	case returnUnchanged:
		return t
	case returnEarly:
	default:
		newRoot := ret.nodes[1] // center
		if nr, ok := newRoot.(*internalNode); ok && nr.len == 1 {
			newRoot = nr.children[0]
		}
		t.root = newRoot
	}
	t.count--
	t.version++
	return t
}

func (t *TBTree) Iterator() Iterator {
	t.ensureEditable()
	i := makeIterator(t.root)
	i.HasNext() // Make sure the initial iterator value is valid
	return i
}

func (t *TBTree) Length() int {
	t.ensureEditable()
	return t.count
}

func (t *TBTree) String() string {
	var b strings.Builder
	t.root.string(&b, 1)
	return b.String()
}

func (t *TBTree) AsPersistent() *BTree {
	t.ensureEditable()
	t.edit.Reset(false)
	return &BTree{
		root:    t.root,
		count:   t.count,
		version: t.version,
		edit:    t.edit,
		cmp:     t.cmp,
		eq:      t.eq,
	}
}

func (t *TBTree) ensureEditable() {
	if !t.edit.Deref() {
		panic(ErrTafterP)
	}
}

type compareFunc func(k1, k2 interface{}) int
type eqFunc func(k1, k2 interface{}) bool

const (
	maxLen    = 64
	minLen    = maxLen >> 1
	expandLen = 8
	// maxIterDepth is log_32(^uintptr(0)) rounded up -- 13.
	// The height is calculated as h <= log_32((n+1)/2). The
	// maximum height must therefore be smaller than
	// log_32(^uintptr(0)) rounded up to the next value. To
	// calculate this we use log_2(^uintptr(0))/log_2(32). Which
	// is of course 64/5 = 12.8.  We round 64 up to get an even
	// 13.
	maxIterDepth = (64 + 1) / 5
)

type node interface {
	search(key interface{}, cmp compareFunc) int
	find(key interface{}, cmp compareFunc) (interface{}, bool)
	add(key interface{}, cmp compareFunc, eq eqFunc, edit *atomic.Bool) nodeReturn
	remove(key interface{}, left, right node, cmp compareFunc, edit *atomic.Bool) nodeReturn
	leafPart() *leafNode
	maxKey() interface{}
	string(b *strings.Builder, lvl int)
}

type returnStatus uint8

const (
	returnUnchanged returnStatus = iota
	returnEarly
	returnReplaced
	returnOne
	returnTwo
	returnThree
)

var returnStatusStrings = [...]string{
	returnUnchanged: "unchanged",
	returnEarly:     "early",
	returnReplaced:  "replaced",
	returnOne:       "one",
	returnTwo:       "two",
	returnThree:     "three",
}

func (s returnStatus) String() string {
	return returnStatusStrings[s]
}

type nodeReturn struct {
	status returnStatus
	nodes  [3]node
}

func (r nodeReturn) String() string {
	return fmt.Sprintf("{ %s %v %v %v }",
		r.status, r.nodes[0], r.nodes[1], r.nodes[2])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
