package debug

import (
	"fmt"
)

func registerStructsForProtoConn() {
	reg := func(v any) {
		edReg.register(v)
	}

	reg(&ReqFilesDataMsg{})
	reg(&ReqStartMsg{})

	reg(&FilesDataMsg{})
	reg(&AnnotatorFileData{})

	reg(&OffsetMsgs{})
	reg(&OffsetMsg{})

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
	reg(&ItemNotAnn{})
}

//----------
//----------
//----------

type ReqFilesDataMsg struct{}
type ReqStartMsg struct{}

type FilesDataMsg struct {
	Data []*AnnotatorFileData
}

//----------

type AnnotatorFileData struct {
	// decl order matters: used by the config generator to fill the struct
	FileIndex     AfdFileIndex
	NMsgIndexes AfdMsgIndex
	Filename      string
	FileSize      AfdFileSize
	FileHash      []byte
}

type AfdFileIndex = uint16
type AfdFileSize = uint32
type AfdMsgIndex = uint32 // uint16 enough?

//----------

type OffsetMsgs []*OffsetMsg

type OffsetMsg struct {
	FileIndex  AfdFileIndex
	MsgIndex AfdMsgIndex
	Offset     AfdFileSize
	Item       Item
}

//----------

type Item interface {
	isItem()
}

type ItemValue struct {
	Item
	Str string
}
type ItemList struct { // separated by ","
	Item
	List []Item
}
type ItemList2 ItemList // separated by ";"
type ItemAssign struct {
	Item
	Lhs *ItemList
	Op  uint16
	Rhs *ItemList
}
type ItemSend struct {
	Item
	Chan, Value Item
}

type ItemCallEnter struct {
	Item
	Fun  Item
	Args *ItemList
}
type ItemCall struct {
	Item
	Enter  *ItemCallEnter
	Result Item
}

type ItemIndex struct {
	Item
	Expr   Item
	Index  Item
	Result Item
}
type ItemIndex2 struct {
	Item
	Expr           Item
	Low, High, Max Item
	Slice3         bool // 2 colons present
	Result         Item
}
type ItemKeyValue struct {
	Item
	Key   Item
	Value Item
}
type ItemSelector struct {
	Item
	X      Item
	Sel    Item
	Result Item
}
type ItemTypeAssert struct {
	Item
	X        Item
	Type     Item
	Result   Item
	IsSwitch bool
}
type ItemBinary struct {
	Item
	X      Item
	Op     uint16
	Y      Item
	Result Item
}

type ItemUnaryEnter struct {
	Item
	Op uint16
	X  Item
}
type ItemUnary struct {
	Item
	Enter  *ItemUnaryEnter
	Result Item
}

type ItemLiteral struct {
	Item
	Fields *ItemList
}
type ItemParen struct {
	Item
	X Item
}
type ItemLabel struct {
	Item
	Reason string // ex: "for" init not debugged
}
type ItemNotAnn struct {
	Item
	Reason string // not annotated (ex: String(), Error())
}
type ItemBranch struct {
	Item
}
type ItemStep struct {
	Item
}
type ItemAnon struct {
	Item
}

//----------
//----------
//----------

// ItemValue: interface (ex: int=1, string="1")
func IVi(v any) Item {
	return &ItemValue{Str: stringify(v)}
}

// ItemValue: string (ex: value of "?" is presented without quotes)
func IVs(s string) Item {
	return &ItemValue{Str: s}
}

// ItemValue: typeof
func IVt(v any) Item {
	s := fmt.Sprintf("%T", v)
	return &ItemValue{Str: s}
}

// ItemValue: range
func IVr(v int) Item {
	s := fmt.Sprintf("range(%v=len())", v)
	return &ItemValue{Str: s}
}

//// ItemValue: printf
//// usage: f(ctx,"IVp", basicLitStringQ("%v"), basicLitInt(1))
//func IVp(format string, args ...interface{}) Item {
//	return &ItemValue{Str: fmt.Sprintf(format, args...)}
//}

// ItemList ("," and ";")
func IL(u ...Item) *ItemList {
	return &ItemList{List: u}
}
func IL2(u ...Item) Item {
	return &ItemList2{List: u}
}

// ItemAssign
func IA(lhs *ItemList, op int, rhs *ItemList) Item {
	return &ItemAssign{Lhs: lhs, Op: uint16(op), Rhs: rhs}
}

// ItemSend
func IS(ch, value Item) Item {
	return &ItemSend{Chan: ch, Value: value}
}

// ItemCall: enter
func ICe(fun Item, args *ItemList) Item {
	return &ItemCallEnter{Fun: fun, Args: args}
}

// ItemCall
func IC(enter Item, result Item) Item {
	u := enter.(*ItemCallEnter)
	return &ItemCall{Enter: u, Result: result}
}

// ItemIndex
func II(expr, index, result Item) Item {
	return &ItemIndex{Expr: expr, Index: index, Result: result}
}
func II2(expr, low, high, max Item, slice3 bool, result Item) Item {
	return &ItemIndex2{Expr: expr, Low: low, High: high, Max: max, Slice3: slice3, Result: result}
}

// ItemKeyValue
func IKV(key, value Item) Item {
	return &ItemKeyValue{Key: key, Value: value}
}

// ItemSelector
func ISel(x, sel, result Item) Item {
	return &ItemSelector{X: x, Sel: sel, Result: result}
}

// ItemTypeAssert
func ITA(x, t, result Item, isSwitch bool) Item {
	return &ItemTypeAssert{X: x, Type: t, Result: result, IsSwitch: isSwitch}
}

// ItemBinary
func IB(x Item, op int, y Item, result Item) Item {
	return &ItemBinary{X: x, Op: uint16(op), Y: y, Result: result}
}

// ItemUnary: enter
func IUe(op int, x Item) Item {
	return &ItemUnaryEnter{Op: uint16(op), X: x}
}

// ItemUnary
func IU(enter Item, result Item) Item {
	u := enter.(*ItemUnaryEnter)
	return &ItemUnary{Enter: u, Result: result}
}

// ItemParen
func IP(x Item) Item {
	return &ItemParen{X: x}
}

// ItemLiteral
func ILit(fields *ItemList) Item {
	return &ItemLiteral{Fields: fields}
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
func ILa(reason string) Item {
	return &ItemLabel{Reason: reason}
}

// ItemNotAnn
func INAnn(reason string) Item {
	return &ItemNotAnn{Reason: reason}
}

//----------
//----------
//----------
