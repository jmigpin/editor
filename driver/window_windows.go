// +build windows,!xproto

package driver

import "fmt"

//import "github.com/jmigpin/editor/driver/windriver"

func NewWindow() (Window, error) {
	//return windriver.NewWindow()
	return nil, fmt.Errorf("Please compile with '-tags=xproto' to compile the version with xserver support. The native version when released will not need this tag.")
}
