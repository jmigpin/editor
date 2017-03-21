package edit

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/jmigpin/editor/edit/cmdutil"
)

func filepathContent(filepath string) (string, error) {
	// row special name
	specialName := len(filepath) >= 1 && filepath[0] == '+'
	if specialName {
		return "", nil
	}
	// empty
	empty := strings.TrimSpace(filepath) == ""
	if empty {
		return "", nil
	}
	// filepath
	fi, err := os.Stat(filepath)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return cmdutil.ListDir(filepath, false, true)
	}
	// file content
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
