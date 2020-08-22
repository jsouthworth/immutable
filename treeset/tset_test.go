package treeset

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"jsouthworth.net/go/dyn"
)

func TestTransientSet(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("s=Empty().Add(i)->s.At(i) == i",
		prop.ForAll(
			func(i int) bool {
				s := Empty().AsTransient().Add(i)
				return s.At(i) == i
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i)->s.At(j)==nil",
		prop.ForAll(
			func(i, j int) bool {
				s := Empty().AsTransient().Add(i)
				return i == j || s.At(j) == nil
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i)->s.Contains(i)",
		prop.ForAll(
			func(i int) bool {
				s := Empty().AsTransient().Add(i)
				return s.Contains(i)
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Conj(i)->s.Contains(i)",
		prop.ForAll(
			func(i int) bool {
				s := Empty().AsTransient().Conj(i)
				return s.(*TSet).Contains(i)
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i).Find(i) -> i, true",
		prop.ForAll(
			func(i int) bool {
				s := Empty().AsTransient().Add(i)
				_, ok := s.Find(i)
				return ok
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Find(i) -> nil, false",
		prop.ForAll(
			func(i int) bool {
				s := Empty().AsTransient()
				_, ok := s.Find(i)
				return !ok
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i); r=s.Add(i)->r == s",
		prop.ForAll(
			func(i int) bool {
				s := Empty().AsTransient().Add(i)
				r := s.Add(i)
				return s == r
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i).Delete(i)->!s.Contains(i)",
		prop.ForAll(
			func(i int) bool {
				s := Empty().AsTransient().Add(i).Delete(i)
				return !s.Contains(i)
			},
			gen.Int(),
		))
	properties.Property("s=Empty().Add(i).Delete(i); r=s.Delete(i)->r == s",
		prop.ForAll(
			func(i int) bool {
				s := Empty().AsTransient().Add(i).Delete(i)
				r := s.Delete(i)
				return r == s
			},
			gen.Int(),
		))

	properties.Property("Creating a map gives expected length",
		prop.ForAll(
			func(is []int) bool {
				m := make(map[int]struct{})
				s := Empty().AsTransient()
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

func TestTransientRange(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Range func(interface{})",
		prop.ForAll(
			func(a, b int) bool {
				expected := a + b
				l := Empty().AsTransient().Add(a).Add(b)
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
				l := Empty().AsTransient().Add(a).Add(b)
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
				l := Empty().AsTransient().Add(a).Add(b)
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
				l := Empty().AsTransient().Add(a).Add(b)
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
				l := Empty().AsTransient().Add(a).Add(b)
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
				l := Empty().AsTransient().Add(a).Add(b)
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
				l := Empty().AsTransient().Add(a).Add(b)
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
				l := Empty().AsTransient().Add(a).Add(b)
				var got int
				l.Range(a)
				return got == expected
			},
			gen.Int(),
			gen.Int(),
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

func TestTransientString(t *testing.T) {
	s := New(1, 2, 3).AsTransient()
	str := s.String()
	switch str {
	case "{ 1 2 3 }":
	default:
		t.Fatal("unexpected string", str)

	}
}

func TestTransientEqual(t *testing.T) {
	s1 := New(1, 2, 3).AsTransient()
	s2 := New(1, 2, 3).AsTransient()
	if !s1.Equal(s2) {
		t.Fatal("Sets should have been equal")
	}
	if s1.Equal(10) {
		t.Fatal("Set should not have been equal to an int")
	}
}
