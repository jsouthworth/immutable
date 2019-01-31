package treemap

type Equaler interface {
	Equal(other interface{}) bool
}

type Comparer interface {
	Compare(other interface{}) int
}

func equal(one, two interface{}) bool {
	switch v := one.(type) {
	case Equaler:
		return v.Equal(two)
	default:
		return one == two
	}
}

func defaultCompare(k1, k2 interface{}) int {
	if k1 == k2 {
		return 0
	}
	if k1 == nil {
		return -1
	}
	if k2 == nil {
		return 1
	}
	switch v1 := k1.(type) {
	case uint:
		v2 := k2.(uint)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case uint8:
		v2 := k2.(uint8)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case uint16:
		v2 := k2.(uint16)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case uint32:
		v2 := k2.(uint32)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case uint64:
		v2 := k2.(uint64)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case int:
		v2 := k2.(int)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case int8:
		v2 := k2.(int8)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case int16:
		v2 := k2.(int16)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case int32:
		v2 := k2.(int32)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case int64:
		v2 := k2.(int64)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case float32:
		v2 := k2.(float32)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case float64:
		v2 := k2.(float64)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	case string:
		v2 := k2.(string)
		switch {
		case v1 < v2:
			return -1
		case v1 > v2:
			return 1
		default:
			return 0
		}
	default:
		return k1.(Comparer).Compare(k2)
	}
}
