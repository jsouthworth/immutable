package btree_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/internal/btree"
)

func TestSet(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("s=Empty().Add(i)->s.Contains(i)",
		prop.ForAll(
			func(i int) bool {
				s := btree.Empty().Add(i)
				return s.Contains(i)
			},
			gen.Int(),
		))

	properties.Property("s=Empty().Add(i)->s.At(i)==i",
		prop.ForAll(
			func(i int) bool {
				s := btree.Empty().Add(i)
				return s.At(i) == i
			},
			gen.Int(),
		))
	properties.Property("s=large.At(i)==i",
		prop.ForAll(
			func(t *rtree) bool {
				foundAll := true
				for _, entry := range t.entries {
					foundAll = foundAll &&
						t.t.At(entry) == entry
				}
				return foundAll
			},
			genRandomTree,
		))

	properties.Property("s=Empty().Add(i).Delete(i)->!s.Contains(i)",
		prop.ForAll(
			func(i int) bool {
				s := btree.Empty().Add(i).Delete(i)
				return !s.Contains(i)
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i); r=s.Delete(i)->r != s",
		prop.ForAll(
			func(i int) bool {
				s := btree.Empty().Add(i)
				r := s.Delete(i)
				return r != s
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i).Delete(i); r=s.Delete(i)->r == s",
		prop.ForAll(
			func(i int) bool {
				s := btree.Empty().Add(i).Delete(i)
				r := s.Delete(i)
				return r == s
			},
			gen.Int(),
		))

	properties.Property("Creating a btree gives expected length",
		prop.ForAll(
			func(is []int) bool {
				m := make(map[int]struct{})
				s := btree.Empty()
				for _, i := range is {
					s = s.Add(i)
					m[i] = struct{}{}
				}
				return s.Length() == len(m)
			},
			gen.SliceOf(gen.Int()),
		))

	properties.TestingRun(t)
}

func TestContains(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.Contains(entry.k)", prop.ForAll(
		func(rt *rtree) bool {
			for _, key := range rt.entries {
				if !rt.t.Contains(key) {
					return false
				}
			}
			return true
		},
		genRandomTree,
	))
	properties.TestingRun(t)
}

func TestDelete(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("new=empty.Delete(k) -> new==empty", prop.ForAll(
		func(t *btree.BTree, k string) bool {
			new := t.Delete(k)
			return new == t
		},
		genTree,
		gen.Identifier(),
	))
	properties.Property("new=large.Delete(k) -> new!=large", prop.ForAll(
		func(lt *ltree) bool {
			new := lt.t.Delete(lt.k + strconv.Itoa(lt.num-1))
			return new != lt.t
		},
		genLargeTree,
	))
	properties.Property("new=large.Delete(k) -> !new.Contains(key) && larg.Contains(key)", prop.ForAll(
		func(lt *ltree) bool {
			key := lt.k + strconv.Itoa(lt.num-1)
			new := lt.t.Delete(key)
			return !new.Contains(key) && lt.t.Contains(key)
		},
		genLargeTree,
	))
	properties.Property("new=removeAll(large) -> new.Length()==0", prop.ForAll(
		func(lt *ltree) bool {
			new := lt.t
			for i := 0; i < lt.num; i++ {
				new = new.Delete(lt.k + strconv.Itoa(i))
			}
			return new.Length() == 0
		},
		genLargeTree,
	))
	properties.TestingRun(t)
}

func TestAdd(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("new=large.Add(k) -> new!=empty ", prop.ForAll(
		func(lm *ltree, k string) bool {
			new := lm.t.Add(k)
			return lm.t.Contains(k) || new != lm.t
		},
		genLargeTree,
		gen.Identifier(),
	))
	properties.Property("new=large.Add(k) -> new.At(k)==v", prop.ForAll(
		func(lm *ltree, k string) bool {
			new := lm.t.Add(k)
			return new.Contains(k)
		},
		genLargeTree,
		gen.Identifier(),
	))

	properties.Property("one=large.Add(k); two=one.Add(k) -> one==two", prop.ForAll(
		func(lm *ltree, k string) bool {
			one := lm.t.Add(k)
			two := one.Add(k)
			return one == two
		},
		genLargeTree,
		gen.Identifier(),
	))

	properties.Property("s=large.At(i).Find(i)==(i, found)",
		prop.ForAll(
			func(t *rtree) bool {
				if len(t.entries) == 0 {
					return true
				}
				val, found := t.t.Find(t.entries[0])
				return found && val == t.entries[0]
			},
			genRandomTree,
		))
	properties.Property("ForAll k=0-lm.num, large.At(k) == v", prop.ForAll(
		func(lm *ltree) bool {
			for i := 0; i < lm.num; i++ {
				k := lm.k + strconv.Itoa(i)
				if !lm.t.Contains(k) {
					return false
				}
			}
			return true
		},
		genLargeTree,
	))

	properties.TestingRun(t)
}

