// TODO: build flags, this is linux/unix

package driver

import (
	"github.com/jmigpin/editor/xgbutil/evreg"
	"github.com/jmigpin/editor/xgbutil/xwindow"
)

func NewDriverWindow(evReg *evreg.Register) (Window, error) {
	return xwindow.NewWindow(evReg)
}
