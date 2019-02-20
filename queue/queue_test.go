package queue

import (
	"fmt"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"jsouthworth.net/go/immutable/vector"
	"jsouthworth.net/go/seq"
)

func TestQueuePushPop(t *testing.T) {
	q := New(1, 2, 3)
	q = q.Push(4)
	for i := 0; i < 4; i++ {
		if q.First() != i+1 {
			t.Fatal("didn't get expected queue")
		}
		q = q.Pop()
	}
	if q.Length() != 0 {
		t.Fatal("pop didn't remove all elements")
	}
}

func TestQueueFrom(t *testing.T) {
	t.Run("*Queue", func(t *testing.T) {
		q := New(1, 2, 3)
		q2 := From(q)
		if q != q2 {
			t.Fatal("from didn't return the same queue")
		}
	})
	t.Run("nil", func(t *testing.T) {
		q := From(nil)
		if q.Length() != 0 {
			t.Fatal("didn't get expected queue")
		}
	})
	t.Run("[]interface{}", func(t *testing.T) {
		q := From([]interface{}{1, 2, 3})
		if q.First() != 1 {
			t.Fatal("from didn't didn't create the right queue")
		}
	})
	t.Run("Seqable", func(t *testing.T) {
		q := From(vector.New(1, 2, 3))
		for i := 0; i < 3; i++ {
			if q.First() != i+1 {
				t.Fatal("didn't get expected queue")
			}
			q = q.Pop()
		}
	})
	t.Run("Sequence", func(t *testing.T) {
		q := From(seq.Cons(1, seq.Cons(2, seq.Cons(3, nil))))
		for i := 0; i < 3; i++ {
			if q.First() != i+1 {
				t.Fatal("didn't get expected queue")
			}
			q = q.Pop()
		}
	})
	t.Run("Other", func(t *testing.T) {
		defer func() {
			e := recover().(error)
			if e == nil {
				t.Fatal("didn't get expected error")
			}
		}()
		From(1)
	})
}

func TestQueueLength(t *testing.T) {
	q := New(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	if q.Length() != 10 {
		t.Fatal("queue.Length didn't return expected value")
	}
}

func TestQueueFirst(t *testing.T) {
	q := New(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	if q.First() != 1 {
		t.Fatal("peek didn't return first element")
	}
	q = q.Pop()
	if q.First() != 2 {
		t.Fatal("peek didn't return first element")
	}
}

func TestQueueSeq(t *testing.T) {
	result := seq.Reduce(func(result, input interface{}) interface{} {
		return result.(int) + input.(int)
	}, 0, New(1, 2, 3).Seq())
	if result != 6 {
		t.Fatal("didn't get the expected result from reduce")
	}
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

func ExampleString() {
	fmt.Println(New(1, 2, 3, 4, 5, 6))
	// Output: [ 1 2 3 4 5 6 ]
}

func ExampleSeqString() {
	fmt.Println(New(1, 2, 3, 4, 5, 6).Seq())
	// Output: (1 2 3 4 5 6)
}
