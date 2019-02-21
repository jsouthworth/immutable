// Package stack implements a persistent stack.
package stack // import "jsouthworth.net/go/immutable/stack"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/immutable/vector"
	"jsouthworth.net/go/seq"
)

var errRangeSig = errors.New("Range requires a function: func(v vT) bool or func(v vT)")

// Stack is a persistent stack.
type Stack struct {
	backingVector *vector.Vector
}

var empty = Stack{
	backingVector: vector.Empty(),
}

// Empty returns the empty stack.
func Empty() *Stack {
	return &empty
}

// New converts as list of elements to a persistent stack.
func New(elems ...interface{}) *Stack {
	out := Empty().AsTransient()
	for _, elem := range elems {
		out = out.Push(elem)
	}
	return out.AsPersistent()
}

// From will convert many go types to an immutable map.
// Converting some types is more efficient than others and the
// mechanisms are described below.
//
// *Stack:
//    Used directly as it is already immutable.
// *TStack:
//    AsPersistent is called on the value and the result used for the stack.
// *Vector:
//    Used directly as the stack as it is already immutable.
// *TVector:
//    AsPersistent is called on the value and the result used for the stack.
// []interface{}:
//    New is called with the elements.
// seq.Sequable:
//    Seq is called on the value and the stack is built from the resulting sequence.
// seq.Sequence:
//    The stack is built from the sequence. Care should be taken to provide finite sequences or the vector will grow without bound.
// []T:
//    The slice is converted to a vector using reflection.
func From(value interface{}) *Stack {
	switch v := value.(type) {
	case *Stack:
		return v
	case *TStack:
		return v.AsPersistent()
	default:
		vec := vector.From(value)
		if vec.Length() == 0 {
			return Empty()
		}
		return &Stack{
			backingVector: vec,
		}
	}
}

// Push returns a new stack with the element as the top of the stack.
func (s *Stack) Push(elem interface{}) *Stack {
	return &Stack{
		backingVector: s.backingVector.Append(elem),
	}
}

// Pop returns a new stack without the top element
func (s *Stack) Pop() *Stack {
	v := s.backingVector.Pop()
	if v.Length() == 0 {
		return Empty()
	}
	return &Stack{
		backingVector: v,
	}
}

// Top returns the top of the stack
func (s *Stack) Top() interface{} {
	return s.backingVector.At(s.backingVector.Length() - 1)
}

// Find whether the value exists in the stack by walking every value.
// Returns the value and whether or not it was found.
func (s *Stack) Find(value interface{}) (interface{}, bool) {
	var out interface{}
	var found bool
	s.Range(func(v interface{}) bool {
		if v == value {
			out = v
			found = true
			return false
		}
		return true
	})
	return out, found
}

// AsTransient will return a mutable copy on write version of the stack.
func (s *Stack) AsTransient() *TStack {
	return &TStack{
		backingVector: s.backingVector.AsTransient(),
	}
}

// Range calls the passed in function on each element of the stack.
// The function passed in may be of many types:
//
// func(value interface{}) bool:
//    Takes a value of any type and returns if the loop should continue.
//    Useful to avoid reflection where not needed and to support
//    heterogenous stacks.
// func(value interface{})
//    Takes a value of any type.
//    Useful to avoid reflection where not needed and to support
//    heterogenous stacks.
// func(value T) bool:
//    Takes a value of the type of element stored in the stack and
//    returns if the loop should continue. Useful for homogeneous stacks.
//    Is called with reflection and will panic if the type is incorrect.
// func(value T)
//    Takes a value of the type of element stored in the stack and
//    returns if the loop should continue. Useful for homogeneous stacks.
//    Is called with reflection and will panic if the type is incorrect.
// Range will panic if passed anything that doesn't match one of these signatures
func (s *Stack) Range(do interface{}) {
	cont := true
	for stack := s; stack != Empty() && cont; stack = stack.Pop() {
		value := stack.Top()
		switch fn := do.(type) {
		case func(value interface{}) bool:
			cont = fn(value)
		case func(value interface{}):
			fn(value)
		default:
			rv := reflect.ValueOf(do)
			if rv.Kind() != reflect.Func {
				panic(errRangeSig)
			}
			rt := rv.Type()
			if rt.NumIn() != 1 || rt.NumOut() > 1 {
				panic(errRangeSig)
			}
			if rt.NumOut() == 1 &&
				rt.Out(0).Kind() != reflect.Bool {
				panic(errRangeSig)
			}
			out := dyn.Apply(do, value)
			if out != nil {
				cont = out.(bool)
			}
		}
	}
}

