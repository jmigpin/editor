package mathutil

import (
	"math"

	"golang.org/x/exp/constraints"
)

func RoundFloat64(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

//----------

// TODO: remove
func LimitFloat64(v float64, min, max float64) float64 {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}

// TODO: remove
func LimitInt(v int, min, max int) int {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}

func Limit[T constraints.Ordered](v, min, max T) T {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}

//----------

func Min[T constraints.Ordered](s ...T) T {
	m := s[0]
	for _, v := range s[1:] {
		if m > v {
			m = v
		}
	}
	return m
}
func Max[T constraints.Ordered](s ...T) T {
	m := s[0]
	for _, v := range s[1:] {
		if m < v {
			m = v
		}
	}
	return m
}