func TestTransientContains(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.Contains(entry.k)", prop.ForAll(
		func(rm *rtree) bool {
			t := rm.t.AsTransient()
			for _, key := range rm.entries {
				if !t.Contains(key) {
					return false
				}
			}
			return true
		},
		genRandomTree,
	))
	properties.TestingRun(t)
}

func TestTransientAdd(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("s=Empty().Add(i)->s.Contains(i)",
		prop.ForAll(
			func(i int) bool {
				s := btree.Empty().AsTransient().Add(i)
				return s.Contains(i)
			},
			gen.Int(),
		))
	properties.Property("Add is idempotent", prop.ForAll(
		func(i int) bool {
			t := btree.Empty().AsTransient()
			new := t.Add(i)
			new2 := t.Add(i)
			return new == new2
		},
		gen.Int(),
	))
	properties.Property("s=Empty().Add(i)->s.At(i)==i",
		prop.ForAll(
			func(i int) bool {
				s := btree.Empty().AsTransient().Add(i)
				return s.At(i) == i
			},
			gen.Int(),
		))
	properties.Property("s=large.At(i)==i",
		prop.ForAll(
			func(t *rtree) bool {
				trans := t.t.AsTransient()
				return len(t.entries) == 0 ||
					trans.At(t.entries[0]) == t.entries[0]
			},
			genRandomTree,
		))
	properties.Property("s=large.At(i).Find(i)==(i, found)",
		prop.ForAll(
			func(t *rtree) bool {
				trans := t.t.AsTransient()
				if len(t.entries) == 0 {
					return true
				}
				val, found := trans.Find(t.entries[0])
				return found && val == t.entries[0]
			},
			genRandomTree,
		))
	properties.Property("Creating a tree gives expected length",
		prop.ForAll(
			func(is []int) bool {
				m := make(map[int]struct{})
				s := btree.Empty().AsTransient()
				for _, i := range is {
					s = s.Add(i)
					m[i] = struct{}{}
				}
				return s.Length() == len(m)
			},
			gen.SliceOf(gen.Int()),
		))

	properties.TestingRun(t)
}

func TestTransientDelete(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("new=large.Delete(k) -> !new.Contains(key) && larg.Contains(key)", prop.ForAll(
		func(lt *ltree) bool {
			key := lt.k + strconv.Itoa(lt.num-1)
			new := lt.t.AsTransient().Delete(key)
			return !new.Contains(key) && lt.t.Contains(key)
		},
		genLargeTree,
	))
	properties.Property("delete is idempotenet", prop.ForAll(
		func(i int) bool {
			t := btree.Empty().AsTransient().Add(i)
			new := t.Delete(i)
			new2 := t.Delete(i)
			return new == new2
		},
		gen.Int(),
	))
	properties.Property("new=removeAll(large) -> new.Length()==0", prop.ForAll(
		func(lt *ltree) bool {
			new := lt.t.AsTransient()
			for i := 0; i < lt.num; i++ {
				new = new.Delete(lt.k + strconv.Itoa(i))
			}
			return new.Length() == 0
		},
		genLargeTree,
	))
	properties.TestingRun(t)
}

type mapEntry struct {
	key interface{}
	val interface{}
}

func (e mapEntry) Equal(other interface{}) bool {
	oe, ok := other.(mapEntry)
	return ok && dyn.Equal(e.key, oe.key) && dyn.Equal(oe.val, e.val)
}

func (e mapEntry) Compare(other interface{}) int {
	oe, ok := other.(mapEntry)
	if !ok {
		return 1
	}
	return dyn.Compare(e.key, oe.key)
}

