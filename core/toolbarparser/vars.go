package toolbarparser

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/statemach"
)

//----------

func ParseVars(data *Data) VarMap {
	m := VarMap{}
	for _, part := range data.Parts {
		if len(part.Args) == 0 {
			continue
		}
		str := part.Args[0].Str()
		v, err := ParseVar(str)
		if err != nil {
			continue
		}
		m[v.Name] = v.Value
	}
	return m
}

//----------

type Var struct {
	Name, Value string
}

func ParseVar(str string) (*Var, error) {
	sm := statemach.NewString(str)
	ru := sm.Peek()
	switch ru {
	case '~':
		return parseVar1(sm)
	case '$':
		return parseVar2(sm)
	}
	return nil, fmt.Errorf("unexpected rune: %v", ru)
}

//----------

func parseVar1(sm *statemach.String) (*Var, error) {
	// name
	if !sm.AcceptAny("~") {
		return nil, errors.New("expecting ~")
	}
	if !sm.AcceptInt() {
		return nil, errors.New("expecting int")
	}
	name := sm.Value()
	sm.Advance()

	// assign
	if !sm.AcceptAny("=") {
		return nil, errors.New("expecting =")
	}
	sm.Advance()

	// value
	var value string
	if sm.AcceptQuoteLoop(parseutil.QuoteRunes, osutil.EscapeRunes) {
		v := sm.Value()
		sm.Advance()
		s, err := strconv.Unquote(v)
		if err != nil {
			return nil, err
		}
		value = s
	} else {
		u, ok := parseutil.AcceptAdvanceFilename(sm)
		if !ok {
			return nil, errors.New("unable to get value")
		}
		value = u
	}

	v := &Var{Name: name, Value: value}
	return v, nil
}

//----------

func parseVar2(sm *statemach.String) (*Var, error) {
	// name
	if !sm.AcceptAny("$") {
		return nil, errors.New("expecting $")
	}
	if !sm.AcceptId() {
		return nil, errors.New("expecting id")
	}
	name := sm.Value()
	sm.Advance()

	// assign
	if !sm.AcceptAny("=") {
		return nil, errors.New("expecting =")
	}
	sm.Advance()

	// value
	var value string
	if sm.AcceptQuoteLoop(parseutil.QuoteRunes, osutil.EscapeRunes) {
		v := sm.Value()
		sm.Advance()
		s, err := strconv.Unquote(v)
		if err != nil {
			return nil, err
		}
		value = s
	} else {
		u, ok := parseutil.AcceptAdvanceFilename(sm)
		if !ok {
			return nil, errors.New("unable to get value")
		}
		value = u
	}

	v := &Var{Name: name, Value: value}
	return v, nil
}

//----------

type VarMap map[string]string // name -> value

func EncodeVars(filename string, m VarMap) string {
	return parseutil.EscapeFilename(encodeVars(filename, m))
}
func encodeVars(f string, m VarMap) string {
	best := ""
	for k, v := range m {
		v2 := DecodeVars(v, m)

		// exact match
		if f == v2 {
			return k
		}

		// (var + separator) prefix match (best is shortest len)
		v3 := v2 + string(filepath.Separator)
		if strings.HasPrefix(f, v3) {
			s := filepath.Join(k, f[len(v2):])
			if best == "" || len(s) < len(best) {
				best = s
			}
		}
	}
	if best != "" {
		return best
	}
	return f
}

//----------

func DecodeVars(f string, m VarMap) string {
	return parseutil.UnescapeString(decodeVars(f, m))
}
func decodeVars(f string, m VarMap) string {
	f = filepath.Clean(f)

	// split on first separator
	i := strings.IndexFunc(f, func(ru rune) bool {
		return ru == filepath.Separator
	})
	s0, s1 := f, ""
	if i > 0 {
		s0, s1 = f[:i], f[i:]
	}

	v, ok := m[s0]
	if ok {
		v2 := DecodeVars(v, m)
		return filepath.Join(append([]string{v2}, s1)...)
	}

	return f
}
