package lrparser

import (
	"fmt"
	"strconv"
	"strings"
)

// parse node
type PNode interface {
	Pos() int
	End() int
}

//----------

func pnodeSrc(node PNode, src []byte) string {
	pos, end := node.Pos(), node.End()
	if pos > end {
		pos, end = end, pos
	}
	return string(src[pos:end])
}
func pnodeSrc2(node PNode, fset *FileSet) string {
	return pnodeSrc(node, fset.Src)
}
func pnodePosStr(node PNode) string {
	return fmt.Sprintf("[%v:%v]", node.Pos(), node.End())
}

//----------
//----------
//----------

// common node
type CmnPNode struct {
	pos int // can have pos>end when in reverse
	end int
}

func (n *CmnPNode) setPos(pos, end int) {
	n.pos = pos
	n.end = end
}
func (n *CmnPNode) Pos() int {
	return n.pos
}
func (n *CmnPNode) End() int {
	return n.end
}

//----------
//----------
//----------

// content parser node
type CPNode struct {
	CmnPNode
	rule      Rule // can be nil in state0
	childs    []*CPNode
	data      interface{}
	simulated bool
}

func newCPNode(pos, end int, r Rule) *CPNode {
	cpn := &CPNode{rule: r}
	cpn.setPos(pos, end)
	return cpn
}
func newCPNode2(n1, n2 PNode, r Rule) *CPNode {
	return newCPNode(n1.Pos(), n2.End(), r)
}

//----------

func (cpn *CPNode) addChilds(reverse bool, cs ...*CPNode) {
	if reverse {
		for i := 0; i < len(cs)/2; i++ {
			k := len(cs) - 1 - i
			cs[i], cs[k] = cs[k], cs[i]
		}
		cpn.childs = append(cs, cpn.childs...)
	} else {
		cpn.childs = append(cpn.childs, cs...)
	}
}

//----------
//----------
//----------

type BuildNodeFn func(*BuildNodeData) error

//----------

type BuildNodeData struct {
	cpn *CPNode
	ps  *PState
}

func (d *BuildNodeData) Pos() int {
	return d.cpn.Pos()
}
func (d *BuildNodeData) End() int {
	return d.cpn.End()
}

func (d *BuildNodeData) Src() string {
	return pnodeSrc(d.cpn, d.ps.src)
}
func (d *BuildNodeData) Data() interface{} {
	return d.cpn.data
}
func (d *BuildNodeData) SetData(v interface{}) {
	d.cpn.data = v
}
func (d *BuildNodeData) IsNil() bool {
	return d.cpn.pos == d.cpn.end // TODO: nil flag?
}

//----------

func (d *BuildNodeData) SprintRuleTree(maxDepth int) string {
	return SprintNodeTree(d.ps.src, d.cpn, maxDepth)
}
func (d *BuildNodeData) PrintRuleTree(maxDepth int) {
	fmt.Printf("%v\n", d.SprintRuleTree(maxDepth))
}

//----------

func (d *BuildNodeData) ChildsLen() int {
	return len(d.cpn.childs)
}
func (d *BuildNodeData) Child(i int) *BuildNodeData {
	return &BuildNodeData{cpn: d.cpn.childs[i], ps: d.ps}
}

func (d *BuildNodeData) ChildStr(i int) string {
	return pnodeSrc(d.cpn.childs[i], d.ps.src)
}
func (d *BuildNodeData) ChildInt(i int) (int, error) {
	s := d.ChildStr(i)
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

//----------
//----------
//----------

// maxdepth=-1 will print all
func SprintNodeTree(src []byte, node PNode, maxDepth int) string {
	sb := &strings.Builder{}

	pr := func(depth int, f string, args ...interface{}) {
		for i := 0; i < depth; i++ {
			fmt.Fprint(sb, "\t")
		}
		fmt.Fprintf(sb, f, args...)
	}

	vis := (func(PNode, int))(nil)
	vis = func(n PNode, depth int) {
		if maxDepth >= 0 && depth >= maxDepth {
			pr(depth, "-> ... (maxdepth=%v)\n", maxDepth)
			return
		}

		tag := ""

		cpn, ok := n.(*CPNode)
		if ok {
			tag = cpn.rule.id()
		} else {
			tag = fmt.Sprintf("%T", n)
		}

		pr(depth, "-> %v: %q\n", tag, pnodeSrc(n, src))

		if cpn != nil {
			for _, child := range cpn.childs {
				vis(child, depth+1)
			}
		}
	}
	vis(node, 0)
	return strings.TrimSpace(sb.String())
}
