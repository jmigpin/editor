package iout

type FnWriter func([]byte) (int, error)

func (w FnWriter) Write(p []byte) (int, error) {
	return w(p)
}
