package btparser

import "fmt"

type fatalError struct {
	err error
}

func FatalError(err error) error {
	if IsFatalError(err) {
		return err
	}
	return &fatalError{err: err}
}
func FatalError2(tag string, err error) error {
	return &fatalError{fmt.Errorf("%v: %v", tag, err)}
}
func (e *fatalError) Error() string {
	return e.err.Error()
}
func (e *fatalError) Unwrap() error {
	return e.err
}

//----------

// IsFatalError is in the parser hot path, so keep it as a shallow check and preserve fatal wrapping at API boundaries.
func IsFatalError(err error) bool {
	_, ok := err.(*fatalError)
	return ok
}
