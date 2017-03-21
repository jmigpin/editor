package cmdutil

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
)

func ListDirEd(erow ERower, tree, hidden bool) {
	fp, fi, ok := erow.FileInfo()
	if !ok || !fi.IsDir() {
		return
	}
	s, err := ListDir(fp, tree, hidden)
	if err != nil {
		erow.Editorer().Error(err)
		return
	}
	erow.Row().TextArea.SetStrClear(s, false, false)
}

func ListDir(filepath string, tree, hidden bool) (string, error) {
	s, err := listDir2(filepath, "", tree, hidden)
	if err != nil {
		return "", err
	}
	return "../\n" + s, nil
}
func listDir2(filepath, addedFilepath string, tree, hidden bool) (string, error) {
	fp2 := path.Join(filepath, addedFilepath)
	f, err := os.Open(fp2)
	if err != nil {
		return "", err
	}
	fis, err := f.Readdir(-1)
	if err != nil {
		//return "", err
		return "", fmt.Errorf("listdirtree: %s: %s", fp2, err.Error())
	}
	sort.Sort(ByListOrder(fis))
	s := ""
	for _, fi := range fis {
		name := fi.Name()

		if !hidden && strings.HasPrefix(name, ".") {
			continue
		}

		name2 := path.Join(addedFilepath, name)
		if fi.IsDir() {
			name2 += "/"
		}
		s += name2 + "\n"

		if fi.IsDir() && tree {
			afp := path.Join(addedFilepath, name)
			s2, err := listDir2(filepath, afp, tree, hidden)
			if err != nil {
				return "", err
			}
			s += s2
		}
	}
	return s, nil
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
