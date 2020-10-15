# Immutable

[![GoDoc](https://godoc.org/jsouthworth.net/go/immutable?status.svg)](https://godoc.org/jsouthworth.net/go/immutable)
[![Build Status](https://travis-ci.com/jsouthworth/immutable.svg?branch=master)](https://travis-ci.com/jsouthworth/immutable.svg?branch=master)
[![Coverage Status](https://coveralls.io/repos/github/jsouthworth/immutable/badge.svg?branch=master)](https://coveralls.io/github/jsouthworth/immutable?branch=master)

This library implements several persistent datastructures for the go programming language. A vector based on Radix Balanced Trees with some optimizations adapted from Clojure. A HAMT based hashmap inspired heavily by Clojure's hashmap. A B-Tree based treemap based on the B-Tree implementation used in [persistent-sorted-set](https://github.com/tonsky/persistent-sorted-set).

Several additional overlay data-structures are provided for conveience. A list, queue, stack, hashset, and treeset are built on top of the 3 basic data-structures.

One of the goals of this library is to feel as idomatic in go as it can. Forced boxing of the values is alliviated by using reflection to call functions of the appropriate type where appropriate.

The APIs of the various implementations can be considered stable. Only extensions will be made to them.

## Getting started
```
go get jsouthworth.net/go/immutable
```

## Usage

The full documentation is available at
[jsouthworth.net/go/immutable](https://jsouthworth.net/go/immutable)

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE)

## Acknowledgments

* The Clojure project's implementation of these structures heavily influenced this implementation.
* [persistent-sorted-set](https://github.com/tonsky/persistent-sorted-set) influenced the btree implementation used to back treemap and treeset.

## TODO

* [ ] Performance benchmarking and improvements. Performance is acceptable but can problably be made better.
* [ ] Add JSON marshalling support.
