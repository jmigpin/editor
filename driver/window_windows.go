//go:build windows && !xproto

package driver

import (
	"github.com/jmigpin/editor/driver/windriver"
)

func NewWindow(opt *WindowOptions) (Window, error) {
	return windriver.NewWindow(opt)
}
