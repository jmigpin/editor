// +build !windows windows,xproto

package driver

import "github.com/jmigpin/editor/driver/xdriver"

func NewWindow2() (Window2, error) {
	return xdriver.NewWindow2()
}

// Deprecated: use NewWindow2
func NewWindow() (Window, error) {
	xw, err := xdriver.NewWindow2()
	if err != nil {
		return nil, err
	}
	return NewW2Window(xw), nil
}
