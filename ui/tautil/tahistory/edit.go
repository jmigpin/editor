package tahistory

type Edit struct {
	ostr, str string
	strEdit   *StrEdit
}

func NewEdit(str string) *Edit {
	return &Edit{ostr: str, str: str, strEdit: &StrEdit{}}
}
func (he *Edit) Str() string {
	return he.str
}
func (he *Edit) Insert(index int, istr string) {
	he.str = he.strEdit.Insert(he.str, index, istr)
}
func (he *Edit) Delete(index, index2 int) {
	he.str = he.strEdit.Delete(he.str, index, index2)
}
func (he *Edit) Close() (string, *StrEdit, bool) {
	changed := he.str != he.ostr
	if !changed {
		return "", nil, false
	}
	return he.str, he.strEdit, true
}
