package treemap

import (
	"fmt"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// The following tests the behavior of the RB tree.
// It does so by treating it as a set to avoid the complications
// of key replacement. The map behavior will be tested at another level.
func propNoRedRed(t tree) bool {
	_, rootIsLeaf := t.(*leaf)
	switch {
	case rootIsLeaf:
		return true
	default:
		root := t.(*node)
		left, leftIsNode := root.left.(*node)
		right, rightIsNode := root.right.(*node)
		switch {
		case leftIsNode && left.color == red:
			return root.color != red
		case rightIsNode && right.color == red:
			return root.color != red
		default:
			return propNoRedRed(root.left) &&
				propNoRedRed(root.right)
		}
	}
}

func blackDepth(t tree) int {
	root, rootIsNode := t.(*node)
	switch {
	case rootIsNode:
		switch root.color {
		case red:
			n, m := blackDepth(root.left), blackDepth(root.right)
			switch {
			case n < 0 || m < 0:
				return -1
			case n == m:
				return n
			default:
				return -1
			}
		default:
			n, m := blackDepth(root.left), blackDepth(root.right)
			switch {
			case n < 0 || m < 0:
				return -1
			case n == m:
				return n + 1
			default:
				return -1
			}
		}
	default:
		_ = t.(*leaf)
		return 1
	}
}
func propBalancedBlack(s *rbset) bool {
	return blackDepth(s.t) > 0
}

func propOrdered(s *rbset) bool {
	return true
}

func propRBValid(s *rbset) bool {
	return propNoRedRed(s.t) &&
		propBalancedBlack(s) &&
		propOrdered(s)
}

func TestRBValid(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("No Red Red", prop.ForAll(
		func(s *rbset) bool { return propNoRedRed(s.t) },
		genRBSet,
	))
	properties.Property("Balanced Black", prop.ForAll(
		propBalancedBlack,
		genRBSet,
	))
	properties.Property("Ordered", prop.ForAll(
		propOrdered,
		genRBSet,
	))
	properties.TestingRun(t)
}

func propInsertValid(s *rbset, i int) bool {
	new, added := insert(s.t, i, nil)
	if added {
		s.entries = append(s.entries, i)
	}
	return propRBValid(&rbset{t: new, entries: s.entries})
}

func propInsertMember(s *rbset, i int) bool {
	new, _ := insert(s.t, i, nil)
	return contains(new, i)
}

func propInsertSafe(s *rbset, x, y int) bool {
	new, _ := insert(s.t, y, nil)
	return contains(s.t, x) == contains(new, x)
}

func propNoInsertPhantom(s *rbset, x, y int) bool {
	new, _ := insert(s.t, y, nil)
	return (!contains(s.t, x) && x != y) == !contains(new, x)
}

func TestInsertion(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Insert maintains RB constraints", prop.ForAll(
		propInsertValid,
		genRBSet,
		gen.Int(),
	))
	properties.Property("Insert adds member", prop.ForAll(
		propInsertMember,
		genRBSet,
		gen.Int(),
	))
	properties.Property("Insert doesn't update unrelated item", prop.ForAll(
		propInsertSafe,
		genRBSet,
		gen.Int(),
		gen.Int(),
	))
	properties.Property("Insert doesn't add more than the expected item", prop.ForAll(
		propNoInsertPhantom,
		genRBSet,
		gen.Int(),
		gen.Int(),
	))
	properties.Property("Insert/Insert produces the same tree", prop.ForAll(
		func(s *rbset, i int) bool {
			t, _ := insert(s.t, i, nil)
			t2, _ := insert(t, i, nil)
			return t == t2
		},
		genRBSet,
		gen.Int(),
	))
	properties.TestingRun(t)
}

func removeFromSlice(sl []int, val int) []int {
	newEntries := sl
	var iAtIndex int
	for idx, entry := range newEntries {
		if entry == val {
			iAtIndex = idx
		}
	}
	copy(newEntries[iAtIndex:], newEntries[iAtIndex+1:])
	newEntries = newEntries[:len(newEntries)-1]
	return newEntries
}

func propInsertDeleteValid(s *rbset, i int) bool {
	newEntries := s.entries
	new, added := insert(s.t, i, nil)
	if added {
		newEntries = append(newEntries, i)
	}
	new = _delete(s.t, i)
	newEntries = removeFromSlice(newEntries, i)
	return propRBValid(&rbset{t: new, entries: newEntries})
}

func propDeleteValid(s *rbset, i int) bool {
	newEntries := s.entries
	new := _delete(s.t, i)
	if new != s.t {
		newEntries = removeFromSlice(newEntries, i)
	}
	return propRBValid(&rbset{t: new, entries: newEntries})
}

func propMemberDelete(s *rbset, i int) bool {
	if contains(s.t, i) {
		return !contains(_delete(s.t, i), i)
	}
	return true
}

func propDeletePreservesOther(s *rbset, x, y int) bool {
	return x != y && contains(s.t, y) == contains(_delete(s.t, x), y)
}

func TestDeletion(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Delete/Insert duals", prop.ForAll(
		propInsertDeleteValid,
		genRBSet,
		gen.Int(),
	))
	properties.Property("Delete produces valid tree", prop.ForAll(
		propDeleteValid,
		genRBSet,
		gen.Int(),
	))
	properties.Property("Delete removes member", prop.ForAll(
		propMemberDelete,
		genRBSet,
		gen.Int(),
	))
	properties.Property("Delete preserves other items", prop.ForAll(
		propDeletePreservesOther,
		genRBSet,
		gen.Int(),
		gen.Int(),
	))
	properties.Property("Delete removes all", prop.ForAll(
		func(s *rbset) (ok bool) {
			t := s.t
			defer func() {
				r := recover()
				ok = r == nil

			}()
			for _, entry := range s.entries {
				t = _delete(t, entry)
			}
			_, ok = t.(*leaf)
			return ok
		},
		genRBSet,
	))
	properties.Property("Insert/Delete/Delete yeilds the same tree", prop.ForAll(
		func(s *rbset, i int) bool {
			t := s.t
			t, _ = insert(t, i, nil)
			t1 := _delete(t, i)
			t2 := _delete(t1, i)
			return t1 == t2
		},
		genRBSet,
		gen.Int(),
	))
	properties.TestingRun(t)
}

type rbset struct {
	entries []int
	t       tree
}

func (s *rbset) String() string {
	return fmt.Sprintf("%v, %s", s.entries, s.t)
}

func makeRBSet(entries []int) *rbset {
	var added bool
	t := tree(&leaf{cmp: defaultCompare})
	storedEntries := make([]int, 0, len(entries))
	for _, entry := range entries {
		t, added = insert(t, entry, nil)
		if added {
			storedEntries = append(storedEntries, entry)
		}
	}
	//fmt.Println("created set with", storedEntries)
	//fmt.Println("set", t)
	return &rbset{entries: storedEntries, t: t}
}

func unmakeRBSet(s *rbset) []int {
	return s.entries
}

var genRBSet = gopter.DeriveGen(makeRBSet, unmakeRBSet,
	gen.SliceOfN(100, gen.Int()).
		SuchThat(func(sl []int) bool { return len(sl) > 0 }))

func BenchmarkInsert(b *testing.B) {
	b.ReportAllocs()
	t := tree(&leaf{cmp: defaultCompare})
	for i := 0; i < b.N; i++ {
		t, _ = insert(t, i, nil)
	}
}
