package atomic

import "sync/atomic"

type Bool struct {
	val int32
}

func boolToInt32(val bool) int32 {
	if val {
		return 1
	}
	return 0
}

func MakeBool(val bool) Bool {
	return Bool{
		val: boolToInt32(val),
	}
}

func NewBool(val bool) *Bool {
	return &Bool{
		val: boolToInt32(val),
	}
}

func (b *Bool) Swap(val bool) bool {
	return atomic.SwapInt32(&b.val, boolToInt32(val)) != 0
}

func (b *Bool) Reset(val bool) {
	atomic.StoreInt32(&b.val, boolToInt32(val))
}

func (b *Bool) Deref() bool {
	return atomic.LoadInt32(&b.val) != 0
}
