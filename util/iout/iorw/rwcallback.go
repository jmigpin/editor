package iorw

// Runs callbacks on operations.
type RWCallback struct {
	ReadWriter
	OnWrite func(*RWCallbackWriteOp)
}

//----------

func (rw *RWCallback) Overwrite(i, n int, p []byte) error {
	if err := rw.ReadWriter.Overwrite(i, n, p); err != nil {
		return err
	}
	u := &RWCallbackWriteOp{i, n, len(p)}
	if rw.OnWrite != nil {
		rw.OnWrite(u)
	}
	return nil
}

//----------

type RWCallbackWriteOp struct {
	Index int
	Dn    int // delete n
	In    int // inserted n
}
