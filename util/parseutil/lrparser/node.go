package lrparser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/util/parseutil/pscan"
)

type PState struct {
	Sc   *pscan.Scanner
	Pos  int
	Node PNode
}

func NewPState(src []byte) *PState {
	ps := &PState{}
	ps.Sc = pscan.NewScanner()
	ps.Sc.SetSrc(src)
	return ps
}

//func (ps *PState) NodeBytes(node PNode) []byte {
//	pos, end := node.Pos(), node.End()
//	if pos > end {
//		pos, end = end, pos
//	}
//	return ps.Sc.SrcFromTo(pos, end)
//}
//func (ps *PState) NodeString(node PNode) string {
//	return string(ps.NodeBytes(node))
//}

//----------
//----------
//----------

type RuneRange = pscan.RuneRange

//type RuneRanges = parseutil.RuneRanges
//type PosError = parseutil.ScError

type PosError struct {
	Err error
	Pos int
	//Callstack string
	Fatal bool
}

func (e *PosError) Error() string {
	return e.Err.Error()
	//cs := ""
	//if e.Callstack != "" {
	//	cs = "\n" + e.Callstack
	//}
	//return fmt.Sprintf("%s%s", e.Err, cs)
}

//----------
//----------
//----------

// parse node
type PNode interface {
	Pos() int
	End() int
}

//----------

func PNodeBytes(node PNode, src []byte) []byte {
	pos, end := node.Pos(), node.End()
	if pos > end {
		pos, end = end, pos
	}
	return src[pos:end]
}
func PNodeString(node PNode, src []byte) string {
	return string(PNodeBytes(node, src))
}
func PNodePosStr(node PNode) string {
	return fmt.Sprintf("[%v:%v]", node.Pos(), node.End())
}

//func pnodeSrc2(node PNode, fset *FileSet) string {
//	return string(PNodeBytes(node, fset.Src))
//}

//----------
//----------
//----------

// basic parse node implementation
type BasicPNode struct {
	pos int // can have pos>end when in reverse
	end int
}

func (n *BasicPNode) Pos() int {
	return n.pos
}
func (n *BasicPNode) End() int {
	return n.end
}
func (n *BasicPNode) SetPos(pos, end int) {
	n.pos = pos
	n.end = end
}
func (n *BasicPNode) PosEmpty() bool {
	return n.pos == n.end
}
func (n *BasicPNode) SrcString(src []byte) string {
	return string(src[n.pos:n.end])
}

//----------
//----------
//----------

// content parser node
type CPNode struct {
	BasicPNode
	rule      Rule // can be nil in state0
	childs    []*CPNode
	data      interface{}
	simulated bool
}

func newCPNode(pos, end int, r Rule) *CPNode {
	cpn := &CPNode{rule: r}
	cpn.SetPos(pos, end)
	return cpn
}
func newCPNode2(n1, n2 PNode, r Rule) *CPNode {
	return newCPNode(n1.Pos(), n2.End(), r)
}

//----------

