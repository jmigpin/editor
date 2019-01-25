package debug

import (
	"encoding/gob"
	"fmt"
)

func init() {
	// register structs to be able to encode/decode from interface{}
	gob.Register(&ReqFilesDataMsg{})
	gob.Register(&FilesDataMsg{})
	gob.Register(&ReqStartMsg{})
	gob.Register(&LineMsg{})

	gob.Register(&ItemValue{})
	gob.Register(&ItemList{})
	gob.Register(&ItemList2{})
	gob.Register(&ItemAssign{})
	gob.Register(&ItemSend{})
	gob.Register(&ItemCall{})
	gob.Register(&ItemIndex{})
	gob.Register(&ItemIndex2{})
	gob.Register(&ItemKeyValue{})
	gob.Register(&ItemBinary{})
	gob.Register(&ItemUnary{})
	gob.Register(&ItemParen{})
	gob.Register(&ItemLiteral{})
	gob.Register(&ItemBranch{})
	gob.Register(&ItemAnon{})
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

type V interface{}
type Item interface{}
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
	Result   Item
	Args     *ItemList
	Name     string
	Entering bool
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
type ItemParen struct {
	X Item
}
type ItemLiteral struct {
	Fields *ItemList
}
type ItemBranch struct{}
type ItemAnon struct{}

//----------

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

// ItemCall: entering
func ICe(name string, args ...Item) Item {
	return &ItemCall{Name: name, Entering: true, Args: IL(args...)}
}

// ItemIndex
func II(result, expr, index Item) Item {
	return &ItemIndex{Result: result, Expr: expr, Index: index}
}
func II2(result, expr, low, high, max Item, slice3 bool) Item {
	return &ItemIndex2{Result: result, Expr: expr, Low: low, High: high, Max: max, Slice3: slice3}
}

// ItemKeyValue
func KV(key, value Item) Item {
	return &ItemKeyValue{Key: key, Value: value}
}

// ItemBinary
func IB(result Item, op int, x, y Item) Item {
	return &ItemBinary{Result: result, Op: op, X: x, Y: y}
}

// ItemUnary
func IU(result Item, op int, x Item) Item {
	return &ItemUnary{Result: result, Op: op, X: x}
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

// ItemAnon
func IAn() Item {
	return &ItemAnon{}
}
