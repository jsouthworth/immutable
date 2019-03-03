// Package hashmap implements a persistent HAMT based hashmap. This
// package is inspired by on Clojure's persistent and transient hashmap
// implementations. See https://lampwww.epfl.ch/papers/idealhashtrees.pdf
// for more information on the algoritm.
//
// A note about Key and Value equality. If you would like to override
// the default go equality operator for keys and values in this map library
// implement the Equal(other interface{}) bool function for the type.
// Otherwise '==' will be used with all its restrictions.
package hashmap
