package iorw

type ReadWriter interface {
	Reader
	Writer
}

type Reader interface {
	ReadRuneAt(i int) (ru rune, size int, err error)
	ReadLastRuneAt(i int) (ru rune, size int, err error)

	// there must be at least N bytes available or there will be an error
	ReadNCopyAt(i, n int) ([]byte, error)
	ReadNSliceAt(i, n int) ([]byte, error) // []byte might not be a copy

	// length min/max
	Min() int
	Max() int
}

// Returns a slice (not a copy).
func ReadFullSlice(rd Reader) ([]byte, error) {
	min, max := rd.Min(), rd.Max()
	return rd.ReadNSliceAt(min, max-min)
}

//----------

type Writer interface {
	Insert(i int, p []byte) error
	Delete(i, length int) error
	Overwrite(i, length int, p []byte) error
}

//----------

// used to determine the write operation (undoredo and others)
type WriterOp int

const (
	InsertWOp WriterOp = iota
	DeleteWOp
	OverwriteWOp
)
