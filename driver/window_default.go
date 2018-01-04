// TODO: build flags, this is linux/unix

package driver

import "github.com/jmigpin/editor/driver/xgbutil/xwindow"

func NewWindow() (Window, error) {
	return xwindow.NewWindow()
}
