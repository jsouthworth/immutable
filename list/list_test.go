package list

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

func TestList(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("l=Cons(a,nil) -> l.First()==a and l.Next()==nil",
		prop.ForAll(
			func(a int) bool {
				l := Cons(a, Empty())
				return l.First() == a && l.Next() == nil
			},
			gen.Int(),
		))
	properties.Property("s=Cons(a,Cons(a,nil)) -> s.First()==a and s.Next()!=nil",
		prop.ForAll(
			func(a int) bool {
				l := Cons(a, Cons(a, Empty()))
				return l.First() == a &&
					l.Next() != nil &&
					l.Next().First() == a &&
					l.Next().Next() == nil
			},
			gen.Int(),
		))
	properties.Property("s=Cons(a,nil).Conj(a) -> s.First()==a and s.Next()!=nil",
		prop.ForAll(
			func(a int) bool {
				l := Cons(a, Empty()).Conj(a).(*List)
				return l.First() == a &&
					l.Next() != nil &&
					l.Next().First() == a &&
					l.Next().Next() == nil
			},
			gen.Int(),
		))
	properties.Property("s=Cons(a,nil).Seq() -> s.First()==a and s.Next()==nil",
		prop.ForAll(
			func(a int) bool {
				l := Cons(a, Empty()).Seq()
				return l.First() == a && l.Next() == nil
			},
			gen.Int(),
		))
	properties.Property("s=Cons(a,Cons(a,nil)).Seq() -> s.First()==a and s.Next()!=nil",
		prop.ForAll(
			func(a int) bool {
				l := Cons(a, Cons(a, Empty())).Seq()
				return l.First() == a &&
					l.Next() != nil &&
					l.Next().First() == a &&
					l.Next().Next() == nil
			},
			gen.Int(),
		))
	properties.Property("s=Cons(b,Cons(a,nil)).Find(a) -> a, ok",
		prop.ForAll(
			func(a, b int) bool {
				v, ok := Cons(b, Cons(a, Empty())).Find(a)
				return v == a && ok
			},
			gen.Int(),
			gen.Int(),
		))
	properties.Property("s=Cons(a, nil).Length() == 1",
		prop.ForAll(
			func(a int) bool {
				return Cons(a, nil).Length() == 1
			},
			gen.Int(),
		))
	properties.Property("s=Cons(a, New(xs)) == len(xs) + 1",
		prop.ForAll(
			func(a int, xs []interface{}) bool {
				return Cons(a, New(xs...)).Length() ==
					len(xs)+1
			},
			gen.Int(),
			gen.SliceOf(gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
		))
	properties.Property("New(xs) == New(xs)",
		prop.ForAll(
			func(xs []interface{}) bool {
				return dyn.Equal(New(xs...), New(xs...))
			},
			gen.SliceOf(gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
		))
	properties.Property("New(xs) != New(reverse(xs))",
		prop.ForAll(
			func(xs []interface{}) bool {
				return !dyn.Equal(New(xs...), New(reverse(xs)...))
			},
			gen.SliceOfN(100, gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
		))
	properties.TestingRun(t)
}

func reverse(xs []interface{}) []interface{} {
	for i := len(xs)/2 - 1; i >= 0; i-- {
		opp := len(xs) - 1 - i
		xs[i], xs[opp] = xs[opp], xs[i]
	}
	return xs
}

func TestRange(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Range func(interface{})",
		prop.ForAll(
			func(a int) bool {
				expected := a + a
				l := Cons(a, Cons(a, Empty()))
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
				l := Cons(a, Cons(a, Empty()))
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
				l := Cons(a, Cons(a, Empty()))
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
				l := Cons(a, Cons(a, Empty()))
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
				l := Cons(a, Cons(a, Empty()))
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
				l := Cons(a, Cons(a, Empty()))
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
				l := Cons(a, Cons(a, Empty()))
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
				l := Cons(a, Cons(a, Empty()))
				var got int
				l.Range(a)
				return got == expected
			},
			gen.Int(),
		))
	properties.TestingRun(t)
}

func TestListFrom(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("From([]interface{})",
		prop.ForAll(
			func(xs []interface{}) bool {
				return dyn.Equal(From(xs), New(xs...))
			},
			gen.SliceOf(gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
		))
	properties.Property("From([]int)",
		prop.ForAll(
			func(xs []int) bool {
				l := From(xs)
				for _, v := range xs {
					lv := l.First()
					l = l.Next()
					if lv != v {
						return false
					}
				}
				return true
			},
			gen.SliceOf(gen.Int()),
		))
	properties.Property("From(Seq(xs)))",
		prop.ForAll(
			func(xs []int) bool {
				l := From(seq.Seq(xs))
				for i := len(xs) - 1; i >= 0; i-- {
					v := xs[i]
					lv := l.First()
					l = l.Next()
					if lv != v {
						return false
					}
				}
				return true
			},
			gen.SliceOf(gen.Int()),
		))
	properties.Property("From(New(xs...)))",
		prop.ForAll(
			func(xs []interface{}) bool {
				l := From(New(xs...))
				for _, v := range xs {
					lv := l.First()
					l = l.Next()
					if lv != v {
						return false
					}
				}
				return true
			},
			gen.SliceOf(gen.Int(),
				reflect.TypeOf((*interface{})(nil)).Elem()),
		))
	properties.Property("From(vector.From(xs)))",
		prop.ForAll(
			func(xs []int) bool {
				l := From(vector.From(xs))
				for i := len(xs) - 1; i >= 0; i-- {
					v := xs[i]
					lv := l.First()
					l = l.Next()
					if lv != v {
						return false
					}
				}
				return true
			},
			gen.SliceOf(gen.Int()),
		))
	properties.TestingRun(t)
}

func ExampleString() {
	fmt.Println(New(1, 2, 3, 4, 5, 6))
	// Output: (1 2 3 4 5 6)
}

func ExampleSeqString() {
	fmt.Println(New(1, 2, 3, 4, 5, 6).Seq())
	// Output: (1 2 3 4 5 6)
}
