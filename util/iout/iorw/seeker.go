package iorw

//type Seeker interface {
//	Seek(i int, from SeekFrom) (int, error)
//}

//type SeekFrom int

//const (
//	SeekStart SeekFrom = iota
//	SeekCurrent
//	SeekEnd
//)

//type Lengther interface {
//	Len() int
//}

////----------

//type ReadWriter2 interface {
//	Reader2
//	Writer2
//}

//type Reader2 interface {
//	ReadFast(n int) ([]byte, error) // might read less then n
//}

//type Writer2 interface {
//	Overwrite(del int, p []byte) error // writes len(p)
//}

////----------

//// TODO
////ReadAt(p []byte, i int) (int, error)
////WriteAt(p []byte, i int) (n int, err error)

////----------

//type ReadSeeker interface {
//	Reader2
//	Seeker
//}

//type WriteSeeker interface {
//	Writer2
//	Seeker
//}

//type RuneReadSeeker interface {
//	ReadRune() (r rune, size int, err error) // io.RuneReader
//	//ReadLastRune() (r rune, size int, err error)
//	Seeker
//}

////----------

//type Seek struct {
//	i int
//	l Lengther
//}

//func NewSeek(l Lengther) *Seek {
//	return &Seek{l: l}
//}

//func (s *Seek) Seek(i int, from SeekFrom) (int, error) {
//	var abs int
//	switch from {
//	case SeekStart:
//		abs = i
//	case SeekCurrent:
//		abs = s.i + i
//	case SeekEnd:
//		abs = s.l.Len() + i
//	}
//	if abs < 0 {
//		return 0, fmt.Errorf("iorws.seek: %v<0", abs)
//	}
//	if abs > s.l.Len() {
//		return 0, fmt.Errorf("iorws.seek: %v>%v", abs, s.l.Len())
//	}
//	s.i = abs
//	return abs, nil
//}

//func (s *Seek) SeekCurrent() int {
//	return s.i
//}

////----------

//type LimitedSeek struct {
//	Seeker
//	start, n int
//}

//func NewLimitedSeek(s Seeker, start, n int) *LimitedSeek {
//	return &LimitedSeek{s, start, n}
//}

//func (s *LimitedSeek) Seek(i int, from SeekFrom) (int, error) {
//	var abs int
//	switch from {
//	case SeekStart:
//		abs = s.start + i
//	case SeekCurrent:
//		k, err := s.Seeker.Seek(0, SeekCurrent)
//		if err != nil {
//			return 0, err
//		}
//		abs = k + i
//	case SeekEnd:
//		abs = s.start + s.n + i
//	}
//	if abs < s.start {
//		return 0, fmt.Errorf("iorws.limitedseek: %v<%v", abs, s.start)
//	}
//	if abs > s.start+s.n {
//		return 0, fmt.Errorf("iorws.limitedseek: %v>%v", abs, s.start+s.n)
//	}
//	return s.Seek(abs, SeekStart)
//}

////----------

//type rs struct {
//	r ReaderAt
//	Seeker
//}

//func NewReadSeeker(r ReaderAt) ReadSeeker {
//	return &rs{r, NewSeek(r)}
//}

//// Implements Reader
//func (rs *rs) ReadFast(n int) ([]byte, error) {
//	i, err := rs.Seek(0, SeekCurrent)
//	if err != nil {
//		return nil, err
//	}
//	b, err := rs.r.ReadFastAt(i, n)
//	if err != nil {
//		return nil, err
//	}
//	_, err = rs.Seek(len(b), SeekCurrent)
//	return b, err
//}

////----------

//type ws struct {
//	w WriterAt
//	Seeker
//}

//func NewWriteSeeker(w WriterAt, l Lengther) WriteSeeker {
//	return &ws{w, NewSeek(l)}
//}

//// Implements Writer
//func (ws *ws) Overwrite(del int, p []byte) error {
//	i, err := ws.Seek(0, SeekCurrent)
//	if err != nil {
//		return err
//	}
//	if err := ws.w.OverwriteAt(i, del, p); err != nil {
//		return err
//	}
//	_, err = ws.Seek(len(p), SeekCurrent)
//	return err
//}

////----------

//type rr struct {
//	r ReaderAt
//	Seeker
//}

//func NewRuneReadSeeker(r ReaderAt) RuneReadSeeker {
//	s := NewSeek(r)
//	return &rr{r, s}
//}

//// Implements io.RuneReader
//func (r *rr) ReadRune() (rune, int, error) {
//	i, err := r.Seek(0, SeekCurrent)
//	if err != nil {
//		return 0, 0, err
//	}
//	defer func() { _, _ = r.Seek(i, SeekStart) }() // restore/advance index

//	rra := NewRuneReaderAt(r.r)
//	ru, size, err := rra.ReadRuneAt(i)
//	i += size // advance
//	return ru, size, err
//}

////----------

//type rrr struct {
//	r ReaderAt
//	Seeker
//}

//func NewReverseRuneReadSeeker(r ReaderAt) RuneReadSeeker {
//	s := NewSeek(r)
//	return &rrr{r, s}
//}

//// Implements io.RuneReader
//func (r *rrr) ReadRune() (rune, int, error) {
//	i, err := r.Seek(0, SeekCurrent)
//	if err != nil {
//		return 0, 0, err
//	}
//	defer func() { _, _ = r.Seek(i, SeekStart) }() // restore/advance index

//	rra := NewRuneReaderAt(r.r)
//	ru, size, err := rra.ReadLastRuneAt(i)
//	i -= size // advance
//	return ru, size, err
//}

////----------

//type ior struct {
//	r Reader2
//}

//func NewIOReader(r Reader2) io.Reader {
//	return &ior{r}
//}

//// Implements io.Reader
//func (r *ior) Read(p []byte) (int, error) {
//	b, err := r.r.ReadFast(len(p))
//	if err != nil {
//		return 0, err
//	}
//	return copy(p, b), nil
//}

////----------

//type iow struct {
//	w Writer2
//}

//func NewIOWriter(w Writer2) io.Writer {
//	return &iow{w}
//}

//// Implements io.Writer
//func (w *iow) Write(p []byte) (int, error) {
//	if err := w.w.Overwrite(0, p); err != nil {
//		return 0, err
//	}
//	return len(p), nil
//}
