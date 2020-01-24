// +build windows,!xproto

package driver

import (
	"github.com/jmigpin/editor/driver/windriver"
)

func NewWindow2() (Window2, error) {
	return windriver.NewWindow2()
}
