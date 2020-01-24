// +build !windows windows,xproto

package driver

import "github.com/jmigpin/editor/driver/xdriver"

func NewWindow2() (Window2, error) {
	return xdriver.NewWindow2()
}
