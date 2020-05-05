// +build !windows windows,xproto

package driver

import "github.com/jmigpin/editor/v2/driver/xdriver"

func NewWindow() (Window, error) {
	return xdriver.NewWindow()
}
