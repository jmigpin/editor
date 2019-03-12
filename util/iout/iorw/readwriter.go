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

	Len() int
}

type Writer interface {
	Insert(i int, p []byte) error
	Delete(i, len int) error
}
