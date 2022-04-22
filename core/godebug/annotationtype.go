package godebug

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"
	"unicode"
)

type AnnotationType int

const (
	// Order matters, last is the bigger set
	AnnotationTypeNone AnnotationType = iota
	AnnotationTypeOff
	AnnotationTypeBlock
	AnnotationTypeFile
	AnnotationTypeImport  // annotates set of files (importspec)
	AnnotationTypePackage // annotates set of files
	AnnotationTypeModule  // annotates set of packages
)

func AnnotationTypeInString(s string) (AnnotationType, string, error) {
	prefix := "//godebug:"
	if !strings.HasPrefix(s, prefix) {
		return AnnotationTypeNone, "", nil
	}

	// type and optional rest of the string
	typ := s[len(prefix):]
	opt, hasOpt := "", false
	i := strings.Index(typ, ":")
	if i >= 0 {
		hasOpt = true
		typ, opt = typ[:i], typ[i+1:]
	} else {
		// allow some space at the end (ex: comments)
		i := strings.IndexFunc(typ, unicode.IsSpace)
		if i >= 0 {
			typ = typ[:i]
		}
	}
	typ = strings.TrimSpace(typ)
	opt = strings.TrimSpace(opt)

	var at AnnotationType
	switch typ {
	case "annotateoff":
		at = AnnotationTypeOff
	case "annotateblock":
		at = AnnotationTypeBlock
	case "annotatefile":
		at = AnnotationTypeFile
	case "annotatepackage":
		at = AnnotationTypePackage
	case "annotateimport":
		at = AnnotationTypeImport
	case "annotatemodule":
		at = AnnotationTypeModule
	default:
		err := fmt.Errorf("unexpected annotate type: %q", typ)
		return AnnotationTypeNone, "", err
	}

	// ensure early error if opt is set on annotations not expecting it
	if hasOpt {
		switch at {
		case AnnotationTypeFile:
		case AnnotationTypePackage:
		case AnnotationTypeModule:
		default:
			return at, opt, fmt.Errorf("unexpected annotate option: %q", opt)
		}
	}

	return at, opt, nil
}

//----------
//----------
//----------

type AnnotationOpt struct {
	Type    AnnotationType
	Opt     string
	Comment *ast.Comment // comment node
	Node    ast.Node     // node associated to comment (can be nil)
}

//----------
//----------
//----------

func annOptInComment(c *ast.Comment) (*AnnotationOpt, bool, error) {
	typ, opt, err := AnnotationTypeInString(c.Text)
	if err != nil {
		return nil, false, err
	}
	if typ == AnnotationTypeNone {
		return nil, false, nil
	}
	u := &AnnotationOpt{Type: typ, Opt: opt, Comment: c}
	return u, true, nil
}

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

func annOptNodesMap2(fset *token.FileSet, astFile *ast.File, opts []*AnnotationOpt) map[*AnnotationOpt]ast.Node {
	m := map[*AnnotationOpt]ast.Node{}
	if len(opts) == 0 {
		return m
	}

	sort.Slice(opts, func(a, b int) bool {
		return opts[a].Comment.Pos() < opts[b].Comment.Pos()
	})

	k := -1
	nextOpt := func() *AnnotationOpt {
		k++
		if k < len(opts) {
			return opts[k]
		}
		return nil
	}
	cur := nextOpt() // current
	ast.Inspect(astFile, func(n ast.Node) bool {
		// catch first node after the comment

		if cur == nil {
			return false
		}
		n2 := (ast.Node)(nil)
		switch n.(type) {
		case ast.Stmt, ast.Decl, ast.Spec:
			n2 = n
		}
		if n2 != nil {
			if n2.Pos() > cur.Comment.Pos() {
				m[cur] = n2
				cur.Node = n2
				cur = nextOpt()
			}
		}
		return true
	})

	// fill rest of nodes with astfile
	for ; cur != nil; cur = nextOpt() {
		m[cur] = astFile
		cur.Node = astFile
	}

	return m
}
