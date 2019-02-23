package hashmap

import (
	"math/bits"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/hash"
	"jsouthworth.net/go/seq"
)

const bitmapCap = width / 2

func emptySeededBitmapNode(seed uintptr) *bitmapIndexedNode {
	return &bitmapIndexedNode{
		edit: zero,
		seed: seed,
	}
}

type bitmapIndexedNode struct {
	bitmap uint32
	seed   uintptr
	array  entries
	edit   *uint32
}

func (n *bitmapIndexedNode) assoc(
	edit *uint32,
	shift uint,
	hash uintptr,
	k, v interface{},
) (node, bool) {
	if n.entryExists(hash, shift) {
		return n.assocExisting(edit, shift, hash, k, v)
	}
	return n.assocNew(edit, shift, hash, k, v), true
}

func (n *bitmapIndexedNode) assocNew(
	edit *uint32,
	shift uint,
	hash uintptr,
	k, v interface{},
) node {
	switch {
	case n.isFull():
		idx := mask(hash, shift)
		child, _ := emptySeededBitmapNode(n.seed).
			assoc(edit, shift+shiftBits, hash, k, v)
		return n.unpack(edit, shift, idx, child)
	default:
		return n.addNewEntry(edit, hash, shift, k, v)
	}
}

func (n *bitmapIndexedNode) assocExisting(
	edit *uint32,
	shift uint,
	hashval uintptr,
	k, v interface{},
) (node, bool) {
	bit := bitpos(hashval, shift)
	idx := n.index(bit)
	e := n.array[idx]
	switch {
	case !e.isLeaf():
		// Non-leaf node
		// Walk down the tree
		new, added := e.v.(node).
			assoc(edit, shift+shiftBits, hashval, k, v)
		if new == e.v {
			return n, added
		}
		editable := n.ensureEditable(edit)
		editable.array[idx].v = new
		return editable, added
	case e.matches(k):
		// A key replacement
		if dyn.Equal(v, e.v) {
			return n, false
		}
		editable := n.ensureEditable(edit)
		editable.array[idx].v = v
		return editable, false
	default:
		h1 := hash.Any(e.k, n.seed)
		if h1 == hashval {
			// A hash collision
			new := &hashCollisionNode{
				edit:  edit,
				seed:  n.seed,
				hash:  h1,
				array: []entry{e, {k: k, v: v}},
			}
			editable := n.ensureEditable(edit)
			editable.array.assoc(idx, entry{k: nil, v: new})
			return editable, true
		}

		// Push into new bitmap
		new, _ := emptySeededBitmapNode(n.seed).
			assoc(edit, shift+shiftBits, h1, e.k, e.v)
		new, _ = new.
			assoc(edit, shift+shiftBits, hashval, k, v)
		editable := n.ensureEditable(edit)
		editable.array.assoc(idx, entry{k: nil, v: new})
		return editable, true
	}
}

func (n *bitmapIndexedNode) addNewEntry(
	edit *uint32,
	hash uintptr,
	shift uint,
	k, v interface{},
) *bitmapIndexedNode {
	bit := bitpos(hash, shift)
	idx := n.index(bit)
	// Using ensureEditable here leads to two copies of
	// the array. To avoid that, inline the logic
	var editable *bitmapIndexedNode
	if isEditable(n.edit, edit) {
		editable = n
	} else {
		editable = &bitmapIndexedNode{
			bitmap: n.bitmap,
			seed:   n.seed,
			edit:   edit,
			array:  n.array.copyWithCap(len(n.array) + 1),
		}
	}
	editable.array = editable.array.insert(idx, entry{k: k, v: v})
	editable.bitmap |= bit
	return editable
}

func (n *bitmapIndexedNode) unpack(
	edit *uint32,
	shift uint,
	idx uint,
	child node,
) *arrayNode {
	nodes := new(array)
	nodes.assoc(idx, child)
	var j uint
	for i := uint(0); i < width; i++ {
		if ((n.bitmap >> i) & 1) == 0 {
			continue
		}
		entry := n.array[j]
		if entry.isLeaf() {
			node, _ := emptySeededBitmapNode(n.seed).
				assoc(edit,
					shift+shiftBits,
					hash.Any(entry.k, n.seed),
					entry.k,
					entry.v)
			nodes.assoc(i, node)
		} else {
			nodes.assoc(i, entry.v.(node))
		}
		j++
	}
	return &arrayNode{
		seed:  n.seed,
		edit:  edit,
		count: len(n.array) + 1,
		array: nodes,
	}
}

func (n *bitmapIndexedNode) without(
	edit *uint32,
	shift uint,
	hash uintptr,
	k interface{},
) (node, bool) {
	bit := bitpos(hash, shift)
	if !n.bitEntryExists(bit) {
		return n, false
	}
	idx := n.index(bit)
	ent := n.array[idx]
	switch {
	case !ent.isLeaf():
		child, removed := ent.v.(node).
			without(edit, shift+shiftBits, hash, k)
		switch {
		case child == ent.v:
			return n, removed
		case child != nil:
			editable := n.ensureEditable(edit)
			editable.array.assoc(idx, entry{k: nil, v: child})
			return editable, removed
		case n.bitmap == bit:
			return nil, removed
		default:
			editable := n.ensureEditable(edit)
			editable.array.remove(idx)
			editable.bitmap = editable.bitmap &^ bit
			return editable, removed
		}
	case dyn.Equal(k, ent.k):
		if n.bitmap == bit {
			return nil, true
		}
		editable := n.ensureEditable(edit)
		editable.array.remove(idx)
		editable.bitmap = editable.bitmap &^ bit
		return editable, true
	default:
		return n, false
	}
}

func (n *bitmapIndexedNode) find(
	shift uint,
	hash uintptr,
	k interface{},
) (interface{}, bool) {
	bit := bitpos(hash, shift)
	if (n.bitmap & bit) == 0 {
		return nil, false
	}
	idx := n.index(bit)
	ent := n.array[idx]
	if !ent.isLeaf() {
		return ent.v.(node).find(shift+shiftBits, hash, k)
	}
	if dyn.Equal(ent.k, k) {
		return ent.v, true
	}
	return nil, false
}

func (n *bitmapIndexedNode) seq() seq.Sequence {
	out := entrySeqNew(n.array, 0, nil)
	if out == nil {
		return nil
	}
	return out
}

func (n *bitmapIndexedNode) index(bit uint32) int {
	return bits.OnesCount32(n.bitmap & (bit - 1))
}

func (n *bitmapIndexedNode) isFull() bool {
	return len(n.array) >= bitmapCap
}

func (n *bitmapIndexedNode) entryExists(hash uintptr, shift uint) bool {
	bit := bitpos(hash, shift)
	return n.bitEntryExists(bit)
}

func (n *bitmapIndexedNode) bitEntryExists(bit uint32) bool {
	return n.bitmap&bit != 0
}

func (n *bitmapIndexedNode) ensureEditable(edit *uint32) *bitmapIndexedNode {
	if isEditable(n.edit, edit) {
		return n
	}
	return &bitmapIndexedNode{
		bitmap: n.bitmap,
		seed:   n.seed,
		array:  n.array.copy(),
		edit:   edit,
	}
}
