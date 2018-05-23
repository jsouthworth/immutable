# Immutable

[![GoDoc](https://godoc.org/jsouthworth.net/go/immutable?status.svg)](https://godoc.org/jsouthworth.net/go/immutable)

This library implements several persistent datastructures for the go programming language. A vector based on Radix Balanced Trees with some optimizations adapted from Clojure. A HAMT based hashmap inspired heavily by Clojure's hashmap. A Red/Black tree based treemap based on Okasaki's persistent red/black tree with a deletion extension from Germane and Might.

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
* Might's blog regarding the Red/Black tree was the source of inspiration for the treemap implementation.

## TODO

* [ ] Performance benchmarking and improvements. Performance is acceptable but can problably be made better.
* [ ] Add JSON marshalling support.
