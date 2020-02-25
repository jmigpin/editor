package iorw

import "sync"

type RWLocked struct {
	sync.RWMutex // can be locked from the outside to use ReadWriter directly
	ReadWriter
}

func NewRWLocker(rw ReadWriter) *RWLocked {
	return &RWLocked{ReadWriter: rw}
}

//----------

func (rw *RWLocked) Insert(i int, p []byte) error {
	rw.Lock()
	defer rw.Unlock()
	return rw.ReadWriter.Insert(i, p)
}

func (rw *RWLocked) Delete(i, n int) error {
	rw.Lock()
	defer rw.Unlock()
	return rw.ReadWriter.Delete(i, n)
}

func (rw *RWLocked) Overwrite(i, n int, p []byte) error {
	rw.Lock()
	defer rw.Unlock()
	return rw.ReadWriter.Overwrite(i, n, p)
}

//----------

func (rw *RWLocked) ReadRuneAt(i int) (ru rune, size int, err error) {
	rw.RLock()
	defer rw.RUnlock()
	return rw.ReadWriter.ReadRuneAt(i)
}

func (rw *RWLocked) ReadLastRuneAt(i int) (ru rune, size int, err error) {
	rw.RLock()
	defer rw.RUnlock()
	return rw.ReadWriter.ReadLastRuneAt(i)
}

func (rw *RWLocked) ReadNSliceAt(i, n int) ([]byte, error) {
	rw.RLock()
	defer rw.RUnlock()
	return rw.ReadWriter.ReadNSliceAt(i, n)
}

func (rw *RWLocked) ReadNCopyAt(i, n int) ([]byte, error) {
	rw.RLock()
	defer rw.RUnlock()
	return rw.ReadWriter.ReadNCopyAt(i, n)
}

func (rw *RWLocked) Min() int {
	rw.RLock()
	defer rw.RUnlock()
	return rw.ReadWriter.Min()
}

func (rw *RWLocked) Max() int {
	rw.RLock()
	defer rw.RUnlock()
	return rw.ReadWriter.Max()
}
