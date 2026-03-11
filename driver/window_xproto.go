//go:build !windows || (windows && xproto)

package driver

import "github.com/jmigpin/editor/driver/xdriver"

func NewWindow(opt *WindowOptions) (Window, error) {
	return xdriver.NewWindow(opt)
}
