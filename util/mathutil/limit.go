package mathutil

func LimitFloat64(v float64, min, max float64) float64 {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}
func LimitInt(v int, min, max int) int {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}

func Smallest(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func Biggest(a, b int) int {
	if a > b {
		return a
	}
	return b
}
