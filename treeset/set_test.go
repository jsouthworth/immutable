package treeset

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/vector"
	"jsouthworth.net/go/seq"
)

func BenchmarkAdd(b *testing.B) {
	s := Empty()
	for i := 0; i < b.N; i++ {
		s = s.Add(i)
	}
}

func BenchmarkTransientAdd(b *testing.B) {
	s := Empty().AsTransient()
	for i := 0; i < b.N; i++ {
		s.Add(i)
	}
}

func BenchmarkDelete(b *testing.B) {
	s := Empty()
	for i := 0; i < b.N; i++ {
		s = s.Add(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s = s.Delete(i)
	}
}

func BenchmarkTransientDelete(b *testing.B) {
	s := Empty().AsTransient()
	for i := 0; i < b.N; i++ {
		s = s.Add(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s = s.Delete(i)
	}
}

func TestSet(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("s=Empty().Add(i)->s.At(i) == i",
		prop.ForAll(
			func(i int) bool {
				s := Empty().Add(i)
				return s.At(i) == i
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i)->s.At(j)==nil",
		prop.ForAll(
			func(i, j int) bool {
				s := Empty().Add(i)
				return i == j || s.At(j) == nil
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i)->s.Contains(i)",
		prop.ForAll(
			func(i int) bool {
				s := Empty().Add(i)
				return s.Contains(i)
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Conj(i)->s.Contains(i)",
		prop.ForAll(
			func(i int) bool {
				s := Empty().Conj(i)
				return s.(*Set).Contains(i)
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i).Find(i) -> i, true",
		prop.ForAll(
			func(i int) bool {
				s := Empty().Add(i)
				_, ok := s.Find(i)
				return ok
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Find(i) -> nil, false",
		prop.ForAll(
			func(i int) bool {
				s := Empty()
				_, ok := s.Find(i)
				return !ok
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i); r=s.Add(i)->r == s",
		prop.ForAll(
			func(i int) bool {
				s := Empty().Add(i)
				r := s.Add(i)
				return s == r
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i); r=s.Add(j)->r != s",
		prop.ForAll(
			func(i, j int) bool {
				s := Empty().Add(i)
				r := s.Add(j)
				return i == j || s != r
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i).Delete(i)->!s.Contains(i)",
		prop.ForAll(
			func(i int) bool {
				s := Empty().Add(i).Delete(i)
				return !s.Contains(i)
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i); r=s.Delete(i)->r != s",
		prop.ForAll(
			func(i int) bool {
				s := Empty().Add(i)
				r := s.Delete(i)
				return r != s
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i).Delete(i); r=s.Delete(i)->r == s",
		prop.ForAll(
			func(i int) bool {
				s := Empty().Add(i).Delete(i)
				r := s.Delete(i)
				return r == s
			},
			gen.Int(),
		))

	properties.Property("Creating a map gives expected length",
		prop.ForAll(
			func(is []int) bool {
				m := make(map[int]struct{})
				s := Empty()
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

func TestRange(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Range func(interface{})",
		prop.ForAll(
			func(a, b int) bool {
				expected := a + b
				l := Empty().Add(a).Add(b)
				var got int
				l.Range(func(i interface{}) {
					got += i.(int)
				})
				return got == expected
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("Range func(interface{}) bool",
		prop.ForAll(
			func(a, b int) bool {
				l := Empty().Add(a).Add(b)
				var got int
				l.Range(func(i interface{}) bool {
					got += i.(int)
					return false
				})
				return got == a || got == b
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("Range func(T)",
		prop.ForAll(
			func(a, b int) bool {
				expected := a + b
				l := Empty().Add(a).Add(b)
				var got int
				l.Range(func(i int) {
					got += i
				})
				return got == expected
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("Range func(T) bool",
		prop.ForAll(
			func(a, b int) bool {
				l := Empty().Add(a).Add(b)
				var got int
				l.Range(func(i int) bool {
					got += i
					return false
				})
				return got == a || got == b
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("Range func(T) T panics",
		prop.ForAll(
			func(a, b int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().Add(a).Add(b)
				var got int
				l.Range(func(i int) int {
					got += i
					return got
				})
				return got == expected
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("Range func(T, T) bool panics",
		prop.ForAll(
			func(a, b int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().Add(a).Add(b)
				var got int
				l.Range(func(i, j int) bool {
					got += i
					return true
				})
				return got == expected
			},

			gen.Int(),
			gen.Int(),
		))
	properties.Property("Range func(T, T) (bool,bool) panics",
		prop.ForAll(
			func(a, b int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().Add(a).Add(b)
				var got int
				l.Range(func(i, j int) (bool, bool) {
					got += i
					return true, false
				})
				return got == expected
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("Range(int) panics",
		prop.ForAll(
			func(a, b int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().Add(a).Add(b)
				var got int
				l.Range(a)
				return got == expected
			},
			gen.Int(),
			gen.Int(),
		))
	properties.TestingRun(t)
}

func TestFrom(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("From(Set) yeilds correct result",
		prop.ForAll(
			func(is []int) bool {
				s := From(is)
				t := From(s)
				return t == s
			},
			gen.SliceOf(gen.Int()),
		))
	properties.Property("From(map[interface{}]struct{}) yeilds correct result",
		prop.ForAll(
			func(is map[string]struct{}) bool {
				in := make(map[interface{}]struct{})
				for k, v := range is {
					in[k] = v
				}
				s := From(in)
				foundAll := true
				s.Range(func(s string) bool {
					if _, ok := in[s]; !ok {
						foundAll = false
						return false
					}
					return true
				})
				return foundAll
			},
			gen.MapOf(gen.Identifier(),
				gen.Struct(reflect.TypeOf(struct{}{}), nil)),
		))
	properties.Property("From([]interface{}) yeilds correct result",
		prop.ForAll(
			func(ss []interface{}) bool {
				set := From(ss)
				for _, s := range ss {
					if !set.Contains(s) {
						return false
					}
				}
				return true
			},
			gen.SliceOf(gen.Identifier(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
		))
	properties.Property("From(seq.Sequence) yeilds correct result",
		prop.ForAll(
			func(ss []interface{}) bool {
				coll := seq.Seq(vector.From(ss))
				set := From(coll)
				for _, s := range ss {
					if !set.Contains(s) {
						return false
					}
				}
				return true
			},
			gen.SliceOf(gen.Identifier(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
		))
	properties.Property("From(seq.Seqable) yeilds correct result",
		prop.ForAll(
			func(ss []interface{}) bool {
				coll := vector.From(ss)
				set := From(coll)
				for _, s := range ss {
					if !set.Contains(s) {
						return false
					}
				}
				return true
			},
			gen.SliceOf(gen.Identifier(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
		))
	properties.Property("From(map[kT]vT) yeilds correct result",
		prop.ForAll(
			func(ss map[string]int) bool {
				set := From(ss)
				for s := range ss {
					if !set.Contains(s) {
						fmt.Println(set, "does not contain", s)
						return false
					}
				}
				return true
			},
			gen.MapOf(gen.Identifier(), gen.Int()),
		))
	properties.Property("From([]T) yeilds correct result",
		prop.ForAll(
			func(ss []int) bool {
				set := From(ss)
				for _, s := range ss {
					if !set.Contains(s) {
						return false
					}
				}
				return true
			},
			gen.SliceOf(gen.Int()),
		))
	properties.Property("From(*TSet) yeilds correct result",
		prop.ForAll(
			func(ss []int) bool {
				tset := From(ss).AsTransient()
				set := From(tset)
				for _, s := range ss {
					if !set.Contains(s) {
						return false
					}
				}
				return true
			},
			gen.SliceOf(gen.Int()),
		))
	properties.Property("From(int) yeilds correct result",
		prop.ForAll(
			func(i int) bool {
				set := From(i)
				return set.Length() == 1 && set.Contains(i)
			},
			gen.Int(),
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

func TestString(t *testing.T) {
	s := New(1, 2, 3)
	str := s.String()
	switch str {
	case "{ 1 2 3 }":
	default:
		t.Fatal("unexpected string", str)

	}
}

func TestSeq(t *testing.T) {
	set := New(1, 2, 3, 4, 5)
	s := set.Seq()
	sum := 0
	for s != nil {
		elem := seq.First(s).(int)
		sum += elem
		s = seq.Seq(seq.Next(s))
	}
	if sum != 15 {
		t.Fatal("Seq didn't traverse all the elements of the set")
	}
}

func TestSeqString(t *testing.T) {
	set := New(1, 2, 3, 4, 5)
	s := set.Seq()
	str := fmt.Sprint(s)
	exp := "(1 2 3 4 5)"
	if str != exp {
		t.Fatalf("Seq didn't produce the correct string: got: %s exp: %s", str, exp)

	}
}

func TestSeqEmpty(t *testing.T) {
	set := Empty()
	s := set.Seq()
	if s != nil {
		t.Fatal("Seq should have been nil")
	}
}

func TestEqual(t *testing.T) {
	t.Run("equal", func(t *testing.T) {
		s1 := New(1, 2, 3)
		s2 := New(1, 2, 3)
		if !s1.Equal(s2) {
			t.Fatal("Sets should have been equal")
		}
	})
	t.Run("different-types", func(t *testing.T) {
		s1 := New(1, 2, 3)
		s2 := New(1, 2).AsTransient()
		if s1.Equal(s2) {
			t.Fatal("Sets should not have been equal")
		}
	})
	t.Run("different-lengths", func(t *testing.T) {
		s1 := New(1, 2, 3)
		s2 := New(1, 2)
		if s1.Equal(s2) {
			t.Fatal("Sets should not have been equal")
		}
	})
	t.Run("different-values", func(t *testing.T) {
		s1 := New(1, 2, 3)
		s2 := New(1, 2, 5)
		if s1.Equal(s2) {
			t.Fatal("Sets should not have been equal")
		}
	})
}

func TestTransform(t *testing.T) {
	s1 := New(1, 2, 3, 4, 5, 6)
	s2 := Empty().Transform(func(s *TSet) {
		s.Add(1).Add(2).Add(3)
	}, func(s *TSet) {
		s.Add(4).Add(5).Add(6)
	})
	if !s1.Equal(s2) {
		t.Fatal("Sets should have been equal")
	}
}

func TestCustomComparator(t *testing.T) {
	s1 := Empty(Compare(func(a, b interface{}) int {
		ai, aok := a.(int)
		bi, bok := b.(int)
		if !aok || !bok {
			return -1
		}
		switch {
		case ai > bi:
			return 1
		case ai < bi:
			return -1
		default:
			return 0
		}
	}))
	s1 = s1.Transform(func(s *TSet) {
		s.Add(1).Add(2).Add(3).Add(4)
	})
	s2 := New(1, 2, 3, 5)
	if s1.Equal(s2) {
		t.Fatal("Sets should not have been equal")
	}
}