// Seq returns a representation of the stack as a sequence
// corresponding to the elements of the stack.
func (s *Stack) Seq() seq.Sequence {
	return &stackSequence{
		stack: s,
	}
}

// String returns a representation of the stack as a string.
func (s *Stack) String() string {
	b := new(strings.Builder)
	fmt.Fprint(b, "[ ")
	s.Range(func(item interface{}) {
		fmt.Fprintf(b, "%v ", item)
	})
	fmt.Fprint(b, "]")
	return b.String()
}

type stackSequence struct {
	stack *Stack
}

func (s *stackSequence) First() interface{} {
	return s.stack.Top()
}

func (s *stackSequence) Next() seq.Sequence {
	new := s.stack.Pop()
	if new.backingVector.Length() == 0 {
		return nil
	}
	return &stackSequence{
		stack: new,
	}
}

func (s *stackSequence) String() string {
	return seq.ConvertToString(s)
}

type TStack struct {
	backingVector *vector.TVector
}

// Push places an element at the top of the stack. s is returned
func (s *TStack) Push(elem interface{}) *TStack {
	s.backingVector = s.backingVector.Append(elem)
	return s
}

// Pop removes the top element of the stack. s is returned.
func (s *TStack) Pop() *TStack {
	s.backingVector = s.backingVector.Pop()
	return s
}

// Top returns the top element of the stack.
func (s *TStack) Top() interface{} {
	return s.backingVector.At(s.backingVector.Length() - 1)
}

// Find whether the value exists in the stack by walking every value.
// Returns the value and whether or not it was found.
func (s *TStack) Find(value interface{}) (interface{}, bool) {
	var out interface{}
	var found bool
	s.Range(func(v interface{}) bool {
		if v == value {
			out = v
			found = true
			return false
		}
		return true
	})
	return out, found
}

// AsPersistent returns the an immutable version of the stack. Any
// transient operations performed after this will cause a panic.
func (s *TStack) AsPersistent() *Stack {
	v := s.backingVector.AsPersistent()
	if v.Length() == 0 {
		return Empty()
	}
	return &Stack{
		backingVector: v,
	}
}

// Range calls the passed in function on each element of the stack.
// The function passed in may be of many types:
//
// func(value interface{}) bool:
//    Takes a value of any type and returns if the loop should continue.
//    Useful to avoid reflection where not needed and to support
//    heterogenous stacks.
// func(value interface{})
//    Takes a value of any type.
//    Useful to avoid reflection where not needed and to support
//    heterogenous stacks.
// func(value T) bool:
//    Takes a value of the type of element stored in the stack and
//    returns if the loop should continue. Useful for homogeneous stacks.
//    Is called with reflection and will panic if the type is incorrect.
// func(value T)
//    Takes a value of the type of element stored in the stack and
//    returns if the loop should continue. Useful for homogeneous stacks.
//    Is called with reflection and will panic if the type is incorrect.
// Range will panic if passed anything that doesn't match one of these signatures
func (s *TStack) Range(do interface{}) {
	cont := true
	for i := s.backingVector.Length() - 1; i >= 0 && cont; i-- {
		value := s.backingVector.At(i)
		switch fn := do.(type) {
		case func(value interface{}) bool:
			cont = fn(value)
		case func(value interface{}):
			fn(value)
		default:
			rv := reflect.ValueOf(do)
			if rv.Kind() != reflect.Func {
				panic(errRangeSig)
			}
			rt := rv.Type()
			if rt.NumIn() != 1 || rt.NumOut() > 1 {
				panic(errRangeSig)
			}
			if rt.NumOut() == 1 &&
				rt.Out(0).Kind() != reflect.Bool {
				panic(errRangeSig)
			}
			outs := rv.Call([]reflect.Value{
				reflect.ValueOf(value)})
			if len(outs) != 0 {
				cont = outs[0].Interface().(bool)
			}
		}
	}
}

// String returns a representation of the stack as a string.
func (s *TStack) String() string {
	b := new(strings.Builder)
	fmt.Fprint(b, "[ ")
	s.Range(func(item interface{}) {
		fmt.Fprintf(b, "%v ", item)
	})
	fmt.Fprint(b, "]")
	return b.String()
}
