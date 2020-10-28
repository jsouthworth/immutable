package btree

import (
	"fmt"
	"sort"
	"strings"

	"jsouthworth.net/go/immutable/internal/atomic"
)

type leafNode struct {
	keys []interface{}
	len  int
	edit *atomic.Bool
}

func newLeaf(len int, edit *atomic.Bool) *leafNode {
	out := leafNode{
		len:  len,
		edit: edit,
	}
	if edit.Deref() {
		out.keys = make([]interface{}, min(maxLen, len+expandLen))
	} else {
		out.keys = make([]interface{}, len)
	}
	return &out
}

func (n *leafNode) isEditable() bool {
	return n.edit.Deref()
}

func (n *leafNode) leafPart() *leafNode {
	return n
}

func (n *leafNode) maxKey() interface{} {
	return n.keys[n.len-1]
}

func (n *leafNode) search(key interface{}, cmp compareFunc) int {
	i := sort.Search(n.len, func(i int) bool {
		return cmp(n.keys[i], key) >= 0
	})
	if i < n.len && cmp(key, n.keys[i]) == 0 {
		return i
	} else {
		return (-i) - 1
	}
}

func (n *leafNode) searchFirst(key interface{}, cmp compareFunc) int {
	return sort.Search(n.len, func(i int) bool {
		return cmp(n.keys[i], key) >= 0
	})
}

func (n *leafNode) searchEq(key interface{}, cmp compareFunc, eq eqFunc) (int, bool) {
	i := sort.Search(n.len, func(i int) bool {
		return cmp(n.keys[i], key) >= 0
	})
	if i < n.len && cmp(key, n.keys[i]) == 0 {
		valsEqual := eq(key, n.keys[i])
		if valsEqual {
			return i, false
		}
		return -i - 1, true
	} else {
		return (-i) - 1, false
	}
}

func (n *leafNode) find(key interface{}, cmp compareFunc) (interface{}, bool) {
	var out interface{}
	v := n.search(key, cmp)
	if v >= 0 {
		out = n.keys[v]
	}
	return out, v >= 0
}

func (n *leafNode) add(
	key interface{},
	cmp compareFunc,
	eq eqFunc,
	edit *atomic.Bool,
) (out nodeReturn) {
	idx, replace := n.searchEq(key, cmp, eq)
	if idx >= 0 && !replace {
		return nodeReturn{status: returnUnchanged}
	}
	ins := (-idx) - 1

	if n.isEditable() && (n.len < len(n.keys) || replace) {
		return n.modifyInPlace(ins, key, edit, replace)
	}

	if replace {
		return n.copyAndReplaceNode(ins, key, edit)
	}

	if n.len < maxLen {
		return n.copyAndInsertNode(ins, key, edit)
	}

	return n.split(ins, key, edit)
}

func (n *leafNode) modifyInPlace(
	ins int, key interface{}, edit *atomic.Bool, replace bool,
) nodeReturn {
	if replace {
		n.keys[ins] = key
		return nodeReturn{status: returnReplaced, nodes: [3]node{n}}
	} else if ins == n.len {
		n.keys[n.len] = key
		n.len++
		return nodeReturn{status: returnOne, nodes: [3]node{n}}
	} else {
		copy(n.keys[ins+1:], n.keys[ins:n.len])
		n.keys[ins] = key
		n.len++
		return nodeReturn{status: returnEarly}
	}
}

func (n *leafNode) copyAndInsertNode(
	ins int, key interface{}, edit *atomic.Bool,
) nodeReturn {
	nl := newLeaf(n.len+1, edit)
	ks := keyStitcher{nl.keys, 0}
	ks.copyAll(n.keys, 0, ins)
	ks.copyOne(key)
	ks.copyAll(n.keys, ins, n.len)
	return nodeReturn{status: returnOne, nodes: [3]node{nl}}
}

func (n *leafNode) copyAndReplaceNode(
	ins int, key interface{}, edit *atomic.Bool,
) nodeReturn {
	nl := newLeaf(n.len, edit)
	copy(nl.keys, n.keys)
	nl.keys[ins] = key
	return nodeReturn{status: returnReplaced, nodes: [3]node{nl}}
}

