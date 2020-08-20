package btree

type keyStitcher struct {
	target []interface{}
	offset int
}

func (s *keyStitcher) copyAll(source []interface{}, from, to int) {
	if to >= from {
		copy(s.target[s.offset:s.offset+(to-from)], source[from:to])
		s.offset += to - from
	}
}

func (s *keyStitcher) copyOne(val interface{}) {
	s.target[s.offset] = val
	s.offset++
}

type nodeStitcher struct {
	target []node
	offset int
}

func (s *nodeStitcher) copyAll(source []node, from, to int) {
	if to >= from {
		copy(s.target[s.offset:s.offset+(to-from)], source[from:to])
		s.offset += to - from
	}
}

func (s *nodeStitcher) copyOne(val node) {
	s.target[s.offset] = val
	s.offset++
}
