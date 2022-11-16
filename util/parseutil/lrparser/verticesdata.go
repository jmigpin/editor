package lrparser

import (
	"fmt"
	"strings"
)

// passes rules (ruleindex) to vertices data
type VerticesData struct {
	verts   []*Vertex
	rFirst  *RuleFirstT
	reverse bool
}

func newVerticesData(ri *RuleIndex, startRuleName string, reverse bool) (*VerticesData, error) {
	vd := &VerticesData{reverse: reverse}

	if err := ri.derefRules(); err != nil {
		return nil, err
	}

	vAutoId := 0
	addNewVertexAutoId := func() *Vertex {
		v := newVertex(VertexId(vAutoId))
		vAutoId++
		vd.verts = append(vd.verts, v)
		return v
	}

	dr0, err := ri.startRule(startRuleName)
	if err != nil {
		return nil, err
	}

	vd.rFirst = newRuleFirstT(ri, vd.reverse)

	rd0 := newRuleDot(startRule, dr0, vd.reverse)
	rdlas0 := RuleDotsLaSet{}
	rdlas0.setRule(*rd0, endRule)

	v0 := addNewVertexAutoId()
	v0.rdslasK = rdlas0
	v0.rdslasC = rdlasClosure(rdlas0, vd.rFirst)

	stk := []*Vertex{}
	stk = append(stk, v0)
	seenV := map[string]*Vertex{}
	for len(stk) > 0 {
		k := len(stk) - 1
		v1 := stk[k]  // top
		stk = stk[:k] // pop

		//println("***")
		//fmt.Printf("rdslasC %v\n", v1.rdslasC.ruleDots())
		rset := v1.rdslasC.ruleDots().dotRulesSet()
		//fmt.Printf("rset %v\n", rset)
		for _, x := range rset.sorted() { // stable, otherwise alters vertex creation order and will fail tests
			//fmt.Printf("***%v\n", x)

			rdslasK, rdslasC := rdlasGoto(v1.rdslasC, x, vd.rFirst)
			if len(rdslasK) == 0 {
				continue
			}

			j := rdslasK.String() // union of sets with the same core
			v2, ok := seenV[j]
			if !ok {
				v2 = addNewVertexAutoId()
				v2.rdslasK = rdslasK
				v2.rdslasC = rdslasC
				seenV[j] = v2
				stk = append(stk, v2)
			}
			v1.gotoVert[x] = v2

		}
	}

	return vd, nil
}

//----------

func (vd *VerticesData) String() string {
	sb := &strings.Builder{}
	for _, v := range vd.verts {
		fmt.Fprintf(sb, "%v\n", v)
	}
	return strings.TrimSpace(sb.String())
}

//----------
//----------
//----------

func rdlasGoto(rdlas RuleDotsLaSet, x Rule, rFirst *RuleFirstT) (RuleDotsLaSet, RuleDotsLaSet) {
	res := RuleDotsLaSet{}
	for rd, laSet := range rdlas {
		if r, ok := rd.dotRule(); ok && r == x {
			rd2, _ := rd.advanceDot()
			res.setRuleSet(*rd2, laSet)
		}
	}
	return res, rdlasClosure(res, rFirst)
}

//----------
//----------
//----------

func rdlasClosure(rdslas RuleDotsLaSet, rFirst *RuleFirstT) RuleDotsLaSet {
	res := RuleDotsLaSet{}

	type entry struct {
		rd  *RuleDot
		las RuleSet
	}

	stk := []*entry{}

	// provided rdslas (kernels) are part of the closure
	for rd, las := range rdslas {
		rd2 := rd
		stk = append(stk, &entry{&rd2, las})
		res.setRuleSet(rd, las)
	}

	for len(stk) > 0 {
		k := len(stk) - 1
		e := stk[k]   // top
		stk = stk[:k] // pop

		// [A->α.Bβ,a], B->γ, b in first(βa), add [B->.γ,b]
		// A = rdla.rd.prod

		B, ok := e.rd.dotRule()
		if !ok {
			continue
		}

		if B.isTerminal() {
			continue
		}
		BProds := ruleProductions(B)

		β := []Rule{}
		rd2, ok := e.rd.advanceDot()
		if ok {
			w := rd2.dotAndAfterRules()
			β = append(β, w...)
		}

		for a := range e.las {
			βa := append(β, a)
			firstβa := rFirst.sequenceFirst(βa)
			for _, γ := range BProds {
				rd3 := newRuleDot(B, γ, rFirst.reverse)

				las := RuleSet{}
				for b := range firstβa { // b is terminal
					if !res.hasRule(*rd3, b) {
						res.setRule(*rd3, b)
						las.set(b)
					}
				}
				// add to continue processing
				if len(las) > 0 {
					stk = append(stk, &entry{rd3, las})
				}
			}
		}
	}

	return res
}

//----------
//----------
//----------

type Vertex struct {
	id       VertexId
	rdslasK  RuleDotsLaSet    // kernels
	rdslasC  RuleDotsLaSet    // closure
	gotoVert map[Rule]*Vertex // goto vertex
}

func newVertex(id VertexId) *Vertex {
	v := &Vertex{id: id}
	v.gotoVert = map[Rule]*Vertex{}
	return v
}
func (v *Vertex) String() string {
	s := fmt.Sprintf("%v:\n", v.id)

	// print kernels/rdlas
	s += indentStr("\t", v.rdslasC.String())

	// print edges 1 (rule->vertex)
	w := []Rule{}
	for e := range v.gotoVert {
		w = append(w, e)
	}
	sortRules(w)
	u := []string{}
	for _, r := range w {
		v2 := v.gotoVert[r]
		u = append(u, fmt.Sprintf("%v->%v", r.id(), v2.id))
	}
	s += indentStr("\t", strings.Join(u, "\n"))

	//// print edges 2 (vertex<-rule)
	//m2 := map[*Vertex][]Rule{}
	//w := []*Vertex{}
	//for r, v := range v.edges {
	//	w = append(w, v)
	//	m2[v] = append(m2[v], r)
	//}
	//sort.Slice(w, func(a, b int) bool {
	//	return w[a].id < w[b].id
	//})
	//for _, v := range w {
	//	u := []string{}
	//	for _, r := range m2[v] {
	//		u = append(u, fmt.Sprintf("%v", r.id()))
	//	}
	//	s += fmt.Sprintf("\t%v<-%v\n", v.id, strings.Join(u, ";"))
	//}

	return strings.TrimSpace(s)
}

//----------
//----------
//----------

type VertexId int

func (vid VertexId) String() string {
	return fmt.Sprintf("vertex%v", int(vid))
}
