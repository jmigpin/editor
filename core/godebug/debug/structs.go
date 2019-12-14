package debug

import (
	"fmt"
)

func init() {
	// register structs to be able to encode/decode from interface{}

	reg := RegisterStructure

	reg(&ReqFilesDataMsg{})
	reg(&FilesDataMsg{})
	reg(&ReqStartMsg{})
	reg(&LineMsg{})
	reg([]*LineMsg{})

	reg(&ItemValue{})
	reg(&ItemList{})
	reg(&ItemList2{})
	reg(&ItemAssign{})
	reg(&ItemSend{})
	reg(&ItemCall{})
	reg(&ItemCallEnter{})
	reg(&ItemIndex{})
	reg(&ItemIndex2{})
	reg(&ItemKeyValue{})
	reg(&ItemSelector{})
	reg(&ItemTypeAssert{})
	reg(&ItemBinary{})
	reg(&ItemUnary{})
	reg(&ItemUnaryEnter{})
	reg(&ItemParen{})
	reg(&ItemLiteral{})
	reg(&ItemBranch{})
	reg(&ItemStep{})
	reg(&ItemAnon{})
	reg(&ItemLabel{})
}

//----------

type ReqFilesDataMsg struct{}
type ReqStartMsg struct{}

//----------

type LineMsg struct {
	FileIndex  int
	DebugIndex int
	Offset     int
	Item       Item
}

type FilesDataMsg struct {
	Data []*AnnotatorFileData
}

type AnnotatorFileData struct {
	FileIndex int
	DebugLen  int
	Filename  string
	FileSize  int
	FileHash  []byte
}

//----------

type Item interface {
}
type ItemValue struct {
	Str string
}
type ItemList struct { // separated by ","
	List []Item
}
type ItemList2 struct { // separated by ";"
	List []Item
}
type ItemAssign struct {
	Lhs, Rhs *ItemList
}
type ItemSend struct {
	Chan, Value Item
}
type ItemCall struct {
	Name   string
	Args   *ItemList
	Result Item
}
type ItemCallEnter struct {
	Name string
	Args *ItemList
}
type ItemIndex struct {
	Result Item
	Expr   Item
	Index  Item
}
type ItemIndex2 struct {
	Result         Item
	Expr           Item
	Low, High, Max Item
	Slice3         bool // 2 colons present
}
type ItemKeyValue struct {
	Key   Item
	Value Item
}
type ItemSelector struct {
	X   Item
	Sel Item
}
type ItemTypeAssert struct {
	X    Item
	Type Item
}
type ItemBinary struct {
	Result Item
	Op     int
	X, Y   Item
}
type ItemUnary struct {
	Result Item
	Op     int
	X      Item
}
type ItemUnaryEnter struct {
	Op int
	X  Item
}
type ItemParen struct {
	X Item
}
type ItemLiteral struct {
	Fields *ItemList
}
type ItemBranch struct{}
type ItemStep struct{}
type ItemAnon struct{}
type ItemLabel struct{}

//----------

type V interface{}

// ItemValue
func IV(v V) Item {
	return &ItemValue{Str: stringifyV(v)}
}

// ItemValue: raw string
func IVs(s string) Item {
	return &ItemValue{Str: s}
}

// ItemValue: typeof
func IVt(v V) Item {
	return &ItemValue{Str: fmt.Sprintf("%T", v)}
}

// ItemValue: len
func IVl(v V) Item {
	return &ItemValue{Str: fmt.Sprintf("%v=len()", v)}
}

// ItemList ("," and ";")
func IL(u ...Item) *ItemList {
	return &ItemList{List: u}
}
func IL2(u ...Item) Item {
	return &ItemList2{List: u}
}

// ItemAssign
func IA(lhs, rhs *ItemList) Item {
	return &ItemAssign{Lhs: lhs, Rhs: rhs}
}

// ItemSend
func IS(ch, value Item) Item {
	return &ItemSend{Chan: ch, Value: value}
}

// ItemCall
func IC(name string, result Item, args ...Item) Item {
	return &ItemCall{Name: name, Result: result, Args: IL(args...)}
}

// ItemCall: enter
func ICe(name string, args ...Item) Item {
	return &ItemCallEnter{Name: name, Args: IL(args...)}
}

// ItemIndex
func II(result, expr, index Item) Item {
	return &ItemIndex{Result: result, Expr: expr, Index: index}
}
func II2(result, expr, low, high, max Item, slice3 bool) Item {
	return &ItemIndex2{Result: result, Expr: expr, Low: low, High: high, Max: max, Slice3: slice3}
}

// ItemKeyValue
func IKV(key, value Item) Item {
	return &ItemKeyValue{Key: key, Value: value}
}

// ItemSelector
func ISel(x, sel Item) Item {
	return &ItemSelector{X: x, Sel: sel}
}

// ItemTypeAssert
func ITA(x, t Item) Item {
	return &ItemTypeAssert{X: x, Type: t}
}

// ItemBinary
func IB(result Item, op int, x, y Item) Item {
	return &ItemBinary{Result: result, Op: op, X: x, Y: y}
}

// ItemUnary
func IU(result Item, op int, x Item) Item {
	return &ItemUnary{Result: result, Op: op, X: x}
}

// ItemUnary: enter
func IUe(op int, x Item) Item {
	return &ItemUnaryEnter{Op: op, X: x}
}

// ItemParen
func IP(x Item) Item {
	return &ItemParen{X: x}
}

// ItemLiteral
func ILit(fields ...Item) Item {
	return &ItemLiteral{Fields: IL(fields...)}
}

// ItemBranch
func IBr() Item {
	return &ItemBranch{}
}

// ItemStep
func ISt() Item {
	return &ItemStep{}
}

// ItemAnon
func IAn() Item {
	return &ItemAnon{}
}

// ItemLabel
func ILa() Item {
	return &ItemLabel{}
}
