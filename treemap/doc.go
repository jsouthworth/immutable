// Package treemap implements a map on top of a persistent B-tree.
//
// A note about Key and Value equality. If you would like to override
// the default go equality operator for keys and values in this map library
// implement the Equal(other interface{}) bool function for the type.
// Otherwise '==' will be used with all its restrictions. Additionally,
// Key's must be comparable. One may implement Compare(other interface{}) int
// to override the default comparable restrcitions.
package treemap
