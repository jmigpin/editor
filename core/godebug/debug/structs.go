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
	gob.Register(&ItemCall{})
	gob.Register(&ItemIndex{})
	gob.Register(&ItemIndex2{})
	gob.Register(&ItemBinary{})
	gob.Register(&ItemUnary{})
	gob.Register(&ItemParen{})
	gob.Register(&ItemLiteral{})
	gob.Register(&ItemBranch{})
	gob.Register(&ItemAnon{})
}

type LineMsg struct {
	FileIndex  int
	DebugIndex int
	Offset     int
	Item       Item
}

type FilesDataMsg struct {
	Data []*AnnotatorFileData
}

type ReqFilesDataMsg struct{}
type ReqStartMsg struct{}

type AnnotatorFileData struct {
	FileIndex int
	DebugLen  int
	Filename  string
	FileSize  int
	FileHash  []byte
}

//----------------

type V interface{}

func stringifyV(v V) string {
	str := ""
	switch t := v.(type) {
	case nil:
		return "nil"
	case string:
		str = fmt.Sprintf("%q", t)

	case fmt.Stringer, error:
		str = fmt.Sprintf("≈(%q)", t)

	case float32, float64:
		u := fmt.Sprintf("%f", t)

		// reduce trailing zeros
		j := 0
		for i := len(u) - 1; i >= 0; i-- {
			if u[i] == '0' {
				j++
				continue
			}
			break
		}

		str = u[:len(u)-j]

	default:
		str = fmt.Sprintf("%v", v)
	}

	return ReduceStr(str, 256)
}

func ReduceStr(str string, max int) string {
	if len(str) > max {
		h := max / 2
		str = str[:h] + "◦◦◦" + str[len(str)-h:]
	}
	return str
}

//----------------

type Item interface{}
type ItemValue struct {
	Str string
}
type ItemList struct {
	List []Item
}
type ItemList2 struct {
	List []Item
}
type ItemAssign struct {
	Lhs, Rhs *ItemList
}
type ItemCall struct {
	Result Item
	Args   *ItemList
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

//----------------

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

// ItemCall
func IC(result Item, args ...Item) Item {
	return &ItemCall{Result: result, Args: IL(args...)}
}

// ItemIndex
func II(result, expr, index Item) Item {
	return &ItemIndex{Result: result, Expr: expr, Index: index}
}
func II2(result, expr, low, high, max Item) Item {
	return &ItemIndex2{Result: result, Expr: expr, Low: low, High: high, Max: max}
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
