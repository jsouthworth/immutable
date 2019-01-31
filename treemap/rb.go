package treemap

import (
	"errors"
	"fmt"
)

//Based on : http://www.eecs.usma.edu/webs/people/okasaki/jfp99.ps
//      and: http://matt.might.net/papers/germane2014deletion.pdf

const (
	red color = iota
	black
	doubleBlack
	negativeBlack
)

type color uint8

func (in color) String() string {
	switch in {
	case negativeBlack:
		return "NB"
	case red:
		return "R"
	case black:
		return "B"
	case doubleBlack:
		return "BB"
	default:
		panic(errors.New("invalid color"))
	}
}

func (in color) addBlack() color {
	switch in {
	case negativeBlack:
		return red
	case red:
		return black
	case black:
		return doubleBlack
	case doubleBlack:
		panic(errors.New("already double black"))
	default:
		panic(errors.New("invalid color"))
	}
}

func (in color) addRed() color {
	switch in {
	case negativeBlack:
		panic(errors.New("already negative black"))
	case red:
		return negativeBlack
	case black:
		return red
	case doubleBlack:
		return black
	default:
		panic(errors.New("invalid color"))
	}
}

type cmpFunc func(k1, k2 interface{}) int

type entry struct {
	key   interface{}
	value interface{}
}

func (e entry) Key() interface{} {
	return e.key
}

func (e entry) Value() interface{} {
	return e.value
}

func (e entry) String() string {
	return fmt.Sprintf("[%v %v]", e.key, e.value)
}

type node struct {
	cmp   cmpFunc
	color color
	left  tree
	elem  entry
	right tree
}

func (n *node) String() string {
	return fmt.Sprintf("({%s %s} %s %s)", n.color, n.elem, n.left, n.right)
}

func (n *node) isTreeNode() {}

func (n *node) blacken() tree {
	if n.color == black {
		return n
	}
	return &node{
		cmp:   n.cmp,
		color: black,
		left:  n.left,
		elem:  n.elem,
		right: n.right,
	}
}

func (n *node) redden() tree {
	return &node{
		cmp:   n.cmp,
		color: red,
		left:  n.left,
		elem:  n.elem,
		right: n.right,
	}
}

func (n *node) addRed() tree {
	return &node{
		cmp:   n.cmp,
		color: n.color.addRed(),
		left:  n.left,
		elem:  n.elem,
		right: n.right,
	}
}

func (n *node) balance() tree {
	/* The pattern matching version of this is nicer and easy to understand
	 -- Okasaki's original cases:
	balance B (T R (T R a x b) y c) z d = T R (T B a x b) y (T B c z d)
	balance B (T R a x (T R b y c)) z d = T R (T B a x b) y (T B c z d)
	balance B a x (T R (T R b y c) z d) = T R (T B a x b) y (T B c z d)
	balance B a x (T R b y (T R c z d)) = T R (T B a x b) y (T B c z d)

	 -- Six cases for deletion:
	balance BB (T R (T R a x b) y c) z d = T B (T B a x b) y (T B c z d)
	balance BB (T R a x (T R b y c)) z d = T B (T B a x b) y (T B c z d)
	balance BB a x (T R (T R b y c) z d) = T B (T B a x b) y (T B c z d)
	balance BB a x (T R b y (T R c z d)) = T B (T B a x b) y (T B c z d)

	balance BB a x (T NB (T B b y c) z d@(T B _ _ _))
	    = T B (T B a x b) y (balance B c z (redden d))
	balance BB (T NB a@(T B _ _ _) x (T B b y c)) z d
	    = T B (balance B (redden a) x b) y (T B c z d)

	balance color a x b = T color a x b
	*/
	switch n.color {
	case black, doubleBlack:
		color := n.color.addRed()
		if left, ok := n.left.(*node); ok && left.color == red {
			if ll, ok := left.left.(*node); ok && ll.color == red {
				return &node{
					cmp:   n.cmp,
					color: color,
					left: &node{
						cmp:   n.cmp,
						color: black,
						left:  ll.left,
						elem:  ll.elem,
						right: ll.right,
					},
					elem: left.elem,
					right: &node{
						cmp:   n.cmp,
						color: black,
						left:  left.right,
						elem:  n.elem,
						right: n.right,
					},
				}
			}
			if lr, ok := left.right.(*node); ok && lr.color == red {
				return &node{
					cmp:   n.cmp,
					color: color,
					left: &node{
						cmp:   n.cmp,
						color: black,
						left:  left.left,
						elem:  left.elem,
						right: lr.left,
					},
					elem: lr.elem,
					right: &node{
						cmp:   n.cmp,
						color: black,
						left:  lr.right,
						elem:  n.elem,
						right: n.right,
					},
				}
			}
		}
		if right, ok := n.right.(*node); ok && right.color == red {
			if rl, ok := right.left.(*node); ok && rl.color == red {
				return &node{
					cmp:   n.cmp,
					color: color,
					left: &node{
						cmp:   n.cmp,
						color: black,
						left:  n.left,
						elem:  n.elem,
						right: rl.left,
					},
					elem: rl.elem,
					right: &node{
						cmp:   n.cmp,
						color: black,
						left:  rl.right,
						elem:  right.elem,
						right: right.right,
					},
				}
			}
			if rr, ok := right.right.(*node); ok && rr.color == red {
				return &node{
					cmp:   n.cmp,
					color: color,
					left: &node{
						cmp:   n.cmp,
						color: black,
						left:  n.left,
						elem:  n.elem,
						right: right.left,
					},
					elem: right.elem,
					right: &node{
						cmp:   n.cmp,
						color: black,
						left:  rr.left,
						elem:  rr.elem,
						right: rr.right,
					},
				}
			}
		}
	}
	if n.color == doubleBlack {
		//a few additional cases for the deleteion case.
		if left, ok := n.left.(*node); ok && left.color == negativeBlack {
			if ll, ok := left.left.(*node); ok && ll.color == black {
				if lr, ok := left.right.(*node); ok && lr.color == black {
					return &node{
						cmp:   n.cmp,
						color: black,
						left: balance(&node{
							cmp:   n.cmp,
							color: black,
							left:  redden(ll),
							elem:  left.elem,
							right: lr.left,
						}),
						elem: lr.elem,
						right: &node{
							cmp:   n.cmp,
							color: black,
							left:  lr.right,
							elem:  n.elem,
							right: n.right,
						},
					}
				}
			}
		}
		if right, ok := n.right.(*node); ok && right.color == negativeBlack {
			if rl, ok := right.left.(*node); ok && rl.color == black {
				if rr, ok := right.right.(*node); ok && rr.color == black {
					return &node{
						cmp:   n.cmp,
						color: black,
						left: &node{
							cmp:   n.cmp,
							color: black,
							left:  n.left,
							elem:  n.elem,
							right: rl.left,
						},
						elem: rl.elem,
						right: balance(&node{
							cmp:   n.cmp,
							color: black,
							left:  rl.right,
							elem:  right.elem,
							right: redden(rr),
						}),
					}
				}
			}
		}
	}
	return n
}

