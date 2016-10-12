package edit

//func openPathAtCol(ed *Editor, p string, col *ui.Column) (*ui.Row, error) {
//fi, err := os.Stat(p)
//if err != nil {
//return nil, err
//}
//p2 := toolbar.ReplaceHomeVar(p)
//if fi.IsDir() {
//// always open a new row, even if other exists
//row := col.NewRow()
//row.Toolbar.SetText(p2 + " | Reload")
//err = loadRowContent(ed, row)
//return row, err
//}
//// it's a file
//row, ok := ed.findRow(p2)
//if !ok {
//row = col.NewRow()
//}
//row.Toolbar.SetText(p2 + " | Reload")
//err = loadRowContent(ed, row)
//return row, err
//}
