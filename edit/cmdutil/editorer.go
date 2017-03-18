package cmdutil

import (
	"github.com/jmigpin/editor/edit/toolbardata"
	"github.com/jmigpin/editor/ui"
)

type Editorer interface {
	Error(error)
	UI() *ui.UI
	FindRow(name string) (*ui.Row, bool)
	FindRowOrCreate(name string) *ui.Row
	FindRowOrCreateInColFromFilepath(filepath string, col *ui.Column) (*ui.Row, error)
	RowToolbarStringData(*ui.Row) *toolbardata.StringData
	FilepathContent(filepath string) (string, error)
	FilesWatcherAdd(filename string) error
	FilesWatcherRemove(filename string) error
	ActiveColumn() *ui.Column
	NewRow(*ui.Column) *ui.Row
	RowCtx() *RowCtx
}
