package iout

import (
	"fmt"
	"strings"
	"sync"
)

type MultiError struct {
	errors []error
	addMu  sync.Mutex
}

// Returns an error (MultiError) or nil if the errors added were all nil.
func MultiErrors(errs ...error) error {
	me := &MultiError{}
	me.Add(errs...)
	return me.Result()
}

// Returns itself, or nil if it has no errors.
func (me *MultiError) Result() error {
	if len(me.errors) == 0 {
		return nil
	}
	return me
}

// Only this func can be used concurrently.
func (me *MultiError) Add(errs ...error) {
	w := make([]error, 0, len(errs)) // usually small
	for _, e := range errs {
		if e != nil {
			w = append(w, e)
		}
	}
	if len(w) == 0 {
		return
	}
	me.addMu.Lock()
	defer me.addMu.Unlock()
	me.errors = append(me.errors, w...)
}

func (me *MultiError) Error() string {
	if len(me.errors) == 1 {
		return me.errors[0].Error()
	}
	u := []string{}
	for i, e := range me.errors {
		v := indentNewlines("\t", e.Error())
		u = append(u, fmt.Sprintf("err%d: %v", i+1, v))
	}
	v := "\t" + indentNewlines("\t", strings.Join(u, "\n"))
	return fmt.Sprintf("multierror(%d){\n%s\n}", len(me.errors), v)
}

//----------

func indentNewlines(tab string, u string) string {
	u = strings.ReplaceAll(u, "\n", "\n"+tab)
	return u
}
