package textutil

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
