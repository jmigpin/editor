package cmdutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/ui/tautil/tahistory"
	"github.com/jmigpin/editor/util/uiutil"
)

type EditorerTester struct {
	Editorer
	ui *ui.UI
}

func NewEditorerTester() *EditorerTester {
	//ui0, err := ui.NewUI(make(chan<- interface{}), "test")
	//if err != nil {
	//	panic(err)
	//}
	ui0 := &ui.UI{BasicUI: &uiutil.BasicUI{}}
	return &EditorerTester{ui: ui0}
}

func (ed *EditorerTester) Messagef(f string, args ...interface{}) {
	if !strings.HasSuffix(f, "\n") {
		f += "\n"
	}
	fmt.Printf(f, args...)
}
func (ed *EditorerTester) Errorf(f string, args ...interface{}) {
	ed.Messagef("error: "+f, args...)
}
func (ed *EditorerTester) Error(err error) {
	ed.Errorf(err.Error())
}
func (ed *EditorerTester) UI() *ui.UI {
	return ed.ui
}

type ERowerTester struct {
	ERower
	ed    *EditorerTester
	dir   string
	isDir bool
}

func NewERowerTester() *ERowerTester {
	erow := &ERowerTester{}
	erow.ed = NewEditorerTester()
	return erow
}
func (erow *ERowerTester) Ed() Editorer {
	return erow.ed
}
func (erow *ERowerTester) Dir() string {
	return erow.dir
}
func (erow *ERowerTester) IsDir() bool {
	return erow.isDir
}
func (erow *ERowerTester) Row() *ui.Row {
	return &ui.Row{
		TextArea: &ui.TextArea{
			History: tahistory.NewHistory(8),
		},
	}
}

func (erow *ERowerTester) StartExecState() context.Context {
	return context.Background()
}
func (erow *ERowerTester) TextAreaWriter() io.WriteCloser {
	return os.Stdout
}
