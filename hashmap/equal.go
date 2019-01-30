package hashmap

type Equaler interface {
	Equal(v interface{}) bool
}

func equal(v1, v2 interface{}) bool {
	switch val := v1.(type) {
	case Equaler:
		return val.Equal(v2)
	default:
		return v1 == v2
	}
}