func (cpn *CPNode) addChilds(reverse bool, cs ...*CPNode) {
	if reverse {
		// wARNING: changes slice order
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
	cpr *cpRun
	cpn *CPNode
}

func newBuildNodeData(cpr *cpRun, cpn *CPNode) *BuildNodeData {
	return &BuildNodeData{cpr: cpr, cpn: cpn}
}
func (d *BuildNodeData) Pos() int {
	return d.cpn.Pos()
}
func (d *BuildNodeData) End() int {
	return d.cpn.End()
}
func (d *BuildNodeData) NodeSrc() string {
	return PNodeString(d.cpn, d.cpr.ps.Sc.SrcFullFromOffset())
}
func (d *BuildNodeData) FullSrc() []byte {
	return d.cpr.ps.Sc.SrcFullFromOffset()
}
func (d *BuildNodeData) Data() interface{} {
	return d.cpn.data
}
func (d *BuildNodeData) SetData(v interface{}) {
	d.cpn.data = v
}
func (d *BuildNodeData) IsEmpty() bool {
	return d.cpn.PosEmpty()
}
func (d *BuildNodeData) ExternalData() any {
	return d.cpr.externalData
}

//----------

func (d *BuildNodeData) SprintRuleTree(maxDepth int) string {
	return SprintNodeTree(d.cpr.ps.Sc.SrcFullFromOffset(), d.cpn, maxDepth)
}
func (d *BuildNodeData) PrintRuleTree(maxDepth int) {
	fmt.Printf("%v\n", d.SprintRuleTree(maxDepth))
}

// TODO: func to print cpnode.data if present

//----------

func (d *BuildNodeData) ChildsLen() int {
	return len(d.cpn.childs)
}
func (d *BuildNodeData) Child(i int) *BuildNodeData {
	return newBuildNodeData(d.cpr, d.cpn.childs[i])
}
func (d *BuildNodeData) ChildStr(i int) string {
	return PNodeString(d.cpn.childs[i], d.cpr.ps.Sc.SrcFullFromOffset())
}
func (d *BuildNodeData) ChildInt(i int) (int, error) {
	s := d.ChildStr(i)
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}
func (d *BuildNodeData) ChildOptional(i int) (*BuildNodeData, bool) {
	d2 := d.Child(i)
	if d2.IsEmpty() {
		return nil, false
	}
	return d2, true
}

func (d *BuildNodeData) IsOr(i int) bool {
	or, ok := d.cpn.rule.(*OrRule)
	if !ok {
		return false
	}
	target := or.childs()[i]

	switch t := target.(type) {
	case *DefRule:
		if len(d.cpn.childs) == 1 {
			return d.cpn.childs[0].rule == t
		}
	default:
		if len(d.cpn.childs) == len(target.childs()) {
			for k, cpn := range d.cpn.childs {
				if cpn.rule != target.childs()[k] {
					return false
				}
			}
			return true
		}
	}
	return false
}

//----------

func (d *BuildNodeData) ChildLoop(i int, fn BuildNodeFn) error {
	d2 := d.Child(i)
	//d2.PrintRuleTree(5)
	if d2.IsEmpty() {
		return nil
	}

	dr, ok := d2.cpn.rule.(*DefRule)
	if !ok {
		return fmt.Errorf("not a defrule")
	}

	vis := (BuildNodeFn)(nil)
	vis = func(d3 *BuildNodeData) error {
		if d3.IsEmpty() {
			return nil
		}
		if err := vis(d3.Child(0)); err != nil { // loop child
			return err
		}
		return fn(d3.Child(1)) // rule child
	}

	if dr.isPZeroOrMore {
		return vis(d2)
	}
	if dr.isPOneOrMore {
		if err := vis(d2.Child(0)); err != nil {
			return err
		}
		return fn(d2.Child(1)) // rule child (last)
	}

	return fmt.Errorf("child not a loop (missing loop option)")
}

func (d *BuildNodeData) ChildLoop2(i int, loopi int, pre, post BuildNodeFn) error {
	d2 := d.Child(i)
	//d2.PrintRuleTree(5)
	if d2.IsEmpty() {
		return nil
	}

	vis := (BuildNodeFn)(nil)
	vis = func(d3 *BuildNodeData) error {
		if d3.IsEmpty() {
			return nil
		}

		// could be a production with less childs
		l := d3.ChildsLen()
		if loopi >= l {
			return nil
		}

		// rule
		if pre != nil {
			if err := pre(d3); err != nil {
				return err
			}
		}
		// loop
		if err := vis(d3.Child(loopi)); err != nil {
			return err
		}
		// rule
		if post != nil {
			if err := post(d3); err != nil {
				return err
			}
		}
		return nil
	}
	return vis(d2)
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

		pr(depth, "-> %v: %q\n", tag, PNodeString(n, src))

		if cpn != nil {

			visitChilds := true
			if dr, ok := cpn.rule.(*DefRule); ok {
				if dr.isNoPrint {
					visitChilds = false
				}
			}

			if visitChilds {
				for _, child := range cpn.childs {
					vis(child, depth+1)
				}
			}
		}
	}
	vis(node, 0)
	return strings.TrimSpace(sb.String())
}
