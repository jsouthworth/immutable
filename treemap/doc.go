// Package treemap implements a persistent Red/Black tree. This tree is
// based on Okasaki's persistent Red/Black tree with Germane and Might's
// deletion extension. See:
// http://www.eecs.usma.edu/webs/people/okasaki/jfp99.ps and
// http://matt.might.net/papers/germane2014deletion.pdf for details.
//
// A note about Key and Value equality. If you would like to override
// the default go equality operator for keys and values in this map library
// implement the Equal(other interface{}) bool function for the type.
// Otherwise '==' will be used with all its restrictions. Additionally,
// Key's must be comparable. One may implement Compare(other interface{}) int
// to override the default comparable restrcitions.
package treemap
