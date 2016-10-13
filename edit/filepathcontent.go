package edit

import (
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

func filepathContent(filepath string) (string, error) {
	fi, err := os.Stat(filepath)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		// list directory
		f, err := os.Open(filepath)
		if err != nil {
			return "", err
		}
		fis, err := f.Readdir(-1)
		if err != nil {
			return "", err
		}
		sort.Sort(ByListOrder(fis))
		s := "../\n"
		for _, fi := range fis {
			name := fi.Name()
			if fi.IsDir() {
				name += "/"
			}
			s += name + "\n"
		}
		return s, nil
	}
	// file content
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type ByListOrder []os.FileInfo

func (a ByListOrder) Len() int {
	return len(a)
}
func (a ByListOrder) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByListOrder) Less(i, j int) bool {
	ei := a[i]
	ej := a[j]
	iname := strings.ToLower(ei.Name())
	jname := strings.ToLower(ej.Name())
	if ei.IsDir() && ej.IsDir() {
		return iname < jname
	}
	if ei.IsDir() {
		return true
	}
	if ej.IsDir() {
		return false
	}
	return iname < jname
}
