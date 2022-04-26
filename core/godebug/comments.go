package godebug

import (
	"go/ast"
	"go/token"
	"sort"
)

// https://github.com/golang/go/issues/20744

// Comment nodes mapped to the following relevant node. Used for godebug directives.
func commentsWithNodes(fset *token.FileSet, topNode ast.Node, cgs []*ast.CommentGroup) (res []*CommentWithNode) {

	// ensure it is sorted (sanity check)
	sort.Slice(cgs, func(a, b int) bool {
		return cgs[a].Pos() < cgs[b].Pos()
	})

	k := -1
	next := func() *ast.CommentGroup {
		k++
		if k < len(cgs) {
			return cgs[k]
		}
		return nil
	}
	cur := next() // current
	ast.Inspect(topNode, func(n ast.Node) bool {
		if cur == nil {
			return false
		}
		n2 := (ast.Node)(nil)
		switch n.(type) {
		case ast.Stmt, ast.Decl, ast.Spec: // relevant node
			n2 = n
		}
		if n2 == nil {
			return true
		}
		// catch first node after the comment
		for n2.Pos() > cur.Pos() {
			for _, c := range cur.List {
				u := &CommentWithNode{c, n2}
				res = append(res, u)
			}
			cur = next()
			if cur == nil {
				break
			}
		}
		return true
	})
	return res
}

type CommentWithNode struct {
	Comment *ast.Comment
	Node    ast.Node
}

//----------

//func annOptNodesMap2(fset *token.FileSet, astFile *ast.File, opts []*AnnotationOpt) map[*AnnotationOpt]ast.Node {
//	m := map[*AnnotationOpt]ast.Node{}
//	if len(opts) == 0 {
//		return m
//	}

//	sort.Slice(opts, func(a, b int) bool {
//		return opts[a].Comment.Pos() < opts[b].Comment.Pos()
//	})

//	k := -1
//	nextOpt := func() *AnnotationOpt {
//		k++
//		if k < len(opts) {
//			return opts[k]
//		}
//		return nil
//	}
//	cur := nextOpt() // current
//	ast.Inspect(astFile, func(n ast.Node) bool {
//		// catch first node after the comment

//		if cur == nil {
//			return false
//		}
//		n2 := (ast.Node)(nil)
//		switch n.(type) {
//		case ast.Stmt, ast.Decl, ast.Spec:
//			n2 = n
//		}
//		if n2 != nil {
//			if n2.Pos() > cur.Comment.Pos() {
//				m[cur] = n2
//				cur.Node = n2
//				cur = nextOpt()
//			}
//		}
//		return true
//	})

//	// fill rest of nodes with astfile
//	for ; cur != nil; cur = nextOpt() {
//		m[cur] = astFile
//		cur.Node = astFile
//	}

//	return m
//}

//----------

//// commented: attaches to previous comments
//func annOptNodesMap1(fset *token.FileSet, astFile *ast.File, opts []*AnnotationOpt) map[*AnnotationOpt]ast.Node {
//	// wrap comments in commentgroups to use ast.NewCommentMap
//	cgs := []*ast.CommentGroup{}
//	cmap := map[*ast.CommentGroup]*AnnotationOpt{}
//	for _, opt := range opts {
//		cg := &ast.CommentGroup{List: []*ast.Comment{opt.Comment}}
//		cgs = append(cgs, cg)
//		cmap[cg] = opt
//	}
//	// map annotations to nodes
//	nmap := ast.NewCommentMap(fset, astFile, cgs)
//	optm := map[*AnnotationOpt]ast.Node{}
//	for n, cgs := range nmap {
//		cg := cgs[0]
//		opt, ok := cmap[cg]
//		if ok {
//			opt.Node = n
//			optm[opt] = n
//		}
//	}
//	// annotations that have no node, will have astFile as node
//	for opt := range optm {
//		if opt.Node == nil {
//			opt.Node = astFile
//			optm[opt] = astFile
//		}
//	}
//	return optm
//}
