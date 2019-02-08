package list

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
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
	properties.TestingRun(t)
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
