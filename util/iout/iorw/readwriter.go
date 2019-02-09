package iorw

type Reader interface {
	ReadRuneAt(i int) (ru rune, size int, err error)
	ReadLastRuneAt(i int) (ru rune, size int, err error)

	// TODO: rename to include the word "copy"
	ReadNAt(i, n int) ([]byte, error)      // []byte is a copy
	ReadNSliceAt(i, n int) ([]byte, error) // []byte might not be a copy
	//ReadAtMost(i, n int) ([]byte, error)

	Len() int
}

type Writer interface {
	Insert(i int, p []byte) error
	Delete(i, len int) error
}

type ReadWriter interface {
	Reader
	Writer
}
