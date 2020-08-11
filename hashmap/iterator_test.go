package hashmap

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

func TestIterator(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	properties.Property("Iterator access the full map", prop.ForAll(
		func(rm *rmap) bool {
			foundAll := true
			iter := rm.m.Iterator()
			for iter.HasNext() {
				key, val := iter.Next()
				if !foundAll {
					break
				}
				foundAll = rm.entries[key.(string)] == val
			}
			return foundAll
		},
		genRandomMap,
	))
	properties.TestingRun(t)
}

func BenchmarkIterator(b *testing.B) {
	m := Empty().Transform(func(m *TMap) *TMap {
		for i := 0; i < b.N; i++ {
			m.Assoc(i, i)
		}
		return m
	})
	b.ResetTimer()
	var sum int
	i := m.Iterator()
	for i.HasNext() {
		_, v := i.Next()
		sum += v.(int)
	}
}
