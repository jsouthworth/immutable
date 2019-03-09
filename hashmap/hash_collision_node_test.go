package hashmap

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"jsouthworth.net/go/seq"
)

type hashCollider string

func (h hashCollider) Hash() uintptr {
	return 10
}

func TestHashCollisionNode(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	properties := gopter.NewProperties(parameters)
	//assoc
	properties.Property("random.Assoc(x,x).Assoc(y,y); random.At(x) == x",
		prop.ForAll(
			func(rm *rmap, k1, k2 string) bool {
				if k1 == k2 {
					return true
				}
				m := rm.m.Assoc(hashCollider(k1), k1).
					Assoc(hashCollider(k2), k2)
				return m.At(hashCollider(k1)) == k1
			},
			genRandomMap,
			gen.Identifier(),
			gen.Identifier(),
		))
	properties.Property("random.Assoc(x,x).Assoc(y,y).Assoc(z,z); random.At(x) == x",
		prop.ForAll(
			func(rm *rmap, k1, k2, k3 string) bool {
				if k1 == k2 || k2 == k3 || k1 == k3 {
					return true
				}
				m := rm.m.Assoc(hashCollider(k1), k1).
					Assoc(hashCollider(k2), k2).
					Assoc(hashCollider(k3), k3)
				return m.At(hashCollider(k1)) == k1
			},
			genRandomMap,
			gen.Identifier(),
			gen.Identifier(),
			gen.Identifier(),
		))
	properties.Property("collided.Assoc(x,z); random.At(x) == z",
		prop.ForAll(
			func(rm *rmap, k1, k2, k3 string) bool {
				if k1 == k2 || k2 == k3 || k1 == k3 {
					return true
				}
				m := rm.m.Assoc(hashCollider(k1), k1).
					Assoc(hashCollider(k2), k2).
					Assoc(hashCollider(k3), k3).
					Assoc(hashCollider(k1), k3)
				return m.At(hashCollider(k1)) == k3
			},
			genRandomMap,
			gen.Identifier(),
			gen.Identifier(),
			gen.Identifier(),
		))
	//find
	properties.Property("collided contains all",
		prop.ForAll(
			func(rm *rmap, k1, k2, k3 string) bool {
				if k1 == k2 || k2 == k3 || k1 == k3 {
					return true
				}
				m := rm.m.Assoc(hashCollider(k1), k1).
					Assoc(hashCollider(k2), k2).
					Assoc(hashCollider(k3), k3)
				return m.Contains(hashCollider(k1)) &&
					m.Contains(hashCollider(k2)) &&
					m.Contains(hashCollider(k3))
			},
			genRandomMap,
			gen.Identifier(),
			gen.Identifier(),
			gen.Identifier(),
		))
	//without
	properties.Property("collided.Delete(x); !random.Contains(x)",
		prop.ForAll(
			func(rm *rmap, k1, k2, k3 string) bool {
				if k1 == k2 || k2 == k3 || k1 == k3 {
					return true
				}
				m := rm.m.Assoc(hashCollider(k1), k1).
					Assoc(hashCollider(k2), k2).
					Assoc(hashCollider(k3), k3).
					Assoc(hashCollider(k1), k3).
					Delete(hashCollider(k1))
				return !m.Contains(hashCollider(k1))
			},
			genRandomMap,
			gen.Identifier(),
			gen.Identifier(),
			gen.Identifier(),
		))
	properties.Property("collided Remove All",
		prop.ForAll(
			func(rm *rmap, k1, k2, k3 string) bool {
				if k1 == k2 || k2 == k3 || k1 == k3 {
					return true
				}
				m := rm.m.Assoc(hashCollider(k1), k1).
					Assoc(hashCollider(k2), k2).
					Assoc(hashCollider(k3), k3).
					Assoc(hashCollider(k1), k3).
					Delete(hashCollider(k1)).
					Delete(hashCollider(k2)).
					Delete(hashCollider(k3))
				return !m.Contains(hashCollider(k1))
			},
			genRandomMap,
			gen.Identifier(),
			gen.Identifier(),
			gen.Identifier(),
		))
	//seq
	properties.Property("Seq process full map", prop.ForAll(
		func(rm *rmap, k1, k2, k3 string) (ok bool) {
			hc1 := hashCollider(k1)
			hc2 := hashCollider(k2)
			hc3 := hashCollider(k3)
			m := rm.m.Assoc(hc1, k1).
				Assoc(hc2, k2).
				Assoc(hc3, k3)
			s := seq.Seq(m)
			if s == nil {
				return true
			}
			foundAll := true
			fn := func(entry Entry) bool {
				key := entry.Key()
				switch key {
				case hc1, hc2, hc3:
					return true
				default:
					foundAll = rm.entries[key.(string)] ==
						entry.Value()
				}
				return true
			}
			var cont = true
			for s != nil && cont {
				entry := seq.First(s).(Entry)
				cont = fn(entry)
				s = seq.Seq(seq.Next(s))
			}
			return foundAll
		},
		genRandomMap,
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))
	//range
	properties.Property("Range access the full map", prop.ForAll(
		func(rm *rmap, k1, k2, k3 string) bool {
			foundAll := true
			hc1 := hashCollider(k1)
			hc2 := hashCollider(k2)
			hc3 := hashCollider(k3)
			m := rm.m.Assoc(hc1, k1).
				Assoc(hc2, k2).
				Assoc(hc3, k3)
			m.Range(func(key, val interface{}) bool {
				if !foundAll {
					return false
				}
				switch key {
				case hc1, hc2, hc3:
					return true
				default:
					foundAll = rm.entries[key.(string)] == val
				}
				return true
			})

			return foundAll
		},
		genRandomMap,
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))
	properties.TestingRun(t)
}
