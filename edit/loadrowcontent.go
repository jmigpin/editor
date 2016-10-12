package edit

import (
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/jmigpin/editor/ui"
)

func loadRowContent(ed *Editor, row *ui.Row) error {
	tsd := ed.rowToolbarStringData(row)
	v := tsd.FirstPart()
	fi, err := os.Stat(v)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		// list directory
		f, err := os.Open(v)
		if err != nil {
			return err
		}
		fis, err := f.Readdir(-1)
		if err != nil {
			return err
		}
		sort.Sort(ByListOrder(fis))
		s := ""
		for _, fi := range fis {
			name := fi.Name()
			if fi.IsDir() {
				name += "/"
			}
			s += name + "\n"
		}
		row.TextArea.SetText(s)
		row.TextArea.SetSelectionOn(false)
		row.Square.SetDirty(false)
		row.Square.SetCold(false)
		return nil
	}
	// file content
	b, err := ioutil.ReadFile(v)
	if err != nil {
		return err
	}
	row.TextArea.SetText(string(b))
	row.TextArea.SetSelectionOn(false)
	row.Square.SetDirty(false)
	row.Square.SetCold(false)
	return nil
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
