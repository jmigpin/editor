package godebug

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/jmigpin/editor/core/godebug/debug"
)

func StringifyItem(item debug.Item) string {
	is := NewItemStringifier()
	is.stringify(item)
	return is.b.String()
}
func StringifyItemFull(item debug.Item) string {
	is := NewItemStringifier()
	is.fullStr = true
	is.stringify(item)
	return is.b.String()
}

//----------

type ItemStringifier struct {
	b       *strings.Builder
	fullStr bool
}

func NewItemStringifier() *ItemStringifier {
	is := &ItemStringifier{}
	is.b = &strings.Builder{}
	return is
}

func (is *ItemStringifier) p(s string) {
	is.b.WriteString(s)
}

//----------

//func (is *ItemStringifier) captureStringify(item debug.Item) (start, end int, s string) {
//	start = len(is.Str)
//	is.stringify(item)
//	end = len(is.Str)
//	return start, end, is.Str[start:end]
//}

//func (is *ItemStringifier) stringify(item debug.Item) {
//// capture value
//start := len(is.Str)
//defer func() {
//	end := len(is.Str)
//	if is.Offset >= start && is.Offset < end {
//		s := is.Str[start:end]
//		if is.OffsetValueString == "" || len(s) < len(is.OffsetValueString) {
//			is.OffsetValueString = s
//		}
//	}
//}()

//is.stringify2(item)
//}

//----------

func (is *ItemStringifier) stringify(item debug.Item) {
	is.stringify2(item)
}

func (is *ItemStringifier) stringify2(item debug.Item) {
	// NOTE: the string append is done sequentially to allow to detect where the strings are positioned (if later supported)

	//log.Printf("stringifyitem: %T", item)

	switch t := item.(type) {
	case *debug.ItemValue:
		if is.fullStr {
			is.p(t.Str)
		} else {
			is.p(debug.SprintCutCheckQuote(20, t.Str))
		}

	case *debug.ItemList: // ex: func args list
		if t == nil {
			break
		}
		for i, e := range t.List {
			if i > 0 {
				is.p(", ")
			}
			is.stringify(e)
		}

	case *debug.ItemList2:
		if t == nil {
			break
		}
		for i, e := range t.List {
			if i > 0 {
				is.p("; ")
			}
			is.stringify(e)
		}

	case *debug.ItemAssign:
		is.stringify(t.Lhs)

		// it's misleading to get a "2 += 1", better to just show "2 = 1"
		//is.p(" " + token.Token(t.Op).String() + " ")
		is.p(" ")
		switch t2 := token.Token(t.Op); t2 {
		case token.ADD_ASSIGN, token.SUB_ASSIGN,
			token.MUL_ASSIGN, token.QUO_ASSIGN,
			token.REM_ASSIGN,
			token.INC, token.DEC:
			is.p("=")
		default:
			is.p(t2.String())
		}
		is.p(" ")

		is.stringify(t.Rhs)

	case *debug.ItemSend:
		is.stringify(t.Chan)
		is.p(" <- ")
		is.stringify(t.Value)

	case *debug.ItemCallEnter:
		is.p("=> ")
		is.stringify(t.Fun)
		is.p("(")
		is.stringify(t.Args)
		is.p(")")
	case *debug.ItemCall:
		_ = is.result(t.Result)
		is.stringify(t.Enter.Fun)
		is.p("(")
		is.stringify(t.Enter.Args)
		is.p(")")

	case *debug.ItemIndex:
		_ = is.result(t.Result)
		if t.Expr != nil {
			//switch t2 := t.Expr.(type) {
			//case string:
			//	is.p( t2
			//default:
			//	is.p( "("
			//	is.stringify(t.Expr)
			//	is.p( ")"
			//}
			is.stringify(t.Expr)
		}
		is.p("[")
		if t.Index != nil {
			is.stringify(t.Index)
		}
		is.p("]")

	case *debug.ItemIndex2:
		_ = is.result(t.Result)
		if t.Expr != nil {
			//switch t2 := t.Expr.(type) {
			//case string:
			//	is.p( t2
			//default:
			//	is.p( "("
			//	is.stringify(t.Expr)
			//	is.p( ")"
			//}
			is.stringify(t.Expr)
		}
		is.p("[")
		if t.Low != nil {
			is.stringify(t.Low)
		}
		is.p(":")
		if t.High != nil {
			is.stringify(t.High)
		}
		if t.Slice3 {
			is.p(":")
		}
		if t.Max != nil {
			is.stringify(t.Max)
		}
		is.p("]")

	case *debug.ItemKeyValue:
		is.stringify(t.Key)
		is.p(":")
		is.stringify(t.Value)

	case *debug.ItemSelector:
		is.p("(")
		is.stringify(t.X)
		is.p(").")
		is.stringify(t.Sel)

	case *debug.ItemTypeAssert:
		is.stringify(t.Type)
		is.p("=type(")
		is.stringify(t.X)
		is.p(")")

	case *debug.ItemBinary:
		showRes := is.result(t.Result)
		if showRes {
			is.p("(")
		}
		is.stringify(t.X)
		is.p(" " + token.Token(t.Op).String() + " ")
		is.stringify(t.Y)
		if showRes {
			is.p(")")
		}

	case *debug.ItemUnaryEnter:
		is.p("=> ")
		is.p(token.Token(t.Op).String())
		is.stringify(t.X)
	case *debug.ItemUnary:
		_ = is.result(t.Result)
		is.p(token.Token(t.Enter.Op).String())
		is.stringify(t.Enter.X)

	case *debug.ItemParen:
		is.p("(")
		is.stringify(t.X)
		is.p(")")

	case *debug.ItemLiteral:
		is.p("{") // other runes: τ, s // ex: A{a:1}, []byte{1,2}
		if t != nil {
			is.stringify(t.Fields)
		}
		is.p("}")

	case *debug.ItemAnon:
		is.p("_")

	case *debug.ItemBranch:
		is.p("#")
	case *debug.ItemStep:
		is.p("#")
	case *debug.ItemLabel:
		is.p("#")
		if t.Reason != "" {
			is.p(" label: " + t.Reason)
		}
	case *debug.ItemNotAnn:
		is.p(fmt.Sprintf("# not annotated: %v", t.Reason))

	default:
		is.p(fmt.Sprintf("[TODO:(%T)%v]", item, item))
	}
}

//----------

func (is *ItemStringifier) result(result debug.Item) bool {
	if result == nil {
		return false
	}

	isList := false
	if _, ok := result.(*debug.ItemList); ok {
		isList = true
	}
	if isList {
		is.p("(")
	}

	is.stringify(result)

	if isList {
		is.p(")")
	}

	is.p("=") // other runes: ≡ // nice, but not all fonts have it defined

	return true
}
