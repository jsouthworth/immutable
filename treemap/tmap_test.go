package treemap

import (
	"strconv"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"jsouthworth.net/go/dyn"
)

func TestTransientAt(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.At(entry.k)==entry.v", prop.ForAll(
		func(rm *rmap) bool {
			t := rm.m.AsTransient()
			for key, val := range rm.entries {
				if val != t.At(key) {
					return false
				}
			}
			return true
		},
		genRandomMap,
	))
	properties.TestingRun(t)
}

func TestTransientEntryAt(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.EntryAt(entry.k).Value()==entry.v", prop.ForAll(
		func(rm *rmap) bool {
			t := rm.m.AsTransient()
			for key, val := range rm.entries {
				entry := t.EntryAt(key)
				if entry.Key() != key || entry.Value() != val {
					return false
				}
			}
			return true
		},
		genRandomMap,
	))
	properties.Property("new=large.Delete(k) -> new.EntryAt(k)==nil && large.EntryAt(k)==nil", prop.ForAll(
		func(lm *lmap) bool {
			t := lm.m.AsTransient()
			key := lm.k + strconv.Itoa(lm.num-1)
			new := t.Delete(key)
			return new.EntryAt(key) == nil && t.EntryAt(key) == nil
		},
		genLargeMap,
	))
	properties.TestingRun(t)
}

func TestTransientContains(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.Contains(entry.k)", prop.ForAll(
		func(rm *rmap) bool {
			t := rm.m.AsTransient()
			for key := range rm.entries {
				if !t.Contains(key) {
					return false
				}
			}
			return true
		},
		genRandomMap,
	))
	properties.TestingRun(t)
}

func TestTransientFind(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.Find(entry.k) is non-nil and exists", prop.ForAll(
		func(rm *rmap) bool {
			t := rm.m.AsTransient()
			for key := range rm.entries {
				v, ok := t.Find(key)
				if v == nil || !ok {
					return false
				}
			}
			return true
		},
		genRandomMap,
	))
	properties.Property("Non-existent keys don't exist in map", prop.ForAll(
		func(rm *rmap, key string) bool {
			t := rm.m.AsTransient()
			_, inEntries := rm.entries[key]
			_, inMap := t.Find(key)
			return inEntries == inMap
		},
		genRandomMap,
		gen.Identifier(),
	))
	properties.TestingRun(t)
}

