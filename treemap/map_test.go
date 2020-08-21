package treemap

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"jsouthworth.net/go/dyn"
)

func assert(t *testing.T, b bool, msg string) {
	if !b {
		t.Fatal(msg)
	}
}

func BenchmarkPMapAssoc(b *testing.B) {
	b.ReportAllocs()
	m := Empty()
	for i := 0; i < b.N; i++ {
		m = m.Assoc(i, i)
	}
}

func BenchmarkNativeMapAssoc(b *testing.B) {
	b.ReportAllocs()
	m := make(map[int]int)
	for i := 0; i < b.N; i++ {
		m[i] = i
	}
}

func BenchmarkNativeMapInterfaceAssoc(b *testing.B) {
	b.ReportAllocs()
	m := make(map[interface{}]interface{})
	for i := 0; i < b.N; i++ {
		m[i] = i
	}
}

func TestNew(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("New requires even number of elements", prop.ForAll(
		func(elems []interface{}) (ok bool) {
			ok = true
			defer func() {
				_ = recover()
			}()
			_ = New(elems...)
			return false
		},
		gen.SliceOf(gen.Identifier(), reflect.TypeOf((*interface{})(nil)).Elem()).
			SuchThat(func(sl []interface{}) bool { return len(sl)%2 != 0 }),
	))
	properties.Property("New produces expected map", prop.ForAll(
		func(elems []interface{}) bool {
			m := New(elems...)
			exp := make(map[interface{}]interface{})
			for i := 0; i < len(elems); i = i + 2 {
				key := elems[i]
				val := elems[i+1]
				exp[key] = val
			}
			for key, val := range exp {
				if m.At(key) != val {
					fmt.Println(key, m.At(key), val)
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.Identifier(), reflect.TypeOf((*interface{})(nil)).Elem()).
			SuchThat(func(sl []interface{}) bool { return len(sl)%2 == 0 }),
	))
	properties.TestingRun(t)
}

func TestFrom(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("From(m) == m", prop.ForAll(
		func(rm *rmap) bool {
			new := From(rm.m)
			return new == rm.m
		},
		genRandomMap,
	))
	properties.Property("From(map[interface{}]interface{}) builds correct map", prop.ForAll(
		func(m map[interface{}]interface{}) bool {
			pm := From(m)
			for k, v := range m {
				if pm.At(k) != v {
					return false
				}
			}
			return true
		},
		gopter.DeriveGen(
			func(entries map[string]string) map[interface{}]interface{} {
				out := make(map[interface{}]interface{})
				for k, v := range entries {
					out[k] = v
				}
				return out

			},
			func(m map[interface{}]interface{}) map[string]string {
				out := make(map[string]string)
				for k, v := range m {
					out[k.(string)] = v.(string)
				}
				return out
			},
			gen.MapOf(gen.Identifier(), gen.Identifier()),
		),
	))
	properties.Property("From([]Entry) builds correct map", prop.ForAll(
		func(entries []Entry) bool {
			m := From(entries)
			for _, entry := range entries {
				if m.At(entry.Key()) != entry.Value() {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gopter.DeriveGen(
			func(k, v string) Entry {
				return Entry(entry{key: k, value: v})
			},
			func(e Entry) (k, v string) {
				return e.Key().(string), e.Value().(string)
			},
			gen.Identifier(),
			gen.Identifier(),
		), reflect.TypeOf((*Entry)(nil)).Elem()).
			SuchThat(func(entries []Entry) bool {
				seen := make(map[string]struct{})
				for _, entry := range entries {
					_, ok := seen[entry.Key().(string)]
					if ok {
						return false
					}
					seen[entry.Key().(string)] = struct{}{}
				}
				return true
			}),
	))
	properties.Property("From([]interface{}) builds correct map", prop.ForAll(
		func(elems []interface{}) bool {
			m := From(elems)
			for i := 0; i < len(elems); i += 2 {
				k := elems[i]
				v := elems[i+1]
				if m.At(k) != v {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.Identifier(),
			reflect.TypeOf((*interface{})(nil)).Elem()).
			SuchThat(func(sl []interface{}) bool {
				return len(sl)%2 == 0
			}).
			SuchThat(func(elems []interface{}) bool {
				seen := make(map[string]struct{})
				for i := 0; i < len(elems); i += 2 {
					k := elems[i]
					_, ok := seen[k.(string)]
					if ok {
						return false
					}
					seen[k.(string)] = struct{}{}
				}
				return true
			}),
	))
	properties.Property("From(map[T]T) builds correct map", prop.ForAll(
		func(in map[string]string) bool {
			m := From(in)
			for k, v := range in {
				if m.At(k) != v {
					return false
				}
			}
			return true
		},
		gen.MapOf(gen.Identifier(), gen.Identifier()),
	))
	properties.Property("From(map[kT]vT) builds correct map", prop.ForAll(
		func(in map[string]int) bool {
			m := From(in)
			for k, v := range in {
				if m.At(k) != v {
					return false
				}
			}
			return true
		},
		gen.MapOf(gen.Identifier(), gen.Int()),
	))
	properties.Property("From([]T) builds correct map", prop.ForAll(
		func(elems []string) bool {
			m := From(elems)
			for i := 0; i < len(elems); i += 2 {
				k := elems[i]
				v := elems[i+1]
				if m.At(k) != v {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.Identifier()).
			SuchThat(func(sl []string) bool {
				return len(sl)%2 == 0
			}).
			SuchThat(func(elems []string) bool {
				seen := make(map[string]struct{})
				for i := 0; i < len(elems); i += 2 {
					k := elems[i]
					_, ok := seen[k]
					if ok {
						return false
					}
					seen[k] = struct{}{}
				}
				return true
			}),
	))
	properties.Property("From(int) returns empty", prop.ForAll(
		func(i int) bool {
			m := From(i)
			return m.Length() == 0
		},
		gen.Int(),
	))
	properties.TestingRun(t)
}

func TestAt(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.At(entry.k)==entry.v", prop.ForAll(
		func(rm *rmap) bool {
			for key, val := range rm.entries {
				if val != rm.m.At(key) {
					return false
				}
			}
			return true
		},
		genRandomMap,
	))
	properties.TestingRun(t)
}

func TestEntryAt(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.EntryAt(entry.k).Value()==entry.v", prop.ForAll(
		func(rm *rmap) bool {
			for key, val := range rm.entries {
				entry := rm.m.EntryAt(key)
				if entry.Key() != key || entry.Value() != val {
					return false
				}
			}
			return true
		},
		genRandomMap,
	))
	properties.Property("new=large.Delete(k) -> new.EntryAt(k)==nil && large.EntryAt(k)==entry{k,v}", prop.ForAll(
		func(lm *lmap) bool {
			key := lm.k + strconv.Itoa(lm.num-1)
			val := lm.v + strconv.Itoa(lm.num-1)
			new := lm.m.Delete(key)
			return new.EntryAt(key) == nil &&
				lm.m.EntryAt(key).Key() == key &&
				lm.m.EntryAt(key).Value() == val
		},
		genLargeMap,
	))
	properties.TestingRun(t)
}

func TestContains(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.Contains(entry.k)", prop.ForAll(
		func(rm *rmap) bool {
			for key := range rm.entries {
				if !rm.m.Contains(key) {
					return false
				}
			}
			return true
		},
		genRandomMap,
	))
	properties.TestingRun(t)
}

func TestFind(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("ForAll generatedEntries random.Find(entry.k) is non-nil and exists", prop.ForAll(
		func(rm *rmap) bool {
			for key := range rm.entries {
				v, ok := rm.m.Find(key)
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
			_, inEntries := rm.entries[key]
			_, inMap := rm.m.Find(key)
			return inEntries == inMap
		},
		genRandomMap,
		gen.Identifier(),
	))
	properties.TestingRun(t)
}

func TestAssoc(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("new = empty.Assoc(k,v) -> new != empty ", prop.ForAll(
		func(m *Map, k, v string) bool {
			new := m.Assoc(k, v)
			return new != m
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=empty.Assoc(k, v) -> new.At(k)==v", prop.ForAll(
		func(m *Map, k, v string) bool {
			new := m.Assoc(k, v)
			got := new.At(k)
			return got == v
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=empty.Assoc(k, v) -> empty.At(k)!=v", prop.ForAll(
		func(m *Map, k, v string) bool {
			m.Assoc(k, v)
			got := m.At(k)
			return got != v
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("one=empty.Assoc(k, v); two=one.Assoc(k, v) -> one==two", prop.ForAll(
		func(m *Map, k, v string) bool {
			one := m.Assoc(k, v)
			two := one.Assoc(k, v)
			return one == two
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("one=empty.Assoc(k, v1); two=one.Assoc(k, v2) -> one != two", prop.ForAll(
		func(m *Map, k, v1, v2 string) bool {
			one := m.Assoc(k, v1)
			two := one.Assoc(k, v2)
			return v1 == v2 || one != two
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("one=empty.Assoc(k, v1); two=one.Assoc(k, v2) -> one.At(k)!=two.At(k)", prop.ForAll(
		func(m *Map, k, v1, v2 string) bool {
			one := m.Assoc(k, v1)
			two := one.Assoc(k, v2)
			return v1 == v2 || one.At(k) != two.At(k)
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.Property("new=large.Assoc(k,v) -> new!=empty ", prop.ForAll(
		func(lm *lmap, k, v string) bool {
			new := lm.m.Assoc(k, v)
			return new != lm.m
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=large.Assoc(k, v) -> new.At(k)==v", prop.ForAll(
		func(lm *lmap, k, v string) bool {
			new := lm.m.Assoc(k, v)
			got := new.At(k)
			return got == v
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=large.Assoc(k, v) -> empty.At(k)!=v", prop.ForAll(
		func(lm *lmap, k, v string) bool {
			lm.m.Assoc(k, v)
			got := lm.m.At(k)
			return got != v
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("one=large.Assoc(k, v); two=one.Assoc(k, v) -> one==two", prop.ForAll(
		func(lm *lmap, k, v string) bool {
			one := lm.m.Assoc(k, v)
			two := one.Assoc(k, v)
			return one == two
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("one=large.Assoc(k, v1); two=one.Assoc(k, v2) -> one!=two", prop.ForAll(
		func(lm *lmap, k, v1, v2 string) bool {
			one := lm.m.Assoc(k, v1)
			two := one.Assoc(k, v2)
			return v1 == v2 || one != two
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("one=large.Assoc(k, v1); two=one.Assoc(k, v2) -> one.At(k)!=two.At(k)", prop.ForAll(
		func(lm *lmap, k, v1, v2 string) bool {
			one := lm.m.Assoc(k, v1)
			two := one.Assoc(k, v2)
			return v1 == v2 || one.At(k) != two.At(k)
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("ForAll k=0-lm.num, large.At(k) == v", prop.ForAll(
		func(lm *lmap) bool {
			for i := 0; i < lm.num; i++ {
				k := lm.k + strconv.Itoa(i)
				v := lm.v + strconv.Itoa(i)
				got := lm.m.At(k)
				if got != v {
					return false
				}
			}
			return true
		},
		genLargeMap,
	))

	properties.Property("one=random.Assoc(k, v1); two=one.Assoc(k, v2) -> one.At(k)!=two.At(k)", prop.ForAll(
		func(rm *rmap, k, v1, v2 string) bool {
			one := rm.m.Assoc(k, v1)
			two := one.Assoc(k, v2)
			return v1 == v2 || one.At(k) != two.At(k)
		},
		genRandomMap,
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.TestingRun(t)
}

func TestConj(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("new = empty.Conj(k,v) -> new != empty ", prop.ForAll(
		func(m *Map, k, v string) bool {
			new := m.Conj(EntryNew(k, v))
			return new != m
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.TestingRun(t)
}

func TestDelete(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("new=empty.Delete(k) -> new==empty", prop.ForAll(
		func(m *Map, k, v string) bool {
			new := m.Delete(k)
			return new == m
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=large.Delete(k) -> new!=large", prop.ForAll(
		func(lm *lmap) bool {
			new := lm.m.Delete(lm.k + strconv.Itoa(lm.num-1))
			return new != lm.m
		},
		genLargeMap,
	))
	properties.Property("new=large.Delete(k) -> new.At(k)==nil && large.At(k)==v", prop.ForAll(
		func(lm *lmap) bool {
			key := lm.k + strconv.Itoa(lm.num-1)
			val := lm.v + strconv.Itoa(lm.num-1)
			new := lm.m.Delete(key)
			return new.At(key) == nil && lm.m.At(key) == val
		},
		genLargeMap,
	))
	properties.Property("new=removeAll(large) -> new.Length()==0", prop.ForAll(
		func(lm *lmap) bool {
			new := lm.m
			for i := 0; i < lm.num; i++ {
				new = new.Delete(lm.k + strconv.Itoa(i))
			}
			return new.Length() == 0
		},
		genLargeMap,
	))
	properties.TestingRun(t)
}

func TestLength(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("new=empty.Assoc(k, v) -> new.Length()==empty.Length()+1", prop.ForAll(
		func(m *Map, k, v string) bool {
			new := m.Assoc(k, v)
			return new.Length() == m.Length()+1
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=large.Assoc(k,v) -> new.Length()==large.Length()+1", prop.ForAll(
		func(lm *lmap, k, v string) bool {
			new := lm.m.Assoc(k, v)
			return new.Length() == lm.m.Length()+1
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("lm.num == lm.m.Length()", prop.ForAll(
		func(lm *lmap) bool {
			return lm.m.Length() == lm.num
		},
		genLargeMap,
	))
	properties.Property("new=empty.Assoc(k, v).Delete(k) -> new.Length()==empty.Length()", prop.ForAll(
		func(m *Map, k, v string) bool {
			new := m.Assoc(k, v).Delete(k)
			return new.Length() == m.Length()
		},
		genMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("new=large.Assoc(k,v).Delete(k) -> new.Length()==large.Length()", prop.ForAll(
		func(lm *lmap, k, v string) bool {
			new := lm.m.Assoc(k, v).Delete(k)
			return new.Length() == lm.m.Length()
		},
		genLargeMap,
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.Property("lm.num == lm.m.Length()", prop.ForAll(
		func(lm *lmap) bool {
			return lm.m.Length() == lm.num
		},
		genLargeMap,
	))
	properties.Property("random.Length() increases correctly", prop.ForAll(
		func(rm *rmap, entries map[string]string) bool {
			m := rm.m
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

func TestAsNative(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("AsNative returns the full map", prop.ForAll(
		func(rm *rmap) bool {
			out := rm.m.AsNative()
			for k, v := range out {
				if v == rm.m.At(k) {
					continue
				}
				return false
			}
			return true
		},
		genRandomMap,
	))
	properties.TestingRun(t)
}

func TestEqual(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("m == m", prop.ForAll(
		func(rm *rmap) bool {
			return rm.m.Equal(rm.m)
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
			new := rm.m.Delete(k)
			return !rm.m.Equal(new)
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() != 0
		}),
	))
	properties.Property("m.Equal(10)==false", prop.ForAll(
		func(rm *rmap) bool {
			return !rm.m.Equal(10)
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
			new := rm.m.Assoc(k, "foo")
			return !rm.m.Equal(new)
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() != 0
		}),
	))
	properties.TestingRun(t)
}

func TestApply(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("dyn.Apply(s, i)==s.At(i)",
		prop.ForAll(
			func(is []int) bool {
				s := From(is)
				return s.At(is[0]) == dyn.Apply(s, is[0])
			},
			gen.SliceOfN(10, gen.Int()),
		))
	properties.TestingRun(t)
}

func TestRange(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Range access the full map", prop.ForAll(
		func(rm *rmap) bool {
			foundAll := true
			rm.m.Range(func(key, val interface{}) bool {
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
			rm.m.Range(func(key, val interface{}) {
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
			rm.m.Range(func(entry Entry) bool {
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
			rm.m.Range(func(entry Entry) {
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
			rm.m.Range(func(key, val string) bool {
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
			rm.m.Range(func(key, val string) {
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

			rm.m.Range(1)
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

			rm.m.Range(func(a, b, c string) {})
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

			rm.m.Range(func(a, b, c string) (d, e bool) { return false, false })
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

			rm.m.Range(func(a, b string) string { return "" })
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

			rm.m.Range(func(a, b int) {})
			return false
		},
		genRandomMap.SuchThat(func(rm *rmap) bool {
			return rm.m.Length() > 0
		}),
	))
	properties.TestingRun(t)
}

func makeMap() *Map {
	return Empty()
}

func unmakeMap(m *Map) {
}

var genMap = gopter.DeriveGen(makeMap, unmakeMap)

type lmap struct {
	num  int
	k, v string
	m    *Map
}

func makeLargeMap(num int, k, v string) *lmap {
	m := Empty()
	for i := 0; i < num; i++ {
		m = m.Assoc(k+strconv.Itoa(i), v+strconv.Itoa(i))
	}
	return &lmap{
		num: num,
		k:   k,
		v:   v,
		m:   m,
	}
}

func unmakeLargeMap(lm *lmap) (num int, k, v string) {
	return lm.num, lm.k, lm.v
}

var genLargeMap = gopter.DeriveGen(makeLargeMap, unmakeLargeMap,
	gen.IntRange(10, 100),
	gen.Identifier(),
	gen.Identifier(),
)

func makeEntry(key, val string) entry {
	return entry{key: key, value: val}
}

func unmakeEntry(e entry) (key, val string) {
	return e.key.(string), e.value.(string)
}

type rmap struct {
	entries map[string]string
	m       *Map
}

func makeRandomMap(entries map[string]string) *rmap {
	m := Empty()
	for key, val := range entries {
		m = m.Assoc(key, val)
	}
	return &rmap{
		entries: entries,
		m:       m,
	}
}

func unmakeRandomMap(r *rmap) map[string]string {
	return r.entries
}

var genRandomMap = gopter.DeriveGen(makeRandomMap, unmakeRandomMap,
	gen.MapOf(gen.Identifier(), gen.Identifier()),
)

func ExampleString() {
	fmt.Println(New("1", "2", "3", "4"))
	// Output: { [1 2] [3 4] }
}

func ExampleSeqString() {
	fmt.Println(New("1", "2", "3", "4").Seq())
	// Output: ([1 2] [3 4])
}

func TestIterator(t *testing.T) {
	m := New(1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7)
	expected := (1 + 2 + 3 + 4 + 5 + 6 + 7) * 2
	iter := m.Iterator()
	var got int
	for iter.HasNext() {
		k, v := iter.Next()
		key, value := k.(int), v.(int)
		got += key
		got += value
	}
	if got != expected {
		t.Fatalf("got %v, expected %v", got, expected)
	}
}
