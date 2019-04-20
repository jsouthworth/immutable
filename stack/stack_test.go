package stack

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"jsouthworth.net/go/seq"
)

func TestStack(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("s=Empty().Push(a) -> s.Top()==a and s.Pop()==Empty()",
		prop.ForAll(
			func(a int) bool {
				s := Empty().Push(a)
				return s.Top() == a && s.Pop() == Empty()
			},
			gen.Int(),
		))
	properties.Property("s=New(is).Push(a) -> s.Top()==a)",
		prop.ForAll(
			func(as []interface{}, a int) bool {
				s := New(as...).Push(a)
				return s.Top() == a
			},
			gen.SliceOf(gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
			gen.Int(),
		))
	properties.Property("s=New(is).Push(a) -> s.Pop() != Empty())",
		prop.ForAll(
			func(as []interface{}, a int) bool {
				s := New(as...).Push(a)
				return s.Pop() != Empty()
			},
			gen.SliceOfN(10, gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
			gen.Int(),
		))
	properties.Property("s=New(is).Conj(a) -> s.Pop() != Empty())",
		prop.ForAll(
			func(as []interface{}, a int) bool {
				s := New(as...).Conj(a).(*Stack)
				return s.Pop() != Empty()
			},
			gen.SliceOfN(10, gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
			gen.Int(),
		))
	properties.Property("s=From(is).Push(a) -> s.Pop() != Empty())",
		prop.ForAll(
			func(as []interface{}, a int) bool {
				s := From(as).Push(a)
				return s.Pop() != Empty()
			},
			gen.SliceOfN(10, gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
			gen.Int(),
		))
	properties.Property("s=From(empty).Push(a) -> s.Pop() == Empty())",
		prop.ForAll(
			func(as []interface{}, a int) bool {
				s := From(as).Push(a)
				return s.Pop() == Empty()
			},
			gen.SliceOfN(0, gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
			gen.Int(),
		))
	properties.Property("s=From(empty).Push(a).Find(a) == a, true)",
		prop.ForAll(
			func(as []interface{}, a int) bool {
				v, ok := From(as).Push(a).Find(a)
				return v == a && ok
			},
			gen.SliceOfN(0, gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
			gen.Int(),
		))
	properties.TestingRun(t)
}

func TestString(t *testing.T) {
	s := Empty().Push(1).Push(2).Push(3)
	str := s.String()
	expected := "[ 3 2 1 ]"
	if str != expected {
		t.Fatalf("got %s, expected %s", str, expected)
	}
	tr := Empty().AsTransient().Push(1).Push(2).Push(3)
	str = tr.String()
	if str != expected {
		t.Fatalf("transient: got %s, expected %s", str, expected)
	}
}

func TestSeq(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Reduce iterates all elements",
		prop.ForAll(
			func(as []int) bool {
				var expected int
				for _, a := range as {
					expected += a
				}
				s := From(as)
				got := seq.Reduce(func(res, in int) int {
					return res + in
				}, 0, seq.Seq(s)).(int)

				return got == expected
			},
			gen.SliceOfN(10, gen.Int()),
		))
	properties.TestingRun(t)
}

func TestTStack(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("s=Empty().Push(a) -> s.Top()==a and s.Pop()==Empty()",
		prop.ForAll(
			func(a int) bool {
				s := Empty().AsTransient().Push(a)
				return s.Top() == a && s.Pop().AsPersistent() == Empty()
			},
			gen.Int(),
		))
	properties.Property("s=New(is).Push(a) -> s.Top()==a)",
		prop.ForAll(
			func(as []interface{}, a int) bool {
				s := New(as...).AsTransient().Push(a)
				return s.Top() == a
			},
			gen.SliceOf(gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
			gen.Int(),
		))
	properties.Property("s=New(is).Push(a) -> s.Pop() != Empty())",
		prop.ForAll(
			func(as []interface{}, a int) bool {
				s := New(as...).AsTransient().Push(a)
				return s.Pop().AsPersistent() != Empty()
			},
			gen.SliceOfN(10, gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
			gen.Int(),
		))
	properties.Property("s=From(is).Push(a).Find(a) == a, true)",
		prop.ForAll(
			func(as []interface{}, a int) bool {
				t := From(as).AsTransient().Push(a)
				v, ok := t.Find(a)
				return v == a && ok
			},
			gen.SliceOfN(0, gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
			gen.Int(),
		))
	properties.TestingRun(t)
}

func TestRange(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Range func(interface{})",
		prop.ForAll(
			func(a int) bool {
				expected := a + a
				l := Empty().Push(a).Push(a)
				var got int
				l.Range(func(i interface{}) {
					got += i.(int)
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(interface{}) bool",
		prop.ForAll(
			func(a int) bool {
				expected := a
				l := Empty().Push(a).Push(a)
				var got int
				l.Range(func(i interface{}) bool {
					got += i.(int)
					return false
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T)",
		prop.ForAll(
			func(a int) bool {
				expected := a + a
				l := Empty().Push(a).Push(a)
				var got int
				l.Range(func(i int) {
					got += i
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T) bool",
		prop.ForAll(
			func(a int) bool {
				expected := a

				l := Empty().Push(a).Push(a)
				var got int
				l.Range(func(i int) bool {
					got += i
					return false
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T) T panics",
		prop.ForAll(
			func(a int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().Push(a).Push(a)
				var got int
				l.Range(func(i int) int {
					got += i
					return got
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T, T) bool panics",
		prop.ForAll(
			func(a int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().Push(a).Push(a)
				var got int
				l.Range(func(i, j int) bool {
					got += i
					return true
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T, T) (bool,bool) panics",
		prop.ForAll(
			func(a int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().Push(a).Push(a)
				var got int
				l.Range(func(i, j int) (bool, bool) {
					got += i
					return true, false
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range(int) panics",
		prop.ForAll(
			func(a int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().Push(a).Push(a)
				var got int
				l.Range(a)
				return got == expected
			},
			gen.Int(),
		))
	properties.TestingRun(t)
}

func TestTransientRange(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Range func(interface{})",
		prop.ForAll(
			func(a int) bool {
				expected := a + a
				l := Empty().AsTransient().Push(a).Push(a)
				var got int
				l.Range(func(i interface{}) {
					got += i.(int)
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(interface{}) bool",
		prop.ForAll(
			func(a int) bool {
				expected := a
				l := Empty().AsTransient().Push(a).Push(a)
				var got int
				l.Range(func(i interface{}) bool {
					got += i.(int)
					return false
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T)",
		prop.ForAll(
			func(a int) bool {
				expected := a + a
				l := Empty().AsTransient().Push(a).Push(a)
				var got int
				l.Range(func(i int) {
					got += i
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T) bool",
		prop.ForAll(
			func(a int) bool {
				expected := a

				l := Empty().AsTransient().Push(a).Push(a)
				var got int
				l.Range(func(i int) bool {
					got += i
					return false
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T) T panics",
		prop.ForAll(
			func(a int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().AsTransient().Push(a).Push(a)
				var got int
				l.Range(func(i int) int {
					got += i
					return got
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T, T) bool panics",
		prop.ForAll(
			func(a int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().AsTransient().Push(a).Push(a)
				var got int
				l.Range(func(i, j int) bool {
					got += i
					return true
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range func(T, T) (bool,bool) panics",
		prop.ForAll(
			func(a int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().AsTransient().Push(a).Push(a)
				var got int
				l.Range(func(i, j int) (bool, bool) {
					got += i
					return true, false
				})
				return got == expected
			},
			gen.Int(),
		))
	properties.Property("Range(int) panics",
		prop.ForAll(
			func(a int) (ok bool) {
				defer func() {
					r := recover()
					ok = r == errRangeSig
				}()
				expected := a
				l := Empty().AsTransient().Push(a).Push(a)
				var got int
				l.Range(a)
				return got == expected
			},
			gen.Int(),
		))
	properties.TestingRun(t)
}

func ExampleString() {
	fmt.Println(New(1, 2, 3, 4))
	// Output: [ 4 3 2 1 ]
}

func ExampleSeqString() {
	fmt.Println(New(1, 2, 3, 4).Seq())
	// Output: (4 3 2 1)
}

func ExampleTransform() {
	fmt.Println(Empty().Transform(
		func(in *TStack) *TStack {
			return in.Push(1).Push(2).Push(3)
		},
	))
	// Output: [ 3 2 1 ]
}

func TestEqual(t *testing.T) {
	s1 := New(1, 2, 3)
	s2 := New(1, 2, 3)
	if !s1.Equal(s2) {
		t.Fatal("Stacks should have been equal")
	}
	if s1.Equal(10) {
		t.Fatal("Stack should not have been equal to an int")
	}
}

func TestStackLength(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if Empty().Length() != 0 {
			t.Fatal("empty should have zero length")
		}
	})
	t.Run("small", func(t *testing.T) {
		if New(1, 2, 3, 4, 5).Length() != 5 {
			t.Fatal("length should be 5")
		}
	})
	t.Run("transient", func(t *testing.T) {
		if New(1, 2, 3, 4, 5).AsTransient().Length() != 5 {
			t.Fatal("length should be 5")
		}
	})
}

func ExampleStack_reverse() {
	s := New(1, 2, 3, 4, 5)
	fmt.Println(s)
	fmt.Println(s.Reverse())
	// Output: [ 5 4 3 2 1 ]
	// [1 2 3 4 5]
}
