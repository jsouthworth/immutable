package hashmap

import "unsafe"

const (
	maxDepth = (unsafe.Sizeof(uintptr(0))*8 + shiftBits - 1) / shiftBits
)

// Iterator provides a mutable iterator over the map. This allows
// efficient, heap allocation-less access to the contents. Iterators
// are not safe for concurrent access so they may not be shared
// between goroutines.
func (m *Map) Iterator() Iterator {
	i := makeIterator(m.root)
	i.HasNext() // Make sure the initial iterator value is valid
	return i
}

// Iterator is a mutable iterator for a map. It has a fixed size
// stack, the size of which is computed from the maximum number of
// nested nodes possible based on the branching factor and the size of
// the hash type.
type Iterator struct {
	depth uintptr
	stack [maxDepth + 1]struct {
		n   node
		cur int
	}
}

func makeIterator(n node) Iterator {
	var i Iterator
	i.stack[0].n = n
	return i
}

// HasNext is true when there are more elements to be iterated over.
func (i *Iterator) HasNext() bool {
	state := i.stack[i.depth]
	switch n := state.n.(type) {
	case *arrayNode:
		for j := state.cur; j < width; j++ {
			node := n.array[j]
			if node != nil {
				i.stack[i.depth].cur = j + 1
				i.pushNode(node)
				return i.HasNext()
			}
		}
		if i.depth == 0 {
			return false
		}
		i.popNode()
		return i.HasNext()
	case *bitmapIndexedNode:
		for j := state.cur; j < len(n.array); j++ {
			entry := n.array[j]
			if entry.isLeaf() {
				i.stack[i.depth].cur = j
				return true
			} else {
				n, ok := entry.v.(node)
				if !ok || n == nil {
					continue
				} else {
					i.stack[i.depth].cur = j + 1
					i.pushNode(n)
					return i.HasNext()
				}
			}
		}
		if i.depth == 0 {
			return false
		}
		i.popNode()
		return i.HasNext()
	case *hashCollisionNode:
		for j := state.cur; j < len(n.array); j++ {
			entry := n.array[j]
			if entry.isLeaf() {
				i.stack[i.depth].cur = j
				return true
			} else {
				n, ok := entry.v.(node)
				if !ok || n == nil {
					continue
				} else {
					i.stack[i.depth].cur = j + 1
					i.pushNode(n)
					return i.HasNext()
				}
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

// Next provides the next key value pair and increments the cursor.
func (i *Iterator) Next() (k, v interface{}) {
	state := i.stack[i.depth]
	switch n := state.n.(type) {
	case *arrayNode:
		// HasNext should always step away from arrayNodes
		// panic if we find one in Next
		panic("arrayNode!")
	case *bitmapIndexedNode:
		entry := n.array[state.cur]
		i.stack[i.depth].cur++
		return entry.k, entry.v
	case *hashCollisionNode:
		entry := n.array[state.cur]
		i.stack[i.depth].cur++
		return entry.k, entry.v
	default:
		panic("No such entry")
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
