package cmdutil

func NewRow(ed Editori) {
	col := ed.ActiveColumn()
	row := col.NewRow()
	row.Square.WarpPointer()
}
