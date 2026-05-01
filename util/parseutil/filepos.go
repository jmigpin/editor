package parseutil

type FilePos struct {
	Filename     string
	Offset, Len  int // length after offset for a range
	Line, Column int // bigger than zero to be considered
}

func (fp *FilePos) HasPos() bool {
	return fp.Line != 0 || fp.Offset >= 0
}
func (fp *FilePos) HasNoLinecol() bool {
	return fp.Line == 0
}
