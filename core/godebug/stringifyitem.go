package godebug

import (
	"fmt"
	"go/token"
	"log"

	"github.com/jmigpin/editor/core/godebug/debug"
)

func StringifyItem(item debug.Item) string {
	is := ItemStringifier{Offset: -1}
	is.stringify(item)
	return is.Str
}

func StringifyItemOffset(item debug.Item, offset int) string {
	is := ItemStringifier{Offset: offset}
	is.stringify(item)
	return is.OffsetValueString
}

//var theta = string(rune(952))
//string(rune(8592))
//var eff = string(rune(402))
//string(rune(8801))
var leftArrow = "←"
var threeLines = "≡"

type ItemStringifier struct {
	Offset            int
	OffsetValueString string
	Str               string
	depth             int
}

func (is *ItemStringifier) stringify(item debug.Item) {
	is.depth++
	is.stringify2(item)
	is.depth--
}

func (is *ItemStringifier) stringify2(item debug.Item) {
	// NOTE: the string append is done sequentially to allow to detect where the strings are positioned to correctly set "OffsetValueString"

	//log.Printf("stringifyitem: %T", item)

	switch t := item.(type) {

	case *debug.ItemValue:
		start := len(is.Str)
		is.Str += debug.ReduceStr(t.Str, 20)
		end := len(is.Str)
		if is.Offset >= start && is.Offset < end {
			is.OffsetValueString = t.Str
		}

	case *debug.ItemList:
		for i, e := range t.List {
			if i > 0 {
				is.Str += ", "
			}
			is.stringify(e)
		}

	case *debug.ItemList2:
		for i, e := range t.List {
			if i > 0 {
				is.Str += "; "
			}
			is.stringify(e)
		}

	case *debug.ItemLiteral:
		is.Str += "τ("
		is.stringify(t.Fields)
		is.Str += ")"

	case *debug.ItemAssign:
		simplify := false

		//if is.depth == 1 {
		//	if len(t.Rhs.List) == 1 {
		//		switch t.Rhs.List[0].(type) {
		//		case *debug.ItemCall,
		//			*debug.ItemBinary,
		//			*debug.ItemIndex,
		//			*debug.ItemIndex2,
		//			*debug.ItemValue:
		//			simplify = true
		//		}
		//	}
		//}

		if simplify {
			is.depth -= 2
			is.stringify(t.Rhs)
			is.depth += 2
		} else {
			is.stringify(t.Lhs)
			is.Str += " :≡ "
			is.stringify(t.Rhs)
		}

	case *debug.ItemCall:
		showFunc := true
		//showFunc := (t.Args != nil && len(t.Args.List) > 0) || t.Result == nil
		_ = is.result(t.Result)
		if showFunc {
			is.Str += "λ"
			is.Str += "("
			is.stringify(t.Args)
			is.Str += ")"
		}

	case *debug.ItemUnary:
		is.Str += token.Token(t.Op).String()
		is.stringify(t.X)

	case *debug.ItemBinary:
		// show result
		showRes := true
		//showRes := false
		//if t.Result != nil {
		//	tok := token.Token(t.Op)
		//	switch tok {
		//	case token.MUL, token.ADD, token.SUB, token.QUO, token.REM:
		//		showRes = true
		//	}
		//}

		if showRes {
			showRes = is.result(t.Result)
		}

		if showRes {
			is.Str += "("
		}
		is.stringify(t.X)
		is.Str += " " + token.Token(t.Op).String() + " "
		is.stringify(t.Y)
		if showRes {
			is.Str += ")"
		}

	case *debug.ItemIndex:
		_ = is.result(t.Result)
		if t.Expr != nil {
			is.Str += "("
			is.stringify(t.Expr)
			is.Str += ")"
		}
		is.Str += "["
		if t.Index != nil {
			is.stringify(t.Index)
		}
		is.Str += "]"

	case *debug.ItemIndex2:
		_ = is.result(t.Result)
		if t.Expr != nil {
			is.Str += "("
			is.stringify(t.Expr)
			is.Str += ")"
		}
		is.Str += "["
		if t.Low != nil {
			is.stringify(t.Low)
		}
		if t.High != nil || t.Max != nil {
			is.Str += ":"
		}
		if t.High != nil {
			is.stringify(t.High)
		}
		if t.Max != nil {
			is.Str += ":"
			is.stringify(t.Max)
		}
		is.Str += "]"

	case *debug.ItemParen:
		is.Str += "("
		is.stringify(t.X)
		is.Str += ")"

	case *debug.ItemBranch:
		is.Str += "←"

	default:
		is.Str += fmt.Sprintf("[[?: %v, %T]]", item, item)
		log.Printf("todo: stringifyItem")
	}
}

func (is *ItemStringifier) result(result debug.Item) bool {
	//isFirst := is.depth == 1
	if result != nil {
		is.stringify(result)
		//if isFirst {
		//	is.Str += " " + leftArrow + " "
		//} else {
		is.Str += "≡"
		//}
		return true
	}
	return false
}
