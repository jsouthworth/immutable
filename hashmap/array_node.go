package hashmap

import (
	"jsouthworth.net/go/seq"
)

type arrayNode struct {
	seed  uintptr
	count int
	array *array
	edit  *uint32
}

func (n *arrayNode) ensureEditable(edit *uint32) *arrayNode {
	if isEditable(n.edit, edit) {
		return n
	}
	return &arrayNode{
		seed:  n.seed,
		count: n.count,
		array: n.array.copy(),
		edit:  edit,
	}

}

func (n *arrayNode) editAndSet(edit *uint32, idx uint, v node) *arrayNode {
	n = n.ensureEditable(edit)
	n.array.assoc(idx, v)
	return n
}

func (n *arrayNode) assoc(
	edit *uint32,
	shift uint,
	hash uintptr,
	key, val interface{},
) (node, bool) {
	idx := mask(hash, shift)
	node := n.array[idx]
	if node == nil {
		ch, added := emptySeededBitmapNode(n.seed).
			assoc(edit, shift+shiftBits, hash, key, val)
		editable := n.editAndSet(edit, idx, ch)
		editable.count++
		return editable, added
	}
	ch, added := node.assoc(edit, shift+shiftBits, hash, key, val)
	if ch == node && !isEditable(n.edit, edit) {
		return n, false
	}
	return n.editAndSet(edit, idx, ch), added
}

func (n *arrayNode) without(
	edit *uint32,
	shift uint,
	hash uintptr,
	k interface{},
) (node, bool) {
	idx := mask(hash, shift)
	node := n.array[idx]
	if node == nil {
		return n, false
	}
	child, removed := node.without(edit, shift+shiftBits, hash, k)
	switch child {
	case node:
		return n, removed
	case nil:
		if n.count <= bitmapCap/2 {
			return n.pack(edit, idx), removed
		}
		editable := n.editAndSet(edit, idx, child)
		editable.count--
		return editable, removed
	default:
		return n.editAndSet(edit, idx, child), removed
	}

}

func (n *arrayNode) pack(edit *uint32, idx uint) *bitmapIndexedNode {
	var bitmap uint32
	var j int
	array := make(entries, n.count-1)
	for i := uint(0); i < idx; i++ {
		if n.array[i] == nil {
			continue
		}
		array[j].v = n.array[i]
		bitmap |= 1 << uint32(i)
		j++
	}
	for i := idx + 1; i < uint(len(n.array)); i++ {
		if n.array[i] == nil {
			continue
		}
		array[j].v = n.array[i]
		bitmap |= 1 << uint32(i)
		j++
	}
	return &bitmapIndexedNode{
		bitmap: bitmap,
		seed:   n.seed,
		array:  array,
		edit:   edit,
	}
}

func (n *arrayNode) find(
	shift uint,
	hash uintptr,
	k interface{},
) (interface{}, bool) {
	idx := mask(hash, shift)
	node := n.array[idx]
	if node == nil {
		return nil, false
	}
	return node.find(shift+shiftBits, hash, k)
}

func (n *arrayNode) seq() seq.Sequence {
	out := arrayNodeSeqNew(n.array, 0, nil)
	if out == nil {
		return nil
	}
	return out
}

func (n *arrayNode) rnge(fn func(Entry) bool) bool {
	for _, node := range n.array {
		if node == nil {
			continue
		}
		if !node.rnge(fn) {
			return false
		}
	}
	return true
}

type array [width]node

func (a *array) copy() *array {
	var tmp array
	copy(tmp[:], a[:])
	return &tmp
}

func (a *array) assoc(i uint, v node) *array {
	a[i] = v
	return a
}

func arrayNewFromSlice(in []node) *array {
	var out array
	copy(out[:], in)
	return &out
}

type arrayNodeSeq struct {
	nodes *array
	index int
	s     seq.Sequence
}

func arrayNodeSeqNew(nodes *array, index int, s seq.Sequence) *arrayNodeSeq {
	if s != nil {
		return &arrayNodeSeq{
			nodes: nodes,
			index: index,
			s:     s,
		}
	}
	for i := index; i < len(nodes); i++ {
		node := nodes[i]
		if node == nil {
			continue
		}
		nodeSeq := node.seq()
		if nodeSeq == nil {
			continue
		}
		return &arrayNodeSeq{
			nodes: nodes,
			index: i + 1,
			s:     nodeSeq,
		}
	}
	return nil
}

func (s *arrayNodeSeq) First() interface{} {
	return s.s.First()
}

func (s *arrayNodeSeq) Next() seq.Sequence {
	out := arrayNodeSeqNew(s.nodes, s.index, s.s.Next())
	if out == nil {
		return nil
	}
	return out

}

func (s *arrayNodeSeq) String() string {
	return seq.ConvertToString(s)
}
