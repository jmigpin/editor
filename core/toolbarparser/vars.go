package toolbarparser

//godebug:annotatefile

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

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
//----------
//----------

type VarMap map[string]string // [name]value; name includes {"~","$",...}

//----------
//----------
//----------

func ParseVars(data *Data) VarMap {
	vm := VarMap{}
	for _, part := range data.Parts {
		if len(part.Args) != 1 { // must have 1 arg
			continue
		}
		str := part.Args[0].String() // parse first arg only
		v, err := parseVarDecl(str)
		if err != nil {
			continue
		}

		// allows to reuse previously value of the same variable
		s := expandVariables(v.Value, vm)

		if u, err := parseutil.UnquoteString(s, '\\'); err == nil {
			s = u
		}

		vm[v.Name] = s
	}
	return vm
}
func expandVariables(src string, vm VarMap) string {
	// commented: alternative
	//v.Value = os.Expand(v.Value, func(name string) string {
	//	name = "$" + name
	//	return vm[name]
	//})

	// replaces "$" and also "~" vars
	return expandVarRefs(src, func(name string) (string, bool) {
		v, ok := vm[name]
		return v, ok
	})
}