func TestAsMap(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("entry can be repalaced", prop.ForAll(
		func(k, v1, v2 int) bool {
			m := btree.Empty().Add(mapEntry{key: k, val: v1})
			m2 := m.Add(mapEntry{key: k, val: v2})
			e1 := m.At(mapEntry{key: k}).(mapEntry)
			e2 := m2.At(mapEntry{key: k}).(mapEntry)
			return v1 == v2 ||
				(e1.key == e2.key && e1.val != e2.val) &&
					m2.Length() == 1
		},
		gen.Int(),
		gen.Int(),
		gen.Int(),
	))
	properties.Property("entry can be removed", prop.ForAll(
		func(k, v1, v2 int) bool {
			m := btree.Empty().Add(mapEntry{key: k, val: v1})
			m2 := m.Add(mapEntry{key: k, val: v2})
			m3 := m2.Delete(mapEntry{key: k})
			e1 := m.At(mapEntry{key: k}).(mapEntry)
			e2 := m2.At(mapEntry{key: k}).(mapEntry)
			e3 := m3.At(mapEntry{key: k})
			return v1 == v2 ||
				(e1.key == e2.key && e1.val != e2.val && e3 == nil)
		},
		gen.Int(),
		gen.Int(),
		gen.Int(),
	))
	properties.Property("custom compare and eq entry can be repalaced", prop.ForAll(
		func(k, v1, v2 int) bool {
			m := btree.Empty(
				btree.Compare(func(a, b interface{}) int {
					ae := a.(mapEntry)
					be := b.(mapEntry)
					return dyn.Compare(ae.key, be.key)
				}),
				btree.Equal(func(a, b interface{}) bool {
					ae, aok := a.(mapEntry)
					be, bok := b.(mapEntry)
					return aok && bok &&
						dyn.Compare(ae.key, be.key) == 0 &&
						dyn.Equal(ae.val, be.val)
				}),
			)
			m = m.Add(mapEntry{key: k, val: v1})
			m2 := m.Add(mapEntry{key: k, val: v2})
			val1, ok1 := m.Find(mapEntry{key: k})
			var e1, e2 mapEntry
			if ok1 {
				e1 = val1.(mapEntry)
			}
			val2, ok2 := m2.Find(mapEntry{key: k})
			if ok2 {
				e2 = val2.(mapEntry)
			}
			return v1 == v2 ||
				(e1.key == e2.key && e1.val != e2.val)
		},
		gen.Int(),
		gen.Int(),
		gen.Int(),
	))
	properties.Property("replace on large BTree works", prop.ForAll(
		func(num, k, v1, v2 int) bool {
			m := btree.Empty().Add(mapEntry{key: k, val: v1})
			for i := 1000; i < 1000+num; i++ {
				m = m.Add(mapEntry{key: i, val: i})
			}
			m2 := m.Add(mapEntry{key: k, val: v2})
			e1 := m.At(mapEntry{key: k}).(mapEntry)
			e2 := m2.At(mapEntry{key: k}).(mapEntry)
			return v1 == v2 ||
				(e1.key == e2.key && e1.val != e2.val) &&
					m2.Length() == num+1 &&
					m.Length() == num+1

		},
		gen.IntRange(10000, 20000),
		gen.IntRange(1, 100),
		gen.Int(),
		gen.Int(),
	))
	properties.Property("replace on large transient BTree works", prop.ForAll(
		func(num, k, v1, v2 int) bool {
			m := btree.Empty().AsTransient().
				Add(mapEntry{key: k, val: v1})
			for i := 1000; i < 1000+num; i++ {
				m = m.Add(mapEntry{key: i, val: i})
			}
			m = m.Add(mapEntry{key: k, val: v2})
			e := m.At(mapEntry{key: k}).(mapEntry)
			return v1 == v2 ||
				(e.key == k && e.val == v2) &&
					m.Length() == num+1

		},
		gen.IntRange(10000, 20000),
		gen.IntRange(1, 100),
		gen.Int(),
		gen.Int(),
	))
	properties.TestingRun(t)
}

func BenchmarkTransientAdd(b *testing.B) {
	t := btree.Empty().AsTransient()
	for i := 0; i < b.N; i++ {
		t = t.Add(i)
	}
}

