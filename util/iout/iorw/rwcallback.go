package iorw

// Runs callbacks on operations.
type RWCallback struct {
	ReadWriter
	OnWrite func(*RWCallbackWriteOp)
}

//----------

func (rw *RWCallback) writeOpCallback(v *RWCallbackWriteOp) {
	if rw.OnWrite != nil {
		rw.OnWrite(v)
	}
}

//----------

func (rw *RWCallback) Insert(i int, p []byte) error {
	if err := rw.ReadWriter.Insert(i, p); err != nil {
		return err
	}
	u := &RWCallbackWriteOp{WopInsert, i, len(p), 0}
	rw.writeOpCallback(u)
	return nil
}

func (rw *RWCallback) Delete(i, length int) error {
	if err := rw.ReadWriter.Delete(i, length); err != nil {
		return err
	}
	u := &RWCallbackWriteOp{WopDelete, i, length, 0}
	rw.writeOpCallback(u)
	return nil
}

func (rw *RWCallback) Overwrite(i, length int, p []byte) error {
	if err := rw.ReadWriter.Overwrite(i, length, p); err != nil {
		return err
	}
	u := &RWCallbackWriteOp{WopOverwrite, i, length, len(p)}
	rw.writeOpCallback(u)
	return nil
}

//----------

type RWCallbackWriteOp struct {
	Type    WriterOp
	Index   int
	Length1 int
	Length2 int
}
