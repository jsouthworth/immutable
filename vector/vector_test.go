package vector

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

func BenchmarkSliceAppend(b *testing.B) {
	b.ReportAllocs()
	v := []interface{}{}
	for i := 0; i < b.N; i++ {
		v = append(v, i)
	}
}

func BenchmarkSliceAlloc(b *testing.B) {
	b.ReportAllocs()
	v := make([]interface{}, 0, b.N)
	for i := 0; i < b.N; i++ {
		v = append(v, i)
	}
}

func BenchmarkSliceAlloc2(b *testing.B) {
	b.ReportAllocs()
	v := make([]interface{}, b.N)
	for i := 0; i < b.N; i++ {
		v[i] = i
	}
}

func BenchmarkVectorAppend(b *testing.B) {
	b.ReportAllocs()
	v := Empty()
	for i := 0; i < b.N; i++ {
		v = v.Append(i)
	}
}

func BenchmarkTVectorAppend(b *testing.B) {
	b.ReportAllocs()
	v := Empty().AsTransient()
	for i := 0; i < b.N; i++ {
		v = v.Append(i)
	}
}

func BenchmarkVectorAt(b *testing.B) {
	b.ReportAllocs()
	v := New(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	for i := 0; i < b.N; i++ {
		v.At(i % v.Length())
	}
}

func BenchmarkTVectorAt(b *testing.B) {
	b.ReportAllocs()
	v := New(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).AsTransient()
	for i := 0; i < b.N; i++ {
		v.At(i % v.Length())
	}
}

func BenchmarkSliceAt(b *testing.B) {
	b.ReportAllocs()
	v := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for i := 0; i < b.N; i++ {
		_ = v[i%len(v)]
	}
}

func BenchmarkAssoc(b *testing.B) {
	b.ReportAllocs()
	v := New(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	for i := 0; i < b.N; i++ {
		v.Assoc(i%v.Length(), 10)
	}
}

func BenchmarkTVectorAssoc(b *testing.B) {
	b.ReportAllocs()
	v := New(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).AsTransient()
	for i := 0; i < b.N; i++ {
		v.Assoc(i%v.Length(), 10)
	}
}

func BenchmarkSliceAssoc(b *testing.B) {
	b.ReportAllocs()
	v := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for i := 0; i < b.N; i++ {
		v[i%len(v)] = 10
	}
}

func TestSpeed(t *testing.T) {
	start := time.Now()
	v := Empty()
	for i := 0; i < 1000000; i++ {
		v = v.Append(i)
	}
	t.Log(time.Since(start))
}

func TestTVectorSpeed(t *testing.T) {
	start := time.Now()
	v := Empty().AsTransient()
	for i := 0; i < 1000000; i++ {
		v = v.Append(i)
	}
	t.Log(time.Since(start))
}

type testPvector struct {
	*Vector
}

func (v *testPvector) Generate(rand *rand.Rand, size int) reflect.Value {
	numElems := rand.Intn(size + 10000)
	elems := make([]interface{}, 0, numElems)
	for i := 0; i < numElems; i++ {
		elems = append(elems, rand.Intn(size))
	}
	return reflect.ValueOf(&testPvector{New(elems...)})
}

func (v *testPvector) String() string {
	return v.String()
}

func TestVectorEqual(t *testing.T) {
	f := func(vec *testPvector) bool {
		return vec.Equal(vec.Vector)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorEqualAfterAssoc(t *testing.T) {
	f := func(vec *testPvector) bool {
		vec2 := vec.Assoc(0, 100)
		return !vec.Equal(vec2)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorAppendPreservesPrevious(t *testing.T) {
	f := func(vec *testPvector, insertElems []int) bool {
		//TODO: use Equivalent instead of stringifying the vector
		old := vec.Vector
		orig := fmt.Sprint(old)
		newvec := old
		for _, elem := range insertElems {
			newvec = newvec.Append(elem)
		}
		cur := fmt.Sprint(old)
		return orig == cur
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorAppendAppends(t *testing.T) {
	f := func(vec *testPvector, insertElems []int) bool {
		old := vec.Vector
		newvec := old
		for _, elem := range insertElems {
			newvec = newvec.Append(elem)
		}
		if len(insertElems) == 0 {
			if old.Length() != newvec.Length() {
				return false
			}
			for i := 0; i < old.Length(); i++ {
				if old.At(i).(int) != newvec.At(i).(int) {
					return false
				}
			}
			return true
		}
		if old.Length() == newvec.Length() {
			return false
		}
		for i := 0; i < old.Length(); i++ {
			if old.At(i).(int) != newvec.At(i).(int) {
				return false
			}
		}
		for i := 0; i < len(insertElems); i++ {
			if newvec.At(i+old.Length()) != insertElems[i] {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorAppendIncrementsLength(t *testing.T) {
	f := func(vec *testPvector, insertElems []int) bool {
		old := vec.Vector
		newvec := old
		for _, elem := range insertElems {
			newvec = newvec.Append(elem)
		}
		return old.Length()+len(insertElems) == newvec.Length()
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorAssocPreservesPrevious(t *testing.T) {
	f := func(vec *testPvector, elem int) bool {
		old := vec.Vector
		orig := fmt.Sprint(old)
		if old.Length() < 2 {
			return true
		}
		idx := rand.Intn(old.Length() - 1)
		newvec := old.Assoc(idx, elem)
		cur := fmt.Sprint(old)
		new := fmt.Sprint(newvec)
		return orig == cur && new != orig
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorAssocUpdatesCorrectValue(t *testing.T) {
	f := func(vec *testPvector, elem int) bool {
		old := vec.Vector
		if old.Length() < 2 {
			return true
		}
		idx := rand.Intn(old.Length() - 1)
		newvec := old.Assoc(idx, elem)
		return newvec.At(idx) == elem
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorPopPreservesPrevious(t *testing.T) {
	f := func(vec *testPvector, elem int) bool {
		old := vec.Vector
		orig := fmt.Sprint(old)
		if old.Length() == 0 {
			return true
		}
		newvec := old.Pop()
		cur := fmt.Sprint(old)
		new := fmt.Sprint(newvec)
		return orig == cur && new != orig
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorPopDecrementsLength(t *testing.T) {
	f := func(vec *testPvector, n uint16) bool {
		old := vec.Vector
		newvec := old
		num := int(n)
		if num > newvec.Length() {
			num = newvec.Length()
		}
		for i := 0; i < num; i++ {
			newvec = newvec.Pop()
		}
		return old.Length()-num == newvec.Length()
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorPopAll(t *testing.T) {
	tv := Empty().AsTransient()
	for i := 0; i < 1000000; i++ {
		tv = tv.Append(i)
	}
	v := tv.AsPersistent()
	newvec := v
	for i := 0; i < v.Length()-1; i++ {
		newvec = newvec.Pop()
	}
	if newvec.Length() != 1 && newvec.At(0) == 0 && newvec.shift == bits {
		t.Fatal("unexpected element in vector", newvec)
	}
}

func TestTVectorAssocUpdatesCorrectValue(t *testing.T) {
	f := func(vec *testPvector, elem int) bool {
		old := vec.Vector.AsTransient()
		if old.Length() < 2 {
			return true
		}
		idx := rand.Intn(old.Length() - 1)
		newvec := old.Assoc(idx, elem)
		return newvec.At(idx) == elem
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestTVectorAssocPreservesPrevious(t *testing.T) {
	f := func(vec *testPvector, elem int) bool {
		old := vec.Vector
		orig := fmt.Sprint(old)
		if old.Length() < 2 {
			return true
		}
		idx := rand.Intn(old.Length() - 1)
		newvec := old.AsTransient()
		newvec = newvec.Assoc(idx, elem)
		cur := fmt.Sprint(old)
		new := fmt.Sprint(newvec)
		return orig == cur && new != orig
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestTVectorAppendPreservesPrevious(t *testing.T) {
	f := func(vec *testPvector, insertElems []int) bool {
		//TODO: use Equivalent instead of stringifying the vector
		old := vec.Vector
		orig := fmt.Sprint(old)
		newvec := old.AsTransient()
		for _, elem := range insertElems {
			newvec = newvec.Append(elem)
		}
		cur := fmt.Sprint(old)
		return orig == cur
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestTVectorAppendAppends(t *testing.T) {
	f := func(vec *testPvector, insertElems []int) bool {
		old := vec.Vector
		newvec := old.AsTransient()
		for _, elem := range insertElems {
			newvec = newvec.Append(elem)
		}
		if len(insertElems) == 0 {
			if old.Length() != newvec.Length() {
				return false
			}
			for i := 0; i < old.Length(); i++ {
				if old.At(i).(int) != newvec.At(i).(int) {
					return false
				}
			}
			return true
		}
		if old.Length() == newvec.Length() {
			return false
		}
		for i := 0; i < old.Length(); i++ {
			if old.At(i).(int) != newvec.At(i).(int) {
				return false
			}
		}
		for i := 0; i < len(insertElems); i++ {
			if newvec.At(i+old.Length()) != insertElems[i] {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestTVectorAppendIncrementsLength(t *testing.T) {
	f := func(vec *testPvector, insertElems []int) bool {
		old := vec.Vector
		newvec := old.AsTransient()
		for _, elem := range insertElems {
			newvec = newvec.Append(elem)
		}
		return old.Length()+len(insertElems) == newvec.Length()
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVector2TVector(t *testing.T) {
	f := func(vec *testPvector) bool {
		old := vec.Vector
		newvec := old.AsTransient()
		for i := 0; i < old.Length(); i++ {
			if old.At(i) != newvec.At(i) {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestTVector2Vector(t *testing.T) {
	f := func(vec *testPvector) bool {
		old := vec.Vector
		newtvec := old.AsTransient()
		newvec := newtvec.AsPersistent()
		for i := 0; i < old.Length(); i++ {
			if old.At(i) != newvec.At(i) {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestTVectorPopPreservesPrevious(t *testing.T) {
	f := func(vec *testPvector, elem int) bool {
		old := vec.Vector
		orig := fmt.Sprint(old)
		tvec := old.AsTransient()
		if tvec.Length() == 0 {
			return true
		}
		newvec := tvec.Pop()
		cur := fmt.Sprint(old)
		new := fmt.Sprint(newvec)
		return orig == cur && new != orig
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestTVectorPopDecrementsLength(t *testing.T) {
	f := func(vec *testPvector, n uint16) bool {
		old := vec.Vector
		newvec := old.AsTransient()
		num := int(n)
		if num > newvec.Length() {
			num = newvec.Length()
		}
		for i := 0; i < num; i++ {
			newvec = newvec.Pop()
		}
		return old.Length()-num == newvec.Length()
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestTVectorPopAll(t *testing.T) {
	tv := Empty().AsTransient()
	for i := 0; i < 1000000; i++ {
		tv = tv.Append(i)
	}
	//v := tv.AsPersistent()
	v := tv
	newvec := v
	for i := 0; i < v.Length()-1; i++ {
		newvec = newvec.Pop()
	}
	if newvec.Length() != 1 && newvec.At(0) == 0 && newvec.shift == bits {
		t.Fatal("unexpected element in vector", newvec)
	}
}

func TestVectorFromVector(t *testing.T) {
	f := func(vec *testPvector) bool {
		vec2 := From(vec.Vector)
		return vec2.Equal(vec.Vector)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorFromSequence(t *testing.T) {
	f := func(vec *testPvector) bool {
		vec2 := From(vec.Vector.Seq())
		return vec2.Equal(vec.Vector)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorFromInterfaceSlice(t *testing.T) {
	f := func(ivec []int) bool {
		vec := make([]interface{}, len(ivec))
		for i, v := range ivec {
			vec[i] = v
		}
		vec2 := From(vec)
		if vec2.Length() != len(vec) {
			return false
		}
		for i := 0; i < vec2.Length(); i++ {
			v := vec2.At(i)
			if vec[i] != v {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorFromIntSlice(t *testing.T) {
	f := func(ivec []int) bool {
		vec := From(ivec)
		for i, v := range ivec {
			if vec.At(i) != v {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorFromStringSlice(t *testing.T) {
	f := func(ivec []string) bool {
		vec := From(ivec)
		for i, v := range ivec {
			if vec.At(i) != v {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorAsSlice(t *testing.T) {
	f := func(ivec []string) bool {
		vec := From(ivec).AsNative()
		for i, v := range ivec {
			if vec[i] != v {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorSliceLength(t *testing.T) {
	f := func(ivec []string) bool {
		if len(ivec) < 2 {
			return true
		}
		vec := From(ivec)
		slice := vec.Slice(1, vec.Length())
		islice := ivec[1:]
		for i, v := range islice {
			if slice.At(i) != v {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorSliceAssoc(t *testing.T) {
	f := func(ivec []string) bool {
		if len(ivec) < 3 {
			return true
		}
		vec := From(ivec)
		slice := vec.Slice(1, vec.Length())
		slice = slice.Assoc(1, "foobar")
		return slice.At(1) == "foobar"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorSliceAppend(t *testing.T) {
	f := func(ivec []string) bool {
		if len(ivec) < 3 {
			return true
		}
		vec := From(ivec)
		slice := vec.Slice(1, vec.Length())
		newslice := slice.Append("foobar")
		return newslice.At(slice.Length()) == "foobar"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorSliceAppendInMiddle(t *testing.T) {
	f := func(ivec []string) bool {
		if len(ivec) < 3 {
			return true
		}
		vec := From(ivec)
		slice := vec.Slice(1, vec.Length()-1)
		newslice := slice.Append("foobar")
		return newslice.At(slice.Length()) == "foobar"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorSliceSlice(t *testing.T) {
	f := func(ivec []int) bool {
		if len(ivec) < 5 {
			return true
		}
		vec := From(ivec)
		slice := vec.Slice(1, vec.Length())
		islice := ivec[1:]
		islice = islice[2 : len(islice)-1]
		slice = slice.Slice(2, slice.Length()-1)

		for i, v := range islice {
			if slice.At(i) != v {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorFromSlice(t *testing.T) {
	f := func(ivec []int) bool {
		if len(ivec) < 3 {
			return true
		}
		vec := From(ivec)
		slice := vec.Slice(1, vec.Length())
		vec2 := From(slice)
		return !vec.Equal(vec2)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVectorRange(t *testing.T) {
	t.Run("func(int,T)", func(t *testing.T) {
		f := func(ivec []int) bool {
			expected := 0
			for _, v := range ivec {
				expected += v
			}
			vec := From(ivec)
			got := 0
			vec.Range(func(_, i int) {
				got += i
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("func(int,interface{})", func(t *testing.T) {
		f := func(ivec []int) bool {
			expected := 0
			for _, v := range ivec {
				expected += v
			}
			vec := From(ivec)
			got := 0
			vec.Range(func(_ int, i interface{}) {
				got += i.(int)
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("func(int,T) bool", func(t *testing.T) {
		f := func(ivec []int) bool {
			expected := 0
			for i, v := range ivec {
				if i > 1 {
					break
				}
				expected += v
			}
			vec := From(ivec)
			got := 0
			vec.Range(func(i, v int) bool {
				if i > 1 {
					return false
				}
				got += v
				return true
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("func(int,interface{}) bool", func(t *testing.T) {
		f := func(ivec []int) bool {
			expected := 0
			for i, v := range ivec {
				if i > 1 {
					break
				}
				expected += v
			}
			vec := From(ivec)
			got := 0
			vec.Range(func(i int, v interface{}) bool {
				if i > 1 {
					return false
				}
				got += v.(int)
				return true
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
}

func TestTVectorRange(t *testing.T) {
	t.Run("func(int,T)", func(t *testing.T) {
		f := func(ivec []int) bool {
			expected := 0
			for _, v := range ivec {
				expected += v
			}
			vec := From(ivec).AsTransient()
			got := 0
			vec.Range(func(_, i int) {
				got += i
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("func(int,interface{})", func(t *testing.T) {
		f := func(ivec []int) bool {
			expected := 0
			for _, v := range ivec {
				expected += v
			}
			vec := From(ivec).AsTransient()
			got := 0
			vec.Range(func(_ int, i interface{}) {
				got += i.(int)
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("func(int,T) bool", func(t *testing.T) {
		f := func(ivec []int) bool {
			expected := 0
			for i, v := range ivec {
				if i > 1 {
					break
				}
				expected += v
			}
			vec := From(ivec)
			got := 0
			vec.Range(func(i, v int) bool {
				if i > 1 {
					return false
				}
				got += v
				return true
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("func(int,interface{}) bool", func(t *testing.T) {
		f := func(ivec []int) bool {
			expected := 0
			for i, v := range ivec {
				if i > 1 {
					break
				}
				expected += v
			}
			vec := From(ivec)
			got := 0
			vec.Range(func(i int, v interface{}) bool {
				if i > 1 {
					return false
				}
				got += v.(int)
				return true
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
}

func TestVectorSliceRange(t *testing.T) {
	t.Run("func(int,T)", func(t *testing.T) {
		f := func(ivec []int) bool {
			if len(ivec) < 2 {
				return true
			}
			expected := 0
			for _, v := range ivec[1:] {
				expected += v
			}
			vec := From(ivec)
			slice := vec.Slice(1, vec.Length())
			got := 0
			slice.Range(func(_, i int) {
				got += i
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("func(int,interface{})", func(t *testing.T) {
		f := func(ivec []int) bool {
			if len(ivec) < 2 {
				return true
			}
			expected := 0
			for _, v := range ivec[1:] {
				expected += v
			}
			vec := From(ivec)
			slice := vec.Slice(1, vec.Length())
			got := 0
			slice.Range(func(_ int, i interface{}) {
				got += i.(int)
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("func(int,T) bool", func(t *testing.T) {
		f := func(ivec []int) bool {
			if len(ivec) < 2 {
				return true
			}
			expected := 0
			for i, v := range ivec[1:] {
				if i > 1 {
					break
				}
				expected += v
			}
			vec := From(ivec)
			slice := vec.Slice(1, vec.Length())
			got := 0
			slice.Range(func(i, v int) bool {
				if i > 1 {
					return false
				}
				got += v
				return true
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("func(int,interface{}) bool", func(t *testing.T) {
		f := func(ivec []int) bool {
			if len(ivec) < 2 {
				return true
			}
			expected := 0
			for i, v := range ivec[1:] {
				if i > 1 {
					break
				}
				expected += v
			}
			vec := From(ivec)
			slice := vec.Slice(1, vec.Length())
			got := 0
			slice.Range(func(i int, v interface{}) bool {
				if i > 1 {
					return false
				}
				got += v.(int)
				return true
			})
			return expected == got
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
}

func TestFind(t *testing.T) {
	t.Run("Vector, in bounds", func(t *testing.T) {
		v := New(1, 2, 3, 4, 5)
		val, ok := v.Find(3)
		if !ok {
			t.Fatal("Expected to find a value at index 3")
		}
		if val != 4 {
			t.Fatal("Did not find the expected value at index 3")
		}
	})
	t.Run("Vector, out of bounds", func(t *testing.T) {
		v := New(1, 2, 3, 4, 5)
		_, ok := v.Find(-1)
		if ok {
			t.Fatal("Didn't expected to find a value at index -1")
		}
		_, ok = v.Find(v.Length())
		if ok {
			t.Fatal("Didn't expected to find a value at", v.Length())
		}
	})

	t.Run("TVector, in bounds", func(t *testing.T) {
		v := New(1, 2, 3, 4, 5).AsTransient()
		val, ok := v.Find(3)
		if !ok {
			t.Fatal("Expected to find a value at index 3")
		}
		if val != 4 {
			t.Fatal("Did not find the expected value at index 3")
		}
	})
	t.Run("TVector, out of bounds", func(t *testing.T) {
		v := New(1, 2, 3, 4, 5).AsTransient()
		_, ok := v.Find(-1)
		if ok {
			t.Fatal("Didn't expected to find a value at index -1")
		}
		_, ok = v.Find(v.Length())
		if ok {
			t.Fatal("Didn't expected to find a value at", v.Length())
		}
	})

	t.Run("Slice, in bounds", func(t *testing.T) {
		v := New(1, 2, 3, 4, 5).Slice(1, 5)
		val, ok := v.Find(3)
		if !ok {
			t.Fatal("Expected to find a value at index 3")
		}
		if val != 5 {
			t.Fatal("Did not find the expected value at index 3")
		}
	})
	t.Run("Slice, out of bounds", func(t *testing.T) {
		v := New(1, 2, 3, 4, 5).Slice(1, 5)
		_, ok := v.Find(-1)
		if ok {
			t.Fatal("Didn't expected to find a value at index -1")
		}
		_, ok = v.Find(v.Length())
		if ok {
			t.Fatal("Didn't expected to find a value at", v.Length())
		}
	})

}

func ExampleString() {
	fmt.Println(New(1, 2, 3, 4, 5))
	// Output: [1 2 3 4 5]
}

func ExampleSeqString() {
	fmt.Println(New(1, 2, 3, 4, 5).Seq())
	// Output: (1 2 3 4 5)
}