func BenchmarkTransientDelete(b *testing.B) {
	t := btree.Empty().AsTransient()
	for i := 0; i < b.N; i++ {
		t = t.Add(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t = t.Delete(i)
	}
}

func BenchmarkAdd(b *testing.B) {
	t := btree.Empty()
	for i := 0; i < b.N; i++ {
		t = t.Add(i)
	}
}

func BenchmarkDelete(b *testing.B) {
	t := btree.Empty().AsTransient()
	for i := 0; i < b.N; i++ {
		t = t.Add(i)
	}
	p := t.AsPersistent()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p = p.Delete(i)
	}
}

func BenchmarkIter(b *testing.B) {
	t := btree.Empty().AsTransient()
	for i := 0; i < b.N; i++ {
		t = t.Add(i)
	}
	p := t.AsPersistent()
	b.ResetTimer()
	iter := p.Iterator()
	for iter.HasNext() {
		iter.Next()
	}
}

func BenchmarkBuiltinMapAdd(b *testing.B) {
	m := make(map[interface{}]interface{})
	for i := 0; i < b.N; i++ {
		m[i] = i
	}
}

func BenchmarkBuiltinMapDelete(b *testing.B) {
	m := make(map[interface{}]interface{})
	for i := 0; i < b.N; i++ {
		m[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		delete(m, i)
	}
}

func BenchmarkBuiltinMapIterate(b *testing.B) {
	m := make(map[interface{}]interface{})
	for i := 0; i < b.N; i++ {
		m[i] = i
	}
	b.ResetTimer()
	for k, v := range m {
		_ = k
		_ = v
	}
}

func TestSpeedTransientAdd(t *testing.T) {
	now := time.Now()
	tree := btree.Empty().AsTransient()
	for i := 0; i < 1000000; i++ {
		tree = tree.Add(i)
	}
	t.Log(time.Since(now))
}

func TestSpeedTransientContains(t *testing.T) {
	tree := btree.Empty().AsTransient()
	for i := 0; i < 100000; i++ {
		tree = tree.Add(i)
	}
	now := time.Now()
	for i := 0; i < 100000; i++ {
		tree.Contains(i)
	}
	t.Log(time.Since(now))
}

func TestSpeedTransientDelete(t *testing.T) {
	tree := btree.Empty().AsTransient()
	for i := 0; i < 100000; i++ {
		tree = tree.Add(i)
	}
	now := time.Now()
	for i := 0; i < 100000; i++ {
		tree = tree.Delete(i)
	}
	t.Log(time.Since(now))
}

func TestSpeedAdd(t *testing.T) {
	now := time.Now()
	tree := btree.Empty()
	for i := 0; i < 1000000; i++ {
		tree = tree.Add(i)
	}
	t.Log(time.Since(now))
}

func TestSpeedContains(t *testing.T) {
	tree := btree.Empty().AsTransient()
	for i := 0; i < 100000; i++ {
		tree = tree.Add(i)
	}
	p := tree.AsPersistent()
	now := time.Now()
	for i := 0; i < 100000; i++ {
		p.Contains(i)
	}
	t.Log(time.Since(now))
}

func TestSpeedDelete(t *testing.T) {
	tree := btree.Empty().AsTransient()
	for i := 0; i < 100000; i++ {
		tree = tree.Add(i)
	}
	p := tree.AsPersistent()
	now := time.Now()
	for i := 0; i < 100000; i++ {
		p = p.Delete(i)
	}
	t.Log(time.Since(now))
}

func TestSpeedIterator(t *testing.T) {
	tree := btree.Empty().AsTransient()
	for i := 0; i < 100000; i++ {
		tree = tree.Add(i)
	}
	p := tree.AsPersistent()
	now := time.Now()
	iter := p.Iterator()
	for iter.HasNext() {
		iter.Next()
	}
	t.Log(time.Since(now))
}

func TestIterator(t *testing.T) {
	tree := btree.Empty().AsTransient()
	var sum int
	for i := 0; i < 100000; i++ {
		tree = tree.Add(i)
		sum += i
	}
	p := tree.AsPersistent()
	iter := p.Iterator()
	var got int
	for iter.HasNext() {
		got += iter.Next().(int)
	}
	if sum != got {
		t.Fatalf("didn't get expected value from iteration: got %v expected %v", got, sum)
	}
}

func TestIteratorFrom(t *testing.T) {
	var froms = []int{-10, 0, 99997, 100000, 100001}
	sums := make([]int, len(froms))
	tree := btree.Empty().AsTransient()
	for i, from := range froms {
		var sum int
		for i := 0; i < 100000; i++ {
			tree = tree.Add(i)
			if i >= from {
				sum += i
			}
		}
		sums[i] = sum
	}
	p := tree.AsPersistent()
	for i, from := range froms {
		iter := p.IteratorFrom(from)
		var got int
		for iter.HasNext() {
			val := iter.Next().(int)
			got += val
		}
		if sums[i] != got {
			t.Fatalf("didn't get expected value from iteration: got %v expected %v", got, sums[i])
		}
	}
}

func TestIteratorEmpty(t *testing.T) {
	tree := btree.Empty()
	iter := tree.Iterator()
	var count int
	for iter.HasNext() {
		count++
		iter.Next()
	}
	if count > 0 {
		t.Fatal("Iterator over empty tree had next")
	}
}

func TestIteratorSmall(t *testing.T) {
	tree := btree.Empty().Add(1).Add(2).Add(3)
	iter := tree.Iterator()
	expected := 1 + 2 + 3
	var got int
	for iter.HasNext() {
		got += iter.Next().(int)
	}
	if got != expected {
		t.Fatalf("didn't get expected value from iteration: got %v expected %v", got, expected)
	}
}

func TestAsMapSmall(t *testing.T) {
	tree := btree.Empty().AsTransient()
	for i := 0; i < 98; i++ {
		tree.Add(mapEntry{key: i, val: i})
	}
	p := tree.AsPersistent()
	// 63 is selected strategically to test that
	// the internal node replacement works properly
	// the construction of the tree above will place
	// 63 as one of the maxKeys in the internal node.
	expected := mapEntry{key: 63, val: 64}
	p = p.Add(expected)
	got := p.At(mapEntry{key: 63})
	if expected != got {
		t.Fatalf("expected: %v, got: %v", expected, got)
	}
}

func TestAsMapSmallTransient(t *testing.T) {
	tree := btree.Empty().AsTransient()
	for i := 0; i < 98; i++ {
		tree.Add(mapEntry{key: i, val: i})
	}
	// 63 is selected strategically to test that
	// the internal node replacement works properly
	// the construction of the tree above will place
	// 63 as one of the maxKeys in the internal node.
	expected := mapEntry{key: 63, val: 64}
	tree.Add(expected)
	got := tree.At(mapEntry{key: 63})
	if expected != got {
		t.Fatalf("expected: %v, got: %v", expected, got)
	}
}

type rtree struct {
	entries []string
	t       *btree.BTree
}

func (t *rtree) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "{ entries:%v, t: %v }", t.entries, t.t)
	return b.String()
}

