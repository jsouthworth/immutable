package treemap

import (
	"jsouthworth.net/go/immutable/internal/btree"
	"jsouthworth.net/go/seq"
)

type sequence struct {
	iter btree.Iterator
}

func sequenceNew(iter btree.Iterator) *sequence {
	return &sequence{
		iter: iter,
	}
}

func (s *sequence) First() interface{} {
	return s.iter.Next()
}

func (s *sequence) Next() seq.Sequence {
	new := &(*s)
	hasNext := new.iter.HasNext()
	if !hasNext {
		return nil
	}
	return new
}

func (s *sequence) String() string {
	return seq.ConvertToString(s)
}
