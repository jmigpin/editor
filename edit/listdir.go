package edit

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/jmigpin/editor/ui"
)

func ListDirTreeEd(ed *Editor, row *ui.Row, tree, hidden bool) {
	tsd := ed.RowToolbarStringData(row)
	fp := tsd.FirstPartFilepath()
	s, err := ListDirTree(fp, tree, hidden)
	if err != nil {
		ed.Error(err)
		return
	}
	row.TextArea.ClearStr(s, true)
	row.Square.SetDirty(false)
	row.Square.SetCold(false)
}

func ListDirTree(filepath string, tree, hidden bool) (string, error) {
	s, err := listDirTree2(filepath, "", tree, hidden)
	if err != nil {
		return "", err
	}
	return "../\n" + s, nil
}
func listDirTree2(filepath, addedFilepath string, tree, hidden bool) (string, error) {
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
			s2, err := listDirTree2(filepath, afp, tree, hidden)
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
