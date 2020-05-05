package toolbarparser

//godebug:annotatefile

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jmigpin/editor/v2/util/iout/iorw"
	"github.com/jmigpin/editor/v2/util/osutil"
	"github.com/jmigpin/editor/v2/util/parseutil"
	"github.com/jmigpin/editor/v2/util/scanutil"
)

type VarMap map[string]string // name -> value (more then one type: "~","$", ...)

//----------

type HomeVarMap struct {
	vm              VarMap
	caseInsensitive bool
}

func NewHomeVarMap(vm VarMap, caseInsensitive bool) *HomeVarMap {
	m := &HomeVarMap{caseInsensitive: caseInsensitive}
	m.filter(vm)
	return m
}

func (m *HomeVarMap) filter(vm VarMap) {
	// filter home var keys
	keys := []string{}
	for k := range vm {
		if strings.HasPrefix(k, "~") {
			keys = append(keys, k)
		}
	}
	// sort to have priority when two variables have the same value
	sort.Strings(keys)
	// filter
	m.vm = VarMap{}
	seen := map[string]bool{}
	for _, k := range keys {
		v := vm[k]

		k = m.caseFilter(k)
		v = m.caseFilter(v)

		// verify that decoding the value doesn't exist already (uses up to date m.vm)
		v = m.Decode(v)

		if seen[v] {
			continue // other key has this value already
		}
		seen[v] = true

		m.vm[k] = v // keep decoded var (performance)
	}
}

func (m *HomeVarMap) caseFilter(s string) string {
	if m.caseInsensitive {
		return strings.ToLower(s)
	}
	return s
}

//----------

func (m *HomeVarMap) Encode(filename string) string {
	filename = osutil.FilepathClean(filename)
	v := m.encode2(filename)
	return parseutil.EscapeFilename(v)
}
func (m *HomeVarMap) encode2(f string) string {
	ff := m.caseFilter(f)
	best := ""
	for k, v := range m.vm {
		//v = m.decode(v) // not needed if values kept decoded

		// exact match
		if f == v {
			return k
		}

		// (var + separator) prefix match (best is shortest len)
		v3 := v + string(filepath.Separator)
		if strings.HasPrefix(ff, v3) {
			s := filepath.Join(k, f[len(v):])
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

func (m *HomeVarMap) Decode(f string) string {
	// input can be from varmap (user input)
	f = parseutil.RemoveEscapes(f, osutil.EscapeRune)
	f = osutil.FilepathClean(f)
	return m.decode2(f)
}
func (m *HomeVarMap) decode2(f string) string {
	// split on first separator
	i := strings.IndexFunc(f, func(ru rune) bool {
		return ru == filepath.Separator
	})
	ff := m.caseFilter(f)
	s0, s1 := ff, ""
	if i > 0 {
		s0, s1 = ff[:i], f[i:]
	}

	v, ok := m.vm[s0]
	if ok {
		v2 := m.decode2(v)
		return filepath.Join(v2, s1)
	}

	return f
}

//----------

func ParseVars(data *Data) VarMap {
	m := VarMap{}
	for _, part := range data.Parts {
		if len(part.Args) != 1 { // must have 1 arg
			continue
		}
		str := part.Args[0].Str() // parse first arg only
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
	rd := iorw.NewStringReaderAt(str)
	sc := scanutil.NewScanner(rd)
	ru := sc.PeekRune()
	switch ru {
	case '~':
		return parseTildeVar(sc)
	case '$':
		return parseDollarVar(sc)
	}
	return nil, fmt.Errorf("unexpected rune: %v", ru)
}

//----------

func parseTildeVar(sc *scanutil.Scanner) (*Var, error) {
	// name
	if !sc.Match.Sequence("~") {
		return nil, sc.Errorf("name")
	}
	if !sc.Match.Int() {
		return nil, sc.Errorf("name")
	}
	name := sc.Value()
	sc.Advance()
	// assign (must have)
	if !sc.Match.Any("=") {
		return nil, sc.Errorf("assign")
	}
	sc.Advance()
	// value (must have)
	v, err := parseVarValue(sc, false)
	if err != nil {
		return nil, err
	}
	// end
	_ = sc.Match.Spaces()
	if !sc.Match.End() {
		return nil, sc.Errorf("not at end")
	}

	w := &Var{Name: name, Value: v}
	return w, nil
}

//----------

func parseDollarVar(sc *scanutil.Scanner) (*Var, error) {
	// name
	if !sc.Match.Sequence("$") {
		return nil, sc.Errorf("name")
	}
	if !sc.Match.Id() {
		return nil, sc.Errorf("name")
	}
	name := sc.Value()
	sc.Advance()

	w := &Var{Name: name}

	// assign (optional)
	if !sc.Match.Any("=") {
		return w, nil
	}
	sc.Advance()
	// value (optional)
	value, err := parseVarValue(sc, true)
	if err != nil {
		return nil, err
	}
	w.Value = value
	// end
	_ = sc.Match.Spaces()
	if !sc.Match.End() {
		return nil, sc.Errorf("not at end")
	}

	return w, nil
}

//----------

func parseVarValue(sc *scanutil.Scanner, allowEmpty bool) (string, error) {
	if sc.Match.Quoted("\"'", osutil.EscapeRune, true, 1000) {
		v := sc.Value()
		sc.Advance()
		u, err := strconv.Unquote(v)
		if err != nil {
			return "", sc.Errorf("unquote: %v", err)
		}
		return u, nil
	} else {
		if !sc.Match.ExceptUnescapedSpaces(osutil.EscapeRune) {
			if !allowEmpty {
				return "", sc.Errorf("value")
			}
		}
		v := sc.Value()
		sc.Advance()
		return v, nil
	}
}
