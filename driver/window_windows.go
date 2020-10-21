// +build windows,!xproto

package driver

import (
	"github.com/jmigpin/editor/v2/driver/windriver"
)

func NewWindow() (Window, error) {
	return windriver.NewWindow()
}
