package pscan

import "github.com/jmigpin/editor/util/mathutil"

func ContainsRune(rs []rune, ru rune) bool {
	for _, ru2 := range rs {
		if ru2 == ru {
			return true
		}
	}
	return false
}

//----------

func SurroundingString(b []byte, k int, pad int) string {
	// pad n in each direction for error string
	i := mathutil.Max(k-pad, 0)
	i2 := mathutil.Min(k+pad, len(b))

	if i > i2 {
		return ""
	}

	s := string(b[i:i2])
	if s == "" {
		return ""
	}

	// position indicator (valid after test of empty string)
	c := k - i

	sep := "●" // "←"
	s2 := s[:c] + sep + s[c:]
	if i > 0 {
		s2 = "..." + s2
	}
	if i2 < len(b)-1 {
		s2 = s2 + "..."
	}
	return s2
}
