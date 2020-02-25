package iout

type FnWriter func([]byte) (int, error)

func (w FnWriter) Write(p []byte) (int, error) {
	return w(p)
}

//----------

type FnCloser func() error

func (c FnCloser) Close() error {
	return c()
}

//----------