func TestTransientAssoc(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("new = transient.Assoc(k,v) -> new == transient ", prop.ForAll(
		func(m *Map, k, v string) bool {
			t := m.AsTransient()
			new := t.Assoc(k, v)
			return new == t
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=t.Assoc(k, v) -> new.At(k)==v", prop.ForAll(
		func(m *Map, k, v string) bool {
			t := m.AsTransient()
			new := t.Assoc(k, v)
			got := new.At(k)
			return got == v
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("one=empty.Assoc(k, v); two=one.Assoc(k, v) -> one==two", prop.ForAll(
		func(m *Map, k, v string) bool {
			one := m.AsTransient().Assoc(k, v)
			two := one.Assoc(k, v)
			return one == two
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("new=large.Assoc(k,v) -> new!=empty ", prop.ForAll(
		func(lm *lmap, k, v string) bool {
			t := lm.m.AsTransient()
			new := t.Assoc(k, v)
			return new == t
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("ForAll k=0-lm.num, large.At(k) == v", prop.ForAll(
		func(lm *lmap) bool {
			t := lm.m.AsTransient()
			for i := 0; i < lm.num; i++ {
				k := lm.k + strconv.Itoa(i)
				v := lm.v + strconv.Itoa(i)
				got := t.At(k)
				if got != v {
					return false
				}
			}
			return true
		},
		genLargeMap,
	))

	properties.TestingRun(t)
}

func TestTransientConj(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("new = t.Conj(k,v) -> new == t ", prop.ForAll(
		func(m *Map, k, v string) bool {
			t := m.AsTransient()
			new := t.Conj(EntryNew(k, v))
			return new == t
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.TestingRun(t)
}

func TestTransientDelete(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)

	properties.Property("new=large.Delete(k) -> new.At(k)==nil", prop.ForAll(
		func(lm *lmap) bool {
			t := lm.m.AsTransient()
			key := lm.k + strconv.Itoa(lm.num-1)
			new := t.Delete(key)
			return new.At(key) == nil
		},
		genLargeMap,
	))
	properties.Property("new=removeAll(large) -> new.Length()==0", prop.ForAll(
		func(lm *lmap) bool {
			new := lm.m.AsTransient()
			for i := 0; i < lm.num; i++ {
				new = new.Delete(lm.k + strconv.Itoa(i))
			}
			return new.Length() == 0
		},
		genLargeMap,
	))
	properties.TestingRun(t)
}

func TestTransientLength(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("new=empty.Assoc(k, v) -> new.Length()==empty.Length()+1", prop.ForAll(
		func(m *Map, k, v string) bool {
			new := m.AsTransient().Assoc(k, v)
			return new.Length() == m.Length()+1
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=large.Assoc(k,v) -> new.Length()==large.Length()+1", prop.ForAll(
		func(lm *lmap, k, v string) bool {
			new := lm.m.AsTransient().Assoc(k, v)
			return new.Length() == lm.m.Length()+1
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("lm.num == lm.m.Length()", prop.ForAll(
		func(lm *lmap) bool {
			return lm.m.AsTransient().Length() == lm.num
		},
		genLargeMap,
	))
	properties.Property("new=empty.Assoc(k, v).Delete(k) -> new.Length()==empty.Length()", prop.ForAll(
		func(m *Map, k, v string) bool {
			new := m.AsTransient().Assoc(k, v).Delete(k)
			return new.Length() == m.Length()
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=large.Assoc(k,v).Delete(k) -> new.Length()==large.Length()", prop.ForAll(
		func(lm *lmap, k, v string) bool {
			new := lm.m.AsTransient().Assoc(k, v).Delete(k)
			return new.Length() == lm.m.Length()
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("lm.num == lm.m.Length()", prop.ForAll(
		func(lm *lmap) bool {
			return lm.m.AsTransient().Length() == lm.num
		},
		genLargeMap,
	))
	properties.Property("random.Length() increases correctly", prop.ForAll(
		func(rm *rmap, entries map[string]string) bool {
			m := rm.m.AsTransient()
			count := m.Length()
			for key, val := range entries {
				if !m.Contains(key) {
					count++
				}
				m = m.Assoc(key, val)
			}
			return m.Length() == count
		},
		genRandomMap,
		gen.MapOf(gen.Identifier(), gen.Identifier()),
	))
	properties.TestingRun(t)
}

func TestTransientEqual(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("m == m", prop.ForAll(
		func(rm *rmap) bool {
			return rm.m.AsTransient().Equal(rm.m.AsTransient())
		},
		genRandomMap,
	))
	properties.Property("new=m.Delete(k) -> new != m", prop.ForAll(
		func(rm *rmap) bool {
			var k string
			rm.m.Range(func(key, val string) bool {
				k = key
				return false
			})
			new := rm.m.AsTransient().Delete(k)
			return !rm.m.AsTransient().Equal(new)
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() != 0
		}),
	))
	properties.Property("m.Equal(10)==false", prop.ForAll(
		func(rm *rmap) bool {
			return !rm.m.AsTransient().Equal(10)
		},
		genRandomMap,
	))
	properties.Property("new=m.Assoc(k,v) -> new != m", prop.ForAll(
		func(rm *rmap) bool {
			var k string
			rm.m.Range(func(key, val string) bool {
				k = key
				return false
			})
			new := rm.m.AsTransient().Assoc(k, "foo")
			return !rm.m.AsTransient().Equal(new)
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() != 0
		}),
	))
	properties.TestingRun(t)
}

func TestTransientApply(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("dyn.Apply(s, i)==s.At(i)",
		prop.ForAll(
			func(is []int) bool {
				s := From(is).AsTransient()
				return s.At(is[0]) == dyn.Apply(s, is[0])
			},
			gen.SliceOfN(10, gen.Int()),
		))
	properties.TestingRun(t)
}

func TestTransientRange(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Range access the full map", prop.ForAll(
		func(rm *rmap) bool {
			foundAll := true
			rm.m.AsTransient().Range(func(key, val interface{}) bool {
				if !foundAll {
					return false
				}
				foundAll = rm.entries[key.(string)] == val
				return true
			})

			return foundAll
		},
		genRandomMap,
	))
	properties.Property("Range access the full map no continue", prop.ForAll(
		func(rm *rmap) bool {
			foundAll := true
			rm.m.AsTransient().Range(func(key, val interface{}) {
				if !foundAll {
					return
				}
				foundAll = rm.entries[key.(string)] == val
				return
			})

			return foundAll
		},
		genRandomMap,
	))
	properties.Property("Range access the full map entries", prop.ForAll(
		func(rm *rmap) bool {
			foundAll := true
			rm.m.AsTransient().Range(func(entry Entry) bool {
				if !foundAll {
					return false
				}
				foundAll = rm.entries[entry.Key().(string)] == entry.Value()
				return true
			})

			return foundAll
		},
		genRandomMap,
	))
	properties.Property("Range access the full map entries no continue", prop.ForAll(
		func(rm *rmap) bool {
			foundAll := true
			rm.m.AsTransient().Range(func(entry Entry) {
				if !foundAll {
					return
				}
				foundAll = rm.entries[entry.Key().(string)] == entry.Value()
				return
			})

			return foundAll
		},
		genRandomMap,
	))
	properties.Property("Range with reflected func", prop.ForAll(
		func(rm *rmap) bool {
			foundAll := true
			rm.m.AsTransient().Range(func(key, val string) bool {
				if !foundAll {
					return false
				}
				foundAll = rm.entries[key] == val
				return true
			})

			return foundAll
		},
		genRandomMap,
	))
	properties.Property("Range with reflected func no continue", prop.ForAll(
		func(rm *rmap) bool {
			foundAll := true
			rm.m.AsTransient().Range(func(key, val string) {
				if !foundAll {
					return
				}
				foundAll = rm.entries[key] == val
			})

			return foundAll
		},
		genRandomMap,
	))
	properties.Property("Range panics when passed a non function", prop.ForAll(
		func(rm *rmap) (ok bool) {
			defer func() {
				r := recover()
				ok = r == errRangeSig
			}()

			rm.m.AsTransient().Range(1)
			return false
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() > 0
		}),
	))
	properties.Property("Range panics when passed a function with the wrong number of inputs", prop.ForAll(
		func(rm *rmap) (ok bool) {
			defer func() {
				r := recover()
				ok = r == errRangeSig
			}()

			rm.m.AsTransient().Range(func(a, b, c string) {})
			return false
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() > 0
		}),
	))
	properties.Property("Range panics when passed a function with the wrong number of outputs", prop.ForAll(
		func(rm *rmap) (ok bool) {
			defer func() {
				r := recover()
				ok = r == errRangeSig
			}()

			rm.m.AsTransient().Range(func(a, b, c string) (d, e bool) { return false, false })
			return false
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() > 0
		}),
	))
	properties.Property("Range panics when passed a function with the wrong output type", prop.ForAll(
		func(rm *rmap) (ok bool) {
			defer func() {
				r := recover()
				ok = r == errRangeSig
			}()

			rm.m.AsTransient().Range(func(a, b string) string { return "" })
			return false
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() > 0
		}),
	))
	properties.Property("Range panics when passed a function with the wrong input types", prop.ForAll(
		func(rm *rmap) (ok bool) {
			ok = true
			defer func() {
				_ = recover()
			}()

			rm.m.AsTransient().Range(func(a, b int) {})
			return false
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() > 0
		}),
	))
	properties.TestingRun(t)
}

func TestTransientReduce(t *testing.T) {
	// This is a quick test of reduce since the underlying mechanisms
	// are tested thoroughly elsewhere

	t.Run("func(init interface{}, entry Entry) interface{}",
		func(t *testing.T) {
			m := New(1, 1, 2, 2, 3, 3, 4, 4, 5, 5).AsTransient()
			out := m.Reduce(func(res interface{}, entry Entry) interface{} {
				return res.(int) + entry.Value().(int)
			}, 0)
			if out != 1+2+3+4+5 {
				t.Fatal("didn't get expected value", out)
			}
		})
	t.Run("func(init interface{}, entry interface{}) interface{}",
		func(t *testing.T) {
			m := New(1, 1, 2, 2, 3, 3, 4, 4, 5, 5).AsTransient()
			out := m.Reduce(func(res interface{}, in interface{}) interface{} {
				entry := in.(Entry)
				return res.(int) + entry.Value().(int)
			}, 0)
			if out != 1+2+3+4+5 {
				t.Fatal("didn't get expected value", out)
			}
		})
	t.Run("func(init, k, v interface{}) interface{}",
		func(t *testing.T) {
			m := New(1, 1, 2, 2, 3, 3, 4, 4, 5, 5).AsTransient()
			out := m.Reduce(func(res, k, v interface{}) interface{} {
				return res.(int) + v.(int)
			}, 0)
			if out != 1+2+3+4+5 {
				t.Fatal("didn't get expected value", out)
			}
		})
	t.Run("func(init int, e Entry) int",
		func(t *testing.T) {
			m := New(1, 1, 2, 2, 3, 3, 4, 4, 5, 5).AsTransient()
			out := m.Reduce(func(res int, e Entry) int {
				return res + e.Value().(int)
			}, 0)
			if out != 1+2+3+4+5 {
				t.Fatal("didn't get expected value", out)
			}
		})

	t.Run("Transient func(init, k, v int) int",
		func(t *testing.T) {
			m := New(1, 1, 2, 2, 3, 3, 4, 4, 5, 5).AsTransient()
			out := m.Reduce(func(res, k, v int) int {
				return res + v
			}, 0)
			if out != 1+2+3+4+5 {
				t.Fatal("didn't get expected value", out)
			}
		})
}