func (n *leafNode) split(
	ins int, key interface{}, edit *atomic.Bool,
) nodeReturn {
	firstHalf := (n.len + 1) >> 1
	secondHalf := n.len + 1 - firstHalf
	n1 := newLeaf(firstHalf, edit)
	n2 := newLeaf(secondHalf, edit)

	if ins < firstHalf {
		ks := keyStitcher{n1.keys, 0}
		ks.copyAll(n.keys, 0, ins)
		ks.copyOne(key)
		ks.copyAll(n.keys, ins, firstHalf-1)
		copy(n2.keys, n.keys[firstHalf-1:n.len])
		return nodeReturn{status: returnTwo, nodes: [3]node{n1, n2}}
	}

	copy(n1.keys, n.keys[0:firstHalf])
	ks := keyStitcher{n2.keys, 0}
	ks.copyAll(n.keys, firstHalf, ins)
	ks.copyOne(key)
	ks.copyAll(n.keys, ins, n.len)
	return nodeReturn{status: returnTwo, nodes: [3]node{n1, n2}}
}

func (n *leafNode) remove(
	key interface{},
	leftNode, rightNode node,
	cmp compareFunc,
	edit *atomic.Bool,
) (out nodeReturn) {
	idx := n.search(key, cmp)
	if idx < 0 {
		return nodeReturn{status: returnUnchanged}
	}

	newLen := n.len - 1

	var left, right *leafNode
	if leftNode != nil {
		left = leftNode.leafPart()
	}
	if rightNode != nil {
		right = rightNode.leafPart()
	}

	switch {
	case !n.needsMerge(newLen, left, right):
		if n.isEditable() {
			return n.removeInPlace(idx, newLen, left, right, edit)
		}
		return n.copyAndRemoveIdx(idx, newLen, left, right, edit)
	case left.canJoin(newLen):
		return n.joinLeft(idx, newLen, left, right, edit)
	case right.canJoin(newLen):
		return n.joinRight(idx, newLen, left, right, edit)
	case left != nil &&
		(left.isEditable() || right == nil || left.len >= right.len):
		return n.borrowLeft(idx, newLen, left, right, edit)
	case right != nil:
		return n.borrowRight(idx, newLen, left, right, edit)
	default:
		panic("unreachable")
	}
}

func (n *leafNode) needsMerge(
	newLen int,
	left, right *leafNode,
) bool {
	return newLen < minLen && (left != nil || right != nil)
}

func (n *leafNode) removeInPlace(
	idx, newLen int,
	left, right *leafNode,
	edit *atomic.Bool,
) nodeReturn {
	copy(n.keys[idx:], n.keys[idx+1:n.len])
	n.len = newLen
	if idx == newLen {
		return nodeReturn{
			status: returnThree,
			nodes: [...]node{
				leafNodeToNode(left),
				n,
				leafNodeToNode(right),
			},
		}
	}
	return nodeReturn{status: returnEarly}
}

func (n *leafNode) copyAndRemoveIdx(
	idx, newLen int,
	left, right *leafNode,
	edit *atomic.Bool,
) nodeReturn {
	center := newLeaf(newLen, edit)
	copy(center.keys, n.keys[0:idx])
	copy(center.keys[idx:], n.keys[idx+1:])
	return nodeReturn{
		status: returnThree,
		nodes: [...]node{
			leafNodeToNode(left),
			center,
			leafNodeToNode(right),
		},
	}
}

func (n *leafNode) joinLeft(
	idx, newLen int,
	left, right *leafNode,
	edit *atomic.Bool,
) nodeReturn {
	join := newLeaf(left.len+newLen, edit)
	ks := keyStitcher{join.keys, 0}
	ks.copyAll(left.keys, 0, left.len)
	ks.copyAll(n.keys, 0, idx)
	ks.copyAll(n.keys, idx+1, n.len)
	return nodeReturn{
		status: returnThree,
		nodes:  [...]node{nil, join, leafNodeToNode(right)},
	}
}

