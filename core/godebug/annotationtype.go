package godebug

import (
	"fmt"
	"go/ast"
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

func annOptInComment(c *ast.Comment, n ast.Node) (*AnnotationOpt, bool, error) {
	typ, opt, err := AnnotationTypeInString(c.Text)
	if err != nil {
		return nil, false, err
	}
	if typ == AnnotationTypeNone {
		return nil, false, nil
	}
	u := &AnnotationOpt{Type: typ, Opt: opt, Comment: c, Node: n}
	return u, true, nil
}
