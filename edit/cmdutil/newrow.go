package cmdutil

func NewRow(ed Editori) {
	row := ed.NewRow(ed.ActiveColumn())
	row.Square.WarpPointer()
}
