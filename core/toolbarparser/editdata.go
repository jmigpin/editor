package toolbarparser

import (
	"fmt"

	"github.com/jmigpin/editor/util/parseutil/pscan"
)

// scan for cmd position, update with arg, or insert new cmd
func UpdateOrInsertPartCmd(data *Data, cmd, arg string) uoipcResult {
	part, ok := findLastCmdPart(data, cmd)
	if ok {
		return updateCmdPartWithArg(part, arg)
	}
	return insertCmdPartAtEnd(data, cmd, arg)
}

//----------

func findLastCmdPart(data *Data, cmd string) (*Part, bool) {
	found := (*Part)(nil)
	for _, p := range data.Parts {
		if len(p.Args) > 0 && p.Args[0].String() == cmd {
			found = p
		}
	}
	if found != nil {
		return found, true
	}
	return nil, false
}

//----------

func updateCmdPartWithArg(part *Part, arg string) uoipcResult {
	// asssume at least one arg (cmd arg)
	start, end := part.Args[0].End(), part.End()

	res := uoipcResult{Pos: start, End: end}
	res.S = part.Data.Str

	args := part.Args[1:]
	if len(args) == 0 {
		// need to insert starting space
		res.S, res.Pos = insDelStr(res.S, res.Pos, res.Pos, " ")
		res.End = res.Pos
	} else {
		// use args positions, don't disturb original disposition
		res.Pos = args[0].Pos()
		res.End = args[len(args)-1].End()
	}

	if arg != "" {
		res.S, res.End = insDelStr(res.S, res.Pos, res.End, arg)
	}

	return res
}

//----------

func insertCmdPartAtEnd(data *Data, cmd, arg string) uoipcResult {
	replaceAt := func(pos int, prefix string) uoipcResult {
		res := uoipcResult{}
		s1 := fmt.Sprintf("%s%s ", prefix, cmd)
		res.S, res.Pos = insDelStr(data.Str, pos, len(data.Str), s1)
		res.S, res.End = insDelStr(res.S, res.Pos, len(res.S), arg)
		return res
	}

	sc := pscan.NewScanner()
	sc.SetSrc([]byte(data.Str))
	sc.Reverse = true

	// backtrack spaces (best effort)
	p2 := len(data.Str)
	if p3, err := sc.M.Loop(p2, sc.W.RuneOneOf([]rune(" \t"))); err == nil {
		p2 = p3
	}

	if _, err := sc.M.Rune(p2, '\n'); err == nil {
		return replaceAt(p2, "")
	}
	if _, err := sc.M.Rune(p2, '|'); err == nil {
		return replaceAt(p2, " ")
	}
	return replaceAt(p2, " | ")
}

//----------

func insDelStr(s string, i1, i2 int, a string) (string, int) {
	u := s[:i1] + a + s[i2:]
	return u, i1 + len(a)
}

//----------
//----------
//----------

type uoipcResult struct {
	S        string
	Pos, End int
}
