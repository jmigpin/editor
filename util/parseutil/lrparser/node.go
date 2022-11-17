package lrparser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/util/parseutil"
)

//----------

type PState = parseutil.PState
type PNode = parseutil.PNode

func pnodeSrc2(node PNode, fset *FileSet) string {
	return string(parseutil.PNodeBytes(node, fset.Src))
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
	return parseutil.PNodeString(d.cpn, d.cpr.ps.Src)
}
func (d *BuildNodeData) FullSrc() []byte {
	return d.cpr.ps.Src
}
func (d *BuildNodeData) Data() interface{} {
	return d.cpn.data
}
func (d *BuildNodeData) SetData(v interface{}) {
	d.cpn.data = v
}
func (d *BuildNodeData) IsEmpty() bool {
	return d.cpn.pos == d.cpn.end
}
func (d *BuildNodeData) ExternalData() any {
	return d.cpr.externalData
}

//----------

func (d *BuildNodeData) SprintRuleTree(maxDepth int) string {
	return SprintNodeTree(d.cpr.ps.Src, d.cpn, maxDepth)
}
func (d *BuildNodeData) PrintRuleTree(maxDepth int) {
	fmt.Printf("%v\n", d.SprintRuleTree(maxDepth))
}

//----------

func (d *BuildNodeData) ChildsLen() int {
	return len(d.cpn.childs)
}
func (d *BuildNodeData) Child(i int) *BuildNodeData {
	return newBuildNodeData(d.cpr, d.cpn.childs[i])
}

func (d *BuildNodeData) ChildStr(i int) string {
	return parseutil.PNodeString(d.cpn.childs[i], d.cpr.ps.Src)
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

//func (d *BuildNodeData) ChildRecursive(i int, loopIndex, ruleIndex int, fn func(*BuildNodeData)) {
//	d2 := d.Child(i)
//	if d2.IsEmpty() {
//		return
//	}
//	//d2.PrintRuleTree(5)
//	vis := (func(*BuildNodeData))(nil)
//	vis = func(d3 *BuildNodeData) {
//		if d3.IsEmpty() {
//			return
//		}
//		l := d3.ChildsLen()
//		if loopIndex >= l || ruleIndex >= l {
//			return
//		}
//		if loopIndex < ruleIndex {
//			vis(d3.Child(loopIndex)) // loop child
//			fn(d3.Child(ruleIndex))  // rule child
//		} else {
//			fn(d3.Child(ruleIndex))  // rule child
//			vis(d3.Child(loopIndex)) // loop child
//		}
//	}
//	vis(d2)
//}

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

		pr(depth, "-> %v: %q\n", tag, parseutil.PNodeString(n, src))

		if cpn != nil {
			for _, child := range cpn.childs {
				vis(child, depth+1)
			}
		}
	}
	vis(node, 0)
	return strings.TrimSpace(sb.String())
}
