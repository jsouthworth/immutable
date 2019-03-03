package vector

import (
	"fmt"

	"jsouthworth.net/go/immutable/list"
	"jsouthworth.net/go/seq"
)

func ExampleEmpty() {
	// Empty returns an empty vector. This is
	// always the same empty vector.
	v := Empty()
	fmt.Println(v)
	// Output: []
}

func ExampleNew() {
	// New allows one to create a vector similar to how
	// one defines a slice inline in go.
	v := New(1, 2, 3, 4)
	s := []int{1, 2, 3, 4}
	fmt.Println(v)
	fmt.Println(s)
	// Output: [1 2 3 4]
	// [1 2 3 4]
}

func ExampleFrom_slice() {
	// From allows one to create a vectore from a go
	// slice.
	s := []int{1, 2, 3, 4}
	v := From(s)
	fmt.Println(v)
	// Output: [1 2 3 4]
}

func ExampleFrom_sequence() {
	// From allows one to create a vectore from a
	// seq.Sequence type
	lseq := seq.Seq(list.New(1, 2, 3, 4))
	v := From(lseq)
	fmt.Println(v)
	// Output: [1 2 3 4]
}

func ExampleFrom_seqable() {
	// From allows one to create a vectore from a
	// seq.Sequable type
	l := list.New(1, 2, 3, 4)
	v := From(l)
	fmt.Println(v)
	// Output: [1 2 3 4]
}

func ExampleVector_Append() {
	// Append adds a new element to the end of a vector.
	// it is equivalent to the go append function.
	v := Empty().Append(1)
	s := append([]int{}, 1)
	fmt.Println(v)
	fmt.Println(s)
	// Output: [1]
	// [1]
}

func ExampleVector_AsNative() {
	// AsNative converts the vector to a []interace{}
	v := New(1, 2, 3, 4, 5)
	s := v.AsNative()
	fmt.Printf("%T %v\n", s, s)
	// Output: []interface {} [1 2 3 4 5]
}

func ExampleVector_Assoc() {
	// Assoc associates a value with an index. This is similar to
	// go's s[i] = v operator except that the vector is not modified
	// in place.
	v := New(1, 2, 3, 4)
	v = v.Assoc(0, 10)

	s := []int{1, 2, 3, 4}
	s[0] = 10

	fmt.Println(v)
	fmt.Println(s)

	// Output: [10 2 3 4]
	// [10 2 3 4]
}

func ExampleVector_At() {
	// At returns the value at the index. This is similar to go's
	// s[i] operator.
	v := New(1, 2, 3, 4)
	fmt.Println(v.At(2))

	s := []int{1, 2, 3, 4}
	fmt.Println(s[2])
	// Output: 3
	// 3
}

func ExampleVector_Delete() {
	// Delete removes the item at an index and shifs the other items
	// down by one. This is similar to the delete from a slice pattern.
	v := New(1, 2, 3, 4)
	v = v.Delete(2)
	fmt.Println(v)

	s := []int{1, 2, 3, 4}
	s = append(s[:2], s[3:]...)
	fmt.Println(s)
	// Output: [1 2 4]
	// [1 2 4]
}

func ExampleVector_Insert() {
	// Insert adds an item at the index and shifts the others up by one.
	v := New(1, 2, 3, 4)
	v = v.Insert(2, 10)
	fmt.Println(v)
	// Output: [1 2 10 3 4]
}

func ExampleVector_Length() {
	// Length returns the length of the vector and is equivalent to
	// the go len function.
	v := New(1, 2, 3, 4)
	s := []int{1, 2, 3, 4}
	fmt.Println(v.Length(), len(s))
	// Output: 4 4
}

func ExampleVector_Pop() {
	v := New(1, 2, 3, 4)
	v = v.Pop()
	fmt.Println(v)
	// Output: [1 2 3]
}

func ExampleVector_Range_continue() {
	// Range is a replacement for go's range builtin
	// it takes several function forms this version
	// allows one to stop processing by returning false
	v := New(1, 2, 3, 4)
	v.Range(func(index, value int) bool {
		if value == 3 {
			return false
		}
		fmt.Println(index, value)
		return true
	})
	// Output: 0 1
	// 1 2
}

func ExampleVector_Range_all() {
	// Range is a replacement for go's range builtin
	// it takes several function forms this version
	// will always process all elements.
	v := New(1, 2, 3, 4)
	v.Range(func(index, value int) {
		fmt.Println(index, value)
	})
	// Output: 0 1
	// 1 2
	// 2 3
	// 3 4
}

func ExampleVector_Slice() {
	// Slice allows one to slice a vector this is similar to go's
	// slice operators with some caveats. The slice shares structure
	// with the original vector but a modification operation on the
	// slice will not effect the original vector only the slices'
	// backing copy.
	v := New(1, 2, 3, 4)
	s := v.Slice(1, 4)
	fmt.Println(s)
	// Output: [2 3 4]
}

func ExampleVector_Transform() {
	// Transform allows one to transactionally change a
	// vector by going through a transient to make changes
	// this allows for faster large changes in a scoped
	// way.
	v := New(1, 2, 3, 4)
	v = v.Transform(func(t *TVector) *TVector {
		return t.Append(5).Append(6).Append(7)
	})
	fmt.Println(v)
	// Output: [1 2 3 4 5 6 7]
}
