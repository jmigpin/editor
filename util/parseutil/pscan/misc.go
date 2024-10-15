package pscan

func SurroundingString(b []byte, k int, pad int) string {
	// pad n in each direction for error string
	i := max(k-pad, 0)
	i2 := min(k+pad, len(b))

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

//----------

// probably belongs to a textutil pkg, but here to reduce dependencies
func FindLineColumn(data []byte, pos int) (int, int, bool) {
	if pos > len(data) {
		return 0, 0, false
	}
	line, col := 1, 1
	for i := 0; i < pos; i++ {
		b := data[i]
		if b == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col, true
}
