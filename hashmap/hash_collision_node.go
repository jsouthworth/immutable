package hashmap

import (
	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/seq"
)

type hashCollisionNode struct {
	hash  uintptr
	seed  uintptr
	edit  *uint32
	array entries
}

func (n *hashCollisionNode) assoc(
	edit *uint32,
	shift uint,
	hash uintptr,
	k, v interface{},
) (node, bool) {
	if hash == n.hash {
		idx, ok := n.findIndex(k)
		if ok {
			if dyn.Equal(n.array[idx].v, v) {
				return n, false
			}
			return n.editAndSet(edit, idx, v), false
		}
		return n.editAndAppend(edit, entry{k: k, v: v}), true
	}
	out := &bitmapIndexedNode{
		edit:   edit,
		bitmap: bitpos(n.hash, shift),
		array:  []entry{entry{k: nil, v: n}},
	}
	return out.assoc(edit, shift, hash, k, v)
}

func (n *hashCollisionNode) findIndex(k interface{}) (int, bool) {
	for i, e := range n.array {
		if dyn.Equal(k, e.k) {
			return i, true
		}
	}
	return -1, false
}

func (n *hashCollisionNode) ensureEditable(edit *uint32) *hashCollisionNode {
	if isEditable(n.edit, edit) {
		return n
	}
	return &hashCollisionNode{
		hash:  n.hash,
		seed:  n.seed,
		edit:  edit,
		array: n.array.copy(),
	}
}

func (n *hashCollisionNode) editAndSet(
	edit *uint32,
	idx int,
	val interface{},
) *hashCollisionNode {
	editable := n.ensureEditable(edit)
	editable.array[idx].v = val
	return editable
}

func (n *hashCollisionNode) editAndAppend(edit *uint32, e entry) *hashCollisionNode {
	if isEditable(n.edit, edit) {
		n.array = n.array.append(e)
		return n
	}

	return &hashCollisionNode{
		hash:  n.hash,
		seed:  n.seed,
		edit:  edit,
		array: n.array.copyWithCap(len(n.array) + 1).append(e),
	}
}

func (n *hashCollisionNode) without(
	edit *uint32,
	shift uint,
	hash uintptr,
	k interface{},
) (node, bool) {
	idx, ok := n.findIndex(k)
	if !ok {
		return n, false
	}
	if len(n.array) == 1 {
		return nil, true
	}
	editable := n.ensureEditable(edit)
	editable.array = editable.array.remove(idx)
	return editable, true
}

func (n *hashCollisionNode) find(
	shift uint,
	hash uintptr,
	k interface{},
) (interface{}, bool) {
	idx, ok := n.findIndex(k)
	if !ok {
		return nil, false
	}
	if dyn.Equal(k, n.array[idx].k) {
		return n.array[idx].v, true
	}
	return nil, false
}

func (n *hashCollisionNode) seq() seq.Sequence {
	out := entrySeqNew(n.array, 0, nil)
	if out == nil {
		return nil
	}
	return out
}

func (n *hashCollisionNode) rnge(fn func(Entry) bool) bool {
	for _, entry := range n.array {
		if entry.isLeaf() {
			if !fn(entry) {
				return false
			}
			continue
		}
		n, ok := entry.v.(node)
		if !ok || n == nil {
			continue
		}
		if !n.rnge(fn) {
			return false
		}
	}
	return true
}
