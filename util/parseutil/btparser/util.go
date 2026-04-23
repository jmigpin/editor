package btparser

import "slices"

func ReverseString(s string) string {
	rs := []rune(s)
	slices.Reverse(rs)
	return string(rs)
}
