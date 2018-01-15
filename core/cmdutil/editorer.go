package cmdutil

import (
	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
)

type Editorer interface {
	Error(error)
	Errorf(string, ...interface{})
	Messagef(f string, a ...interface{})

	UI() *ui.UI

	NewERowerBeforeRow(string, *ui.Column, *ui.Row) ERower
	ERowers() []ERower
	FindERowers(string) []ERower
	ActiveERower() (ERower, bool)

	GoodColumnRowPlace() (col *ui.Column, next *ui.Row)

	HomeVars() *toolbardata.HomeVars
}
