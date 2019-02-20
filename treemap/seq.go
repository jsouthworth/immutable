package treemap

import (
	"jsouthworth.net/go/seq"
)

type sequence struct {
	list seq.Sequence
}

func sequenceNew(t tree) *sequence {
	return &sequence{
		list: sequencePush(t, nil),
	}
}

func sequencePush(t tree, s seq.Sequence) seq.Sequence {
	list := s
	for _, isLeaf := t.(*leaf); !isLeaf; _, isLeaf = t.(*leaf) {
		list = seq.Cons(t, list)
		t = left(t)
	}
	return list
}

func (s *sequence) First() interface{} {
	return value(s.list.First().(tree))
}

func (s *sequence) Next() seq.Sequence {
	t := s.list.First().(tree)
	next := sequencePush(right(t), s.list.Next())
	if next == nil {
		return nil
	}
	return &sequence{list: next}
}

func (s *sequence) String() string {
	return seq.ConvertToString(s)
}
