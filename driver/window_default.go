// TODO: build flags, this is linux/unix

package driver

import (
	"github.com/jmigpin/editor/xgbutil/xwindow"
)

func NewWindow() (Window, error) {
	return xwindow.NewWindow()
}
