package iorw

type ReadWriterAt interface {
	ReaderAt
	WriterAt
}

type ReaderAt interface {
	ReadFastAt(i, n int) ([]byte, error) // not a copy; might read less then n
	// indexes: min>=0 && min<=max && max<=length
	Min() int
	Max() int

	// note: read runes with
	// iorw.ReadRuneAt(..)
	// iorw.ReadLastRuneAt(..)
}

type WriterAt interface {
	// insert: Overwrite(i, 0, p)
	// delete: Overwrite(i, n, nil)
	OverwriteAt(i, del int, p []byte) error // writes len(p)
}
