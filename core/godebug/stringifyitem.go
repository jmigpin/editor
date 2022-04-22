package godebug

import (
	"fmt"
	"go/token"

	"github.com/jmigpin/editor/core/godebug/debug"
)

//var SimplifyStringifyItem = true

func StringifyItem(item debug.Item) string {
	is := ItemStringifier{}
	is.stringify(item)
	return is.Str
}
func StringifyItemFull(item debug.Item) string {
	is := ItemStringifier{FullStr: true}
	is.stringify(item)
	return is.Str
}

// TODO: only option that required is.Str to be updated in order
//func StringifyItemOffset(item debug.Item, offset int) string {
//	is := ItemStringifier{Offset: offset}
//	is.stringify(item)
//	return is.OffsetValueString
//}

//----------

type ItemStringifier struct {
	Str            string
	FullStr        bool
	SimplifyResult string

	//Offset            int
	//OffsetValueString string
}

//----------

func (is *ItemStringifier) captureStringify(item debug.Item) (start, end int, s string) {
	start = len(is.Str)
	is.stringify(item)
	end = len(is.Str)
	return start, end, is.Str[start:end]
}

//----------

func (is *ItemStringifier) stringify(item debug.Item) {
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

	is.stringify2(item)
}

func (is *ItemStringifier) stringify2(item debug.Item) {
	// NOTE: the string append is done sequentially to allow to detect where the strings are positioned to correctly set "OffsetValueString" if trying to obtain the offset string

	//log.Printf("stringifyitem: %T", item)

	switch t := item.(type) {
	//case string:
	//	is.Str += t
	//case int:
	//	is.Str += fmt.Sprintf("%d", t)

	case *debug.ItemValue:
		is.FullStr = true
		if is.FullStr {
			is.Str += t.Str
		} else {
			//is.Str += debug.ReducedSprintf(20, "%s", t.Str)
			is.Str += debug.SprintCutCheckQuote(20, t.Str)
		}

	case *debug.ItemList: // ex: func args list
		if t == nil {
			break
		}
		for i, e := range t.List {
			if i > 0 {
				is.Str += ", "
			}
			is.stringify(e)
		}

	case *debug.ItemList2:
		if t == nil {
			break
		}
		for i, e := range t.List {
			if i > 0 {
				is.Str += "; "
			}
			is.stringify(e)
		}

	case *debug.ItemAssign:
		//if SimplifyStringifyItem {
		//	is.simplifyItemAssign(t)
		//} else {
		is.stringify(t.Lhs)
		is.Str += " " + token.Token(t.Op).String() + " "
		is.stringify(t.Rhs)
		//}

	case *debug.ItemSend:
		is.stringify(t.Chan)
		is.Str += " <- "
		is.stringify(t.Value)

	case *debug.ItemCallEnter:
		is.Str += "=> "
		is.stringify(t.Fun)
		is.Str += "("
		is.stringify(t.Args)
		is.Str += ")"
	case *debug.ItemCall:
		_ = is.result(t.Result)
		is.stringify(t.Enter.Fun)
		is.Str += "("
		is.stringify(t.Enter.Args)
		is.Str += ")"

	case *debug.ItemIndex:
		_ = is.result(t.Result)
		if t.Expr != nil {
			//switch t2 := t.Expr.(type) {
			//case string:
			//	is.Str += t2
			//default:
			//	is.Str += "("
			//	is.stringify(t.Expr)
			//	is.Str += ")"
			//}
			is.stringify(t.Expr)
		}
		is.Str += "["
		if t.Index != nil {
			is.stringify(t.Index)
		}
		is.Str += "]"

	case *debug.ItemIndex2:
		_ = is.result(t.Result)
		if t.Expr != nil {
			//switch t2 := t.Expr.(type) {
			//case string:
			//	is.Str += t2
			//default:
			//	is.Str += "("
			//	is.stringify(t.Expr)
			//	is.Str += ")"
			//}
			is.stringify(t.Expr)
		}
		is.Str += "["
		if t.Low != nil {
			is.stringify(t.Low)
		}
		is.Str += ":"
		if t.High != nil {
			is.stringify(t.High)
		}
		if t.Slice3 {
			is.Str += ":"
		}
		if t.Max != nil {
			is.stringify(t.Max)
		}
		is.Str += "]"

	case *debug.ItemKeyValue:
		is.stringify(t.Key)
		is.Str += ":"
		is.stringify(t.Value)

	case *debug.ItemSelector:
		is.Str += "("
		is.stringify(t.X)
		is.Str += ")."
		is.stringify(t.Sel)

	case *debug.ItemTypeAssert:
		is.stringify(t.Type)
		is.Str += "=type("
		is.stringify(t.X)
		is.Str += ")"

	case *debug.ItemBinary:
		showRes := is.result(t.Result)
		if showRes {
			is.Str += "("
		}
		is.stringify(t.X)
		is.Str += " " + token.Token(t.Op).String() + " "
		is.stringify(t.Y)
		if showRes {
			is.Str += ")"
		}

	case *debug.ItemUnaryEnter:
		is.Str += "=> "
		is.Str += token.Token(t.Op).String()
		is.stringify(t.X)
	case *debug.ItemUnary:
		_ = is.result(t.Result)
		//is.Str += token.Token(t.Op).String()
		//is.stringify(t.X)
		is.Str += token.Token(t.Enter.Op).String()
		is.stringify(t.Enter.X)

	case *debug.ItemParen:
		is.Str += "("
		is.stringify(t.X)
		is.Str += ")"

	case *debug.ItemLiteral:
		is.Str += "{" // other runes: τ, s // ex: A{a:1}, []byte{1,2}
		if t != nil {
			is.stringify(t.Fields)
		}
		is.Str += "}"

	case *debug.ItemAnon:
		is.Str += "_"

	case *debug.ItemBranch:
		is.Str += "#"
	case *debug.ItemStep:
		is.Str += "#"
	case *debug.ItemLabel:
		is.Str += "#"
		if t.Reason != "" {
			is.Str += " label: " + t.Reason
		}
	case *debug.ItemNotAnn:
		is.Str += fmt.Sprintf("# not annotated: %v", t.Reason)

	default:
		is.Str += fmt.Sprintf("[TODO:(%T)%v]", item, item)
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
		is.Str += "("
	}

	is.stringify(result)

	if isList {
		is.Str += ")"
	}

	is.Str += "=" // other runes: ≡

	return true
}

//----------

//func (is *ItemStringifier) simplifyItemAssign(t *debug.ItemAssign) {
//	s1, e1, str1 := is.captureStringify(t.Lhs)
//	is.Str += " := "
//	s2, e2, str2 := is.captureStringify(t.Rhs)
//	_, _, _, _ = s1, e1, s2, e2

//	// remove repeated results
//	w := []string{str1 + "=", "(" + str1 + ")="}
//	for _, s := range w {
//		if strings.HasPrefix(str2, s) {
//			is.Str = is.Str[:s2] + is.Str[s2+len(s):]
//			return
//		}
//	}

//	// also removes ":="
//	s := str1
//	if strings.HasPrefix(str2, s) {
//		is.Str = is.Str[:e1] + is.Str[s2+len(s):]
//		return
//	}
//}
