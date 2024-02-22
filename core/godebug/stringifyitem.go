package godebug

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/jmigpin/editor/core/godebug/debug"
)

func StringifyItem(item debug.Item) string {
	return StringifyItem2(item, 20)
}
func StringifyItem2(item debug.Item, valueStrLen int) string {
	is := NewItemStringifier()
	is.valueStrLen = valueStrLen
	is.stringify(item)
	return is.b.String()
}
func StringifyItemFull(item debug.Item) string {
	is := NewItemStringifier()
	is.valueStrLen = -1 // full str
	is.stringify(item)
	return is.b.String()
}

//----------

type ItemStringifier struct {
	b           *strings.Builder
	valueStrLen int // <0 = full str
}

func NewItemStringifier() *ItemStringifier {
	is := &ItemStringifier{}
	is.b = &strings.Builder{}
	is.valueStrLen = 20
	return is
}

func (is *ItemStringifier) p(s string) {
	is.b.WriteString(s)
}

//----------

func (is *ItemStringifier) stringify(item debug.Item) {
	//log.Printf("stringifyitem: %T", item)

	switch t := item.(type) {
	case *debug.ItemValue:
		if is.valueStrLen < 0 {
			is.p(t.Str)
		} else {
			is.p(debug.SprintCutCheckQuote(is.valueStrLen, t.Str))
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
		is.p(" ")
		is.p(token.Token(t.Op).String())
		is.p(" ")
		is.stringify(t.Rhs)

	//case *debug.ItemSendEnter: // TODO
	//	is.p("=> ")
	//	is.stringify(t.Chan)
	//	is.p(" <- ")
	//	is.stringify(t.Value)
	case *debug.ItemSend:
		//is.stringify(t.Enter.Chan)
		//is.p(" <- ")
		//is.stringify(t.Enter.Value)
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
		is.pResult(t.Result, func() {
			is.stringify(t.Enter.Fun)
			is.p("(")
			is.stringify(t.Enter.Args)
			is.p(")")
		})

	case *debug.ItemIndex:
		is.pResult(t.Result, func() {
			if t.Expr != nil {
				is.stringify(t.Expr)
			}
			is.p("[")
			if t.Index != nil {
				is.stringify(t.Index)
			}
			is.p("]")
		})

	case *debug.ItemIndex2:
		is.pResult(t.Result, func() {
			if t.Expr != nil {
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
		})

	case *debug.ItemKeyValue:
		if t.Key == nil {
			is.p("_")
		} else {
			is.stringify(t.Key)
		}
		is.p(":")
		is.stringify(t.Value)

	case *debug.ItemSelector:
		is.pResult(t.Result, func() {
			if t.X == nil {
				is.p("_") // this being here saves transfer bytes
			} else {
				is.stringify(t.X)
			}
			is.p(".")
			is.stringify(t.Sel)
		})

	case *debug.ItemTypeAssert:
		is.pResult(t.Result, func() {
			is.pResult(t.Type, func() {
				is.stringify(t.X)

				u := "_"
				if t.IsSwitch {
					u = "type"
				}
				s := fmt.Sprintf(".(%s)", u)
				is.p(s)
			})
		})

	case *debug.ItemBinary:
		is.pResult(t.Result, func() {
			is.stringify(t.X)
			is.p(" ")
			is.p(token.Token(t.Op).String())
			is.p(" ")
			is.stringify(t.Y)
		})

	case *debug.ItemUnaryEnter:
		is.p("=> ")
		is.p(token.Token(t.Op).String())
		is.stringify(t.X)
	case *debug.ItemUnary:
		is.pResult(t.Result, func() {
			is.p(token.Token(t.Enter.Op).String())
			if t.Enter.X != nil {
				is.stringify(t.Enter.X)
			}
		})

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
		if t.Reason != "" { // TODO: remove
			is.p(" label: " + t.Reason)
		}
	case *debug.ItemNotAnn:
		is.p(fmt.Sprintf("# not annotated: %v", t.Reason))

	default:
		is.p(fmt.Sprintf("[TODO:stringifyitem:%T,%v]", item, item))
	}
}

//----------

func (is *ItemStringifier) pResult(result debug.Item, fn func()) {
	if result == nil {
		fn()
		return
	}
	_, isList := result.(*debug.ItemList)
	if isList {
		is.p("(")
	}
	is.stringify(result)
	if isList {
		is.p(")")
	}
	is.p("=") // other runes: ≡ // nice, but not all fonts have it defined
	is.p("(")
	fn()
	is.p(")")
}
