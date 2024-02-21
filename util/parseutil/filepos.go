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

//----------

// Parse fmt: <filename:line?:col?>. Accepts escapes but doesn't unescape.
//func ParseFilePos(str string) (*FilePos, error) {
//	rd := iorw.NewStringReaderAt(str)
//	res, err := ParseResource(rd, 0)
//	if err != nil {
//		return nil, err
//	}
//	return NewFilePosFromResource(res), nil
//}

//func NewFilePosFromResource(res *Resource) *FilePos {
//	return &FilePos{
//		Offset:   -1,
//		Filename: res.RawPath, // original string (unescaped)
//		Line:     res.Line,
//		Column:   res.Column,
//	}
//}