func makeRandomTree(entries []string) *rtree {
	m := btree.Empty()
	for _, val := range entries {
		m = m.Add(val)
	}
	return &rtree{
		entries: entries,
		t:       m,
	}
}

func unmakeRandomTree(r *rtree) []string {
	return r.entries
}

var genRandomTree = gopter.DeriveGen(makeRandomTree, unmakeRandomTree,
	gen.SliceOf(gen.Identifier()),
)

type ltree struct {
	num int
	k   string
	t   *btree.BTree
}

func (t *ltree) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "{ num:%v, k:%s, t: %v }", t.num, t.k, t.t)
	return b.String()
}

func makeLargeTree(num int, k string) *ltree {
	t := btree.Empty().AsTransient()
	for i := 0; i < num; i++ {
		t = t.Add(k + strconv.Itoa(i))
	}
	bt := t.AsPersistent()
	return &ltree{
		num: num,
		k:   k,
		t:   bt,
	}
}

func unmakeLargeTree(lt *ltree) (num int, k string) {
	return lt.num, lt.k
}

var genLargeTree = gopter.DeriveGen(makeLargeTree, unmakeLargeTree,
	gen.IntRange(10000, 20000),
	gen.Identifier(),
)

func makeTree() *btree.BTree {
	return btree.Empty()
}

func unmakeTree(m *btree.BTree) {
}

var genTree = gopter.DeriveGen(makeTree, unmakeTree)
