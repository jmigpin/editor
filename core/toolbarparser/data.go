package toolbarparser

import (
	"fmt"

	"github.com/jmigpin/editor/util/parseutil"
	"github.com/jmigpin/editor/util/parseutil/lrparser"
)

type Data struct {
	Str   string // parsed source
	Parts []*Part
}

func (d *Data) PartAtIndex(i int) (*Part, bool) {
	for _, p := range d.Parts {
		if i >= p.Pos() && i <= p.End() { // end includes separator and eos
			return p, true
		}
	}
	return nil, false
}
func (d *Data) Part0Arg0() (*Arg, bool) {
	if len(d.Parts) > 0 && len(d.Parts[0].Args) > 0 {
		return d.Parts[0].Args[0], true
	}
	return nil, false
}

//----------

func (d *Data) String() string {
	s := ""
	for i, p := range d.Parts {
		s += fmt.Sprintf("part%v:\n", i)
		for j, arg := range p.Args {
			s += fmt.Sprintf("\targ%v: %q\n", j, arg)
		}
		for j, v := range p.Vars {
			s += fmt.Sprintf("\tvar%v: %q\n", j, v)
		}
	}
	return s
}

//----------
//----------
//----------

type Part struct {
	Node
	Args []*Arg
	Vars []*VarDecl
}

func (p *Part) ArgsUnquoted() []string {
	args := []string{}
	for _, a := range p.Args {
		args = append(args, a.UnquotedString())
	}
	return args
}

func (p *Part) ArgsStrings() []string {
	args := []string{}
	for _, a := range p.Args {
		args = append(args, a.String())
	}
	return args
}

func (p *Part) FromArgString(i int) string {
	if i >= len(p.Args) {
		return ""
	}
	a := p.Args[i:]
	n1 := a[0]
	n2 := a[len(a)-1]
	return p.Data.Str[n1.Pos():n2.End()]
}

//----------
//----------
//----------

type Arg struct {
	Node
}

//----------
//----------
//----------

type Node struct {
	lrparser.BasicPNode
	Data *Data
}

func (n *Node) String() string {
	return n.SrcString([]byte(n.Data.Str))
}
func (n *Node) UnquotedString() string {
	s := n.String()
	if s2, err := parseutil.UnquoteStringBs(s); err == nil {
		s = s2
	}
	return s
}

//----------
//----------
//----------

type VarDecl struct {
	Name, Value string
}

func (v *VarDecl) String() string {
	return fmt.Sprintf("%v=%v", v.Name, v.Value)
}

//----------
//----------
//----------

type VarRef struct {
	lrparser.BasicPNode
	Name string
}

func (v *VarRef) String() string {
	return v.Name
}
