package goutil

import (
	"fmt"
	"strings"
)

// Allows a src string to have multiple cursor strings to simulate cursor position. Used in testing. Useful cursor runes: "●". First n position is zero.
func SourceCursor(cursorStr, src string, n int) (string, int, error) {
	// cursor positions
	pos := []int{}
	k := 0
	for {
		j := strings.Index(src[k:], cursorStr)
		if j < 0 {
			break
		}
		k += j
		pos = append(pos, k)
		k++
	}

	// nth position
	if n >= len(pos) {
		return "", 0, fmt.Errorf("nth index not found: n=%v, len=%v", n, len(pos))
	}
	index := pos[n]

	// remove cursors
	index -= n * len(cursorStr)
	src2 := strings.Replace(src, cursorStr, "", -1)

	return src2, index, nil
}