func (n *node) bubble() tree {
	switch {
	case isDoubleBlack(n.left) || isDoubleBlack(n.right):
		return balance(&node{
			cmp:   n.cmp,
			color: n.color.addBlack(),
			left:  addRed(n.left),
			elem:  n.elem,
			right: addRed(n.right),
		})
	default:
		return balance(n)
	}

}

func (n *node) insert(key, value interface{}) (tree, bool) {
	cmp := n.cmp(key, n.elem.key)
	switch {
	case cmp < 0:
		newLeft, added := ins(n.left, key, value)
		if newLeft == n.left {
			return n, false
		}
		return balance(&node{
			cmp:   n.cmp,
			color: n.color,
			left:  newLeft,
			elem:  n.elem,
			right: n.right,
		}), added
	case cmp > 0:
		newRight, added := ins(n.right, key, value)
		if newRight == n.right {
			return n, false
		}
		return balance(&node{
			cmp:   n.cmp,
			color: n.color,
			left:  n.left,
			elem:  n.elem,
			right: newRight,
		}), added
	default:
		if !equal(n.elem.value, value) {
			return &node{
				cmp:   n.cmp,
				color: n.color,
				left:  n.left,
				elem:  entry{key, value},
				right: n.right,
			}, false
		}
		return n, false
	}
}

func (n *node) delete(key interface{}) tree {
	cmp := n.cmp(key, n.elem.key)
	switch {
	case cmp < 0:
		left := del(n.left, key)
		if left == n.left {
			return n
		}
		return bubble(&node{
			cmp:   n.cmp,
			color: n.color,
			left:  left,
			elem:  n.elem,
			right: n.right,
		})
	case cmp > 0:
		right := del(n.right, key)
		if right == n.right {
			return n
		}
		return bubble(&node{
			cmp:   n.cmp,
			color: n.color,
			left:  n.left,
			elem:  n.elem,
			right: right,
		})
	default:
		return remove(n)
	}
}

func (n *node) remove() tree {
	left, leftIsNode := n.left.(*node)
	right, rightIsNode := n.right.(*node)
	_, leftIsLeaf := n.left.(*leaf)
	_, rightIsLeaf := n.right.(*leaf)
	switch {
	case n.color == red && leftIsLeaf && rightIsLeaf:
		return &leaf{}
	case n.color == black && leftIsLeaf && rightIsLeaf:
		return &doubleBlackLeaf{}
	case n.color == black && leftIsLeaf && rightIsNode && right.color == red:
		return &node{
			cmp:   n.cmp,
			color: black,
			left:  right.left,
			elem:  right.elem,
			right: right.right,
		}
	case n.color == black && leftIsNode && left.color == red && rightIsLeaf:
		return &node{
			cmp:   n.cmp,
			color: black,
			left:  left.left,
			elem:  left.elem,
			right: left.right,
		}
	default:
		return bubble(&node{
			cmp:   n.cmp,
			color: n.color,
			left:  removeMax(n.left),
			elem:  max(n.left),
			right: n.right,
		})
	}
}