func (n *leafNode) joinRight(
	idx, newLen int,
	left, right *leafNode,
	edit *atomic.Bool,
) nodeReturn {
	join := newLeaf(right.len+newLen, edit)
	ks := keyStitcher{join.keys, 0}
	ks.copyAll(n.keys, 0, idx)
	ks.copyAll(n.keys, idx+1, n.len)
	ks.copyAll(right.keys, 0, right.len)
	return nodeReturn{
		status: returnThree,
		nodes:  [...]node{leafNodeToNode(left), join, nil},
	}
}

func (n *leafNode) canJoin(newLen int) bool {
	return n != nil && (n.len+newLen) < maxLen
}

func (n *leafNode) borrowLeft(
	idx, newLen int,
	left, right *leafNode,
	edit *atomic.Bool,
) nodeReturn {
	var (
		totalLen     = left.len + newLen
		newLeftLen   = totalLen >> 1
		newCenterLen = totalLen - newLeftLen
		leftTail     = left.len - newLeftLen
	)

	var newLeft, newCenter *leafNode

	// prepend to center
	if n.isEditable() && newCenterLen <= len(n.keys) {
		newCenter = n
		copy(n.keys[leftTail+idx:], n.keys[idx+1:n.len])
		copy(n.keys[leftTail:], n.keys[0:idx])
		copy(n.keys[0:], left.keys[newLeftLen:left.len])
		n.len = newCenterLen
	} else {
		newCenter = newLeaf(newCenterLen, edit)
		ks := keyStitcher{newCenter.keys, 0}
		ks.copyAll(left.keys, newLeftLen, left.len)
		ks.copyAll(n.keys, 0, idx)
		ks.copyAll(n.keys, idx+1, n.len)
	}

	// shrink left
	if left.isEditable() {
		newLeft = left
		left.len = newLeftLen
	} else {
		newLeft = newLeaf(newLeftLen, edit)
		copy(newLeft.keys, left.keys[0:newLeftLen])
	}

	return nodeReturn{
		status: returnThree,
		nodes:  [...]node{newLeft, newCenter, leafNodeToNode(right)},
	}
}

func (n *leafNode) borrowRight(
	idx, newLen int,
	left, right *leafNode,
	edit *atomic.Bool,
) nodeReturn {
	var (
		totalLen     = newLen + right.len
		newCenterLen = totalLen >> 1
		newRightLen  = totalLen - newCenterLen
		rightHead    = right.len - newRightLen
	)

	var newCenter, newRight *leafNode

	// append to center
	if n.isEditable() && newCenterLen <= len(n.keys) {
		newCenter = n
		ks := keyStitcher{n.keys, idx}
		ks.copyAll(n.keys, idx+1, n.len)
		ks.copyAll(right.keys, 0, rightHead)
		n.len = newCenterLen
	} else {
		newCenter = newLeaf(newCenterLen, edit)
		ks := keyStitcher{newCenter.keys, 0}
		ks.copyAll(n.keys, 0, idx)
		ks.copyAll(n.keys, idx+1, n.len)
		ks.copyAll(right.keys, 0, rightHead)
	}

	//cut head from right
	if right.isEditable() {
		newRight = right
		copy(right.keys, right.keys[rightHead:right.len])
		right.len = newRightLen
	} else {
		newRight = newLeaf(newRightLen, edit)
		copy(newRight.keys, right.keys[rightHead:right.len])
	}
	return nodeReturn{
		status: returnThree,
		nodes:  [...]node{leafNodeToNode(left), newCenter, newRight},
	}
}

func (n *leafNode) String() string {
	var b strings.Builder
	n.string(&b, 0)
	return b.String()
}

func (n *leafNode) string(b *strings.Builder, lvl int) {
	b.WriteRune('{')
	for i := 0; i < n.len; i++ {
		if i > 0 {
			b.WriteRune(' ')
		}
		fmt.Fprintf(b, "%v", n.keys[i])
	}
	b.WriteRune('}')
}

func leafNodeToNode(n *leafNode) node {
	if n != nil {
		return n
	}
	return nil
}
