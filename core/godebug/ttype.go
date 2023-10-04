package godebug

import (
	"go/ast"
	"go/constant"
	"go/types"
)

// NOTE:
// types.Type -> Basic,Named,Array,Slice,Struct,Tuple,Signature,...
// types.Object -> Var,Func,Const,TypeName,Builtin,...
//
// ast.Expr -> types.Info.{objectOf,selections,...} -> types.Object
// ast.Expr -> types.Info.Types -> types.Type
// types.Type.(*types.Named) -> types.Object
// types.Object -> types.Type
// types.Object -> types.Package

// types type
type TType struct {
	node ast.Node

	tv  *types.TypeAndValue
	obj types.Object

	Type types.Type // cache, check typeTypes()
}

func newTType(node ast.Node, info *types.Info) (*TType, bool) {
	tt := &TType{node: node}
	if !tt.init(node, info) {
		return nil, false
	}
	return tt, true
}
func (tt *TType) init(node ast.Node, info *types.Info) bool {
	// special case
	switch t := node.(type) {
	case *ast.FuncDecl:
		node = t.Name
	}

	setObj := func(obj types.Object) {
		if tt.obj == nil {
			tt.obj = obj
			if tt.Type == nil {
				tt.Type = obj.Type()
			}
		}
	}
	setType := func(t types.Type) {
		if tt.Type == nil {
			tt.Type = t
			if tt.obj == nil {
				if tn, ok := t.(*types.Named); ok {
					tt.obj = tn.Obj()
				}
			}
		}
	}

	// type and value
	if expr, ok := node.(ast.Expr); ok {
		if tv, ok := info.Types[expr]; ok {
			tt.tv = &tv
			setType(tt.tv.Type)
		}
	}

	// type/object
	switch t := node.(type) {
	case *ast.Ident:
		if obj := info.ObjectOf(t); obj != nil {
			setObj(obj)
		}
		if inst, ok := info.Instances[t]; ok {
			setType(inst.Type)
		}
	case *ast.SelectorExpr:
		if sel, ok := info.Selections[t]; ok && sel.Obj() != nil {
			setObj(sel.Obj())
		}
	default:
		if obj, ok := info.Implicits[node]; ok {
			setObj(obj)
		}
	}

	return tt.Type != nil
}

//----------

func (tt *TType) constValue() (constant.Value, bool) {
	if tt.tv != nil {
		return tt.tv.Value, tt.tv.Value != nil
	}
	return nil, false
}

//----------

func (tt *TType) objPackage() (*types.Package, bool) {
	if tt.obj == nil {
		return nil, false
	}
	pkg := tt.obj.Pkg()
	if pkg == nil {
		return nil, false
	}
	return pkg, true
}

//----------

func (tt *TType) isType() bool {
	if tt.tv != nil {
		return tt.tv.IsType()
	}
	if tt.obj != nil {
		_, ok := tt.obj.(*types.TypeName)
		return ok // TODO: review
	}
	return false
}
func (tt *TType) isNil() bool {
	if tt.tv != nil {
		return tt.tv.IsNil()
	}
	//if tt.obj != nil {
	//}
	//return tt.Type == types.Typ[types.UntypedNil] // TODO
	return false
}
func (tt *TType) isBuiltin() bool {
	if tt.tv != nil {
		return tt.tv.IsBuiltin()
	}
	if tt.obj != nil {
		_, ok := tt.obj.(*types.Builtin) // TODO: review
		return ok
	}
	return false
}
func (tt *TType) isBuiltinWithName(name string) bool {
	if !tt.isBuiltin() {
		return false
	}
	id, ok := tt.node.(*ast.Ident)
	return ok && id.Name == name
}
func (tt *TType) isBasic() bool {
	_, ok := tt.Type.(*types.Basic)
	return ok
}
func (tt *TType) isBasicInfo(bi types.BasicInfo) bool { // ex: types.IsBoolean
	tb, ok := tt.Type.(*types.Basic)
	return ok && tb.Info()&bi != 0
}

func (tt *TType) isSignatureVariadic() bool {
	sig, ok := tt.Type.(*types.Signature)
	return ok && sig.Variadic()
}

//----------

func (tt *TType) nResults() int {
	return tt.nResults2(false)
}
func (tt *TType) nResults2(retInFuncLit bool) int {
	w := tt.typeTypes(retInFuncLit)
	return len(w)
}
func (tt *TType) typeTypes(retInFuncLit bool) []types.Type {
	switch t := tt.Type.(type) {
	case *types.Tuple: // ex: a,ok=map[b]; 1,2=f()
		return tupleTypes(t)
	case *types.Signature:
		// special case
		switch tt.node.(type) {
		case *ast.FuncDecl:
			return tupleTypes(t.Results())
		}
		// special case:
		// there is a difference between a funclit as a value (1 result) and a returnstmt inside a funclit that needs to know the number of results of the function itself
		if _, ok := tt.node.(*ast.FuncLit); ok && retInFuncLit {
			return tupleTypes(t.Results())
		}
	}
	return []types.Type{tt.Type}
}

//----------
//----------
//----------

func tupleTypes(tu *types.Tuple) []types.Type {
	w := []types.Type{}
	for i := 0; i < tu.Len(); i++ {
		w = append(w, tu.At(i).Type())
	}
	return w
}