func (n *node) removeMax() tree {
	if _, rightIsLeaf := n.right.(*leaf); rightIsLeaf {
		return remove(n)
	}
	return bubble(&node{
		cmp:   n.cmp,
		color: n.color,
		left:  n.left,
		elem:  n.elem,
		right: removeMax(n.right),
	})
}

func (n *node) max() entry {
	_, rightIsLeaf := n.right.(*leaf)
	if rightIsLeaf {
		return n.elem
	}
	return max(n.right)
}

func (n *node) get(key interface{}) (entry, bool) {
	cmp := n.cmp(key, n.elem.key)
	switch {
	case cmp < 0:
		return get(n.left, key)
	case cmp > 0:
		return get(n.right, key)
	default:
		return n.elem, true
	}
}

func (n *node) isDoubleBlack() bool {
	return n.color == doubleBlack
}

func (n *node) leftBranch() tree {
	return n.left
}

func (n *node) rightBranch() tree {
	return n.right
}

func (n *node) value() entry {
	return n.elem
}

type leaf struct {
	cmp cmpFunc
}

func (l *leaf) isTreeNode() {}

func (l *leaf) blacken() tree {
	return l
}

func (l *leaf) insert(key, value interface{}) (tree, bool) {
	return &node{
		cmp:   l.cmp,
		color: red,
		left:  &leaf{cmp: l.cmp},
		elem:  entry{key: key, value: value},
		right: &leaf{cmp: l.cmp},
	}, true
}
func (l *leaf) delete(_ interface{}) tree {
	return l
}

func (l *leaf) get(key interface{}) (entry, bool) {
	return entry{}, false
}

func (l *leaf) String() string {
	return "L"
}

func (l *leaf) isDoubleBlack() bool { return false }

type doubleBlackLeaf struct {
	cmp cmpFunc
}

func (l *doubleBlackLeaf) isTreeNode() {}

func (l *doubleBlackLeaf) blacken() tree {
	return &leaf{cmp: l.cmp}
}

func (l *doubleBlackLeaf) addRed() tree {
	return &leaf{cmp: l.cmp}
}

func (l *doubleBlackLeaf) String() string {
	return "BBL"
}

func (l *doubleBlackLeaf) isDoubleBlack() bool { return true }

type tree interface {
	isTreeNode()
}

// These helper functions allow us to implement different behavior on each node type and
// avoid the need to panic ourselves if a particular type doesn't understand the the behavior.
// The panics will signal a problem with the implementation but should never occur in real code.
func blacken(t tree) tree {
	return t.(interface{ blacken() tree }).blacken()
}

func redden(t tree) tree {
	return t.(interface{ redden() tree }).redden()
}

func addRed(t tree) tree {
	return t.(interface{ addRed() tree }).addRed()
}

func balance(t tree) tree {
	return t.(interface{ balance() tree }).balance()
}

func bubble(t tree) tree {
	return t.(interface{ bubble() tree }).bubble()
}

func ins(t tree, key, value interface{}) (tree, bool) {
	return t.(interface {
		insert(key, value interface{}) (tree, bool)
	}).insert(key, value)
}
func insert(t tree, key, value interface{}) (tree, bool) {
	t, added := ins(t, key, value)
	t = blacken(t)
	return t, added
}

func del(t tree, key interface{}) tree {
	return t.(interface {
		delete(key interface{}) tree
	}).delete(key)
}

func _delete(t tree, key interface{}) tree {
	return blacken(del(t, key))
}

func remove(t tree) tree {
	return t.(interface{ remove() tree }).remove()
}

func removeMax(t tree) tree {
	return t.(interface{ removeMax() tree }).removeMax()
}

func max(t tree) entry {
	return t.(interface{ max() entry }).max()
}

func get(t tree, key interface{}) (entry, bool) {
	return t.(interface {
		get(key interface{}) (entry, bool)
	}).get(key)
}

func contains(t tree, key interface{}) bool {
	_, ok := get(t, key)
	return ok
}

func isDoubleBlack(t tree) bool {
	return t.(interface{ isDoubleBlack() bool }).isDoubleBlack()
}

func right(t tree) tree {
	return t.(interface{ rightBranch() tree }).rightBranch()
}

func left(t tree) tree {
	return t.(interface{ leftBranch() tree }).leftBranch()
}

func value(t tree) entry {
	return t.(interface{ value() entry }).value()
}
