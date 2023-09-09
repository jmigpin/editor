package strconvutil

func BasicUnquote(s string) (string, bool) {
	if len(s) < 2 {
		return "", false
	}

	q := rune(s[0])
	lq := len(string(q))

	ok := false
	quotes := []rune("\"'`") // allowed quotes
	for _, u := range quotes {
		if u == q {
			ok = true
			break
		}
	}
	if !ok {
		return "", false
	}

	// end quote must equal start
	q2 := rune(s[len(s)-lq])
	if q2 != q {
		return "", false
	}

	return s[lq : len(s)-lq], true
}
