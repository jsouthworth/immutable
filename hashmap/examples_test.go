package hashmap

import (
	"fmt"

	"jsouthworth.net/go/dyn"
)

func ExampleEmpty() {
	// Empty returns a new empty map with a unique hashseed.
	m := Empty()
	fmt.Println(m)
	// Output: { }
}

func ExampleNew() {
	// New generates pairs from a list of keys and values
	m := New("a", true, "b", false)
	fmt.Println(m)

	// It is equivalent to the following code using go's
	// native map type
	gm := map[string]bool{"a": true, "b": false}
	fmt.Println(gm)
}

func ExampleFrom_map() {
	// From generates a map from several different types.
	// One of these types are go native maps.
	m := From(map[string]bool{"a": true, "b": false})
	fmt.Println(m)
}

type tent struct {
	key   string
	value bool
}

func (t *tent) Key() interface{} {
	return t.key
}
func (t *tent) Value() interface{} {
	return t.value
}
func ExampleFrom_entries() {
	// From generates a map from several different types.
	// One of these types is a slice of entry types.
	sl := []Entry{
		&tent{key: "a", value: false},
		&tent{key: "b", value: true},
	}
	m := From(sl)
	fmt.Println(m)
}

func ExampleFrom_slice() {
	// From generates a map from several different types.
	// One of these types is a slice of arbitray type
	// that has an even number of elements.
	m := From([]int{1, 2, 3, 4})
	fmt.Println(m)
}

func ExampleMap_Assoc() {
	// Assoc is similar to the go builtin m[k]=v operation, except
	// it does not modify the map in place.
	gm := map[string]bool{"a": true, "b": false}
	m := From(gm)

	m = m.Assoc("c", true)
	gm["c"] = true

	fmt.Println(dyn.Equal(m, From(gm)))
	// Output: true
}

func ExampleMap_AsNative() {
	m := New("a", true, "b", false)
	gm := m.AsNative()
	fmt.Printf("%T\n", gm)
	// Output: map[interface {}]interface {}
}

func ExampleMap_At() {
	// At is similar to the go builtin operator m[k].
	gm := map[string]bool{"a": true, "b": false}
	m := From(gm)
	fmt.Println(m.At("a"))
	fmt.Println(gm["a"])
	// Output: true
	// true
}

func ExampleMap_Contains() {
	gm := map[string]bool{"a": true, "b": false}
	m := From(gm)

	fmt.Println(m.Contains("a"))

	_, contains := gm["a"]
	fmt.Println(contains)

	// Output: true
	// true
}

func ExampleMap_Delete() {
	// Delete is similar to the builtin delete function in go,
	// except it does not modify the map in place.
	gm := map[string]bool{"a": true, "b": false}
	m := From(gm)

	m = m.Delete("b")
	delete(gm, "b")

	fmt.Println(dyn.Equal(m, From(gm)))
	// Output: true

}

func ExampleMap_EntryAt() {
	m := New("a", true, "b", false)
	fmt.Println(m.EntryAt("a"))
	// Output: [a true]
}

func ExampleMap_Find() {
	gm := map[string]bool{"a": true, "b": false}
	m := From(gm)

	val, gotIt := m.Find("a")
	fmt.Println(val, gotIt)

	val, contains := gm["a"]
	fmt.Println(val, contains)

	// Output: true true
	// true true
}

func ExampleMap_Length() {
	gm := map[string]bool{"a": true, "b": false}
	m := From(gm)

	fmt.Println(m.Length(), len(gm))
	// Output: 2 2
}

func ExampleMap_Range_entry() {
	// Range is a replacement for go's range builtin
	// it takes several function forms this version
	// will always process all elements.
	m := New("a", true, "b", false, "c", true, "d", false)
	m.Range(func(e Entry) {
		fmt.Println("key", e.Key(), "value", e.Value())
	})
}

func ExampleMap_Range_kv() {
	// Range is a replacement for go's range builtin
	// it takes several function forms this version
	// will always process all elements.
	m := New("a", true, "b", false, "c", true, "d", false)
	m.Range(func(key string, value bool) {
		fmt.Println("key", key, "value", value)
	})
}

func ExampleMap_Range_entry_continue() {
	// Range is a replacement for go's range builtin
	// it takes several function forms this version
	// allows one to stop processing by returning false
	m := New("a", true, "b", false, "c", true, "d", false)
	m.Range(func(e Entry) bool {
		if e.Value().(bool) {
			return false
		}
		fmt.Println("key", e.Key(), "value", e.Value())
		return true
	})
}

func ExampleMap_Range_kv_continue() {
	// Range is a replacement for go's range builtin
	// it takes several function forms this version
	// allows one to stop processing by returning false
	m := New("a", true, "b", false, "c", true, "d", false)
	m.Range(func(key string, value bool) bool {
		if value {
			return false
		}
		fmt.Println("key", key, "value", value)
		return true
	})
}

func ExampleMap_Transform() {
	// Transform allows one to transactionally change a
	// map by going through a transient to make changes
	// this allows for faster large changes in a scoped
	// way.
	m := New("a", true, "b", false, "c", true, "d", false)
	m = m.Transform(func(t *TMap) *TMap {
		return t.Assoc("e", true).Assoc("f", false)
	})
	fmt.Println(m)
}

func ExampleMap_Iterator() {
	// Iterator returns a mutable iterator over the map contents
	m := New("a", true, "b", false, "c", true, "d", false)
	iter := m.Iterator()
	for iter.HasNext() {
		key, value := iter.Next()
		if value.(bool) {
			break
		}
		fmt.Println("key", key, "value", value)
	}
}
