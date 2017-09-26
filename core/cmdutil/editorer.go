package cmdutil

import (
	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
)

type Editorer interface {
	Error(error)
	Errorf(string, ...interface{})
	UI() *ui.UI

	NewERowBeforeRow(string, *ui.Column, *ui.Row) ERower
	FindERow(string) (ERower, bool)
	ERows() []ERower
	ActiveERow() (ERower, bool)

	GoodColumnRowPlace() (col *ui.Column, next *ui.Row)

	HomeVars() *toolbardata.HomeVars
}
