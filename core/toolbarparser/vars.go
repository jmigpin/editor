package toolbarparser

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

func (m *HomeVarMap) EncodeShortest(filename string) string {
	candidates := append([]string{m.Encode(filename)}, m.encodeNear(filename)...)
	sort.Strings(candidates)
	return shortestString(candidates)
}

func (m *HomeVarMap) encodeNear(filename string) []string {
	filename = osutil.FilepathClean(filename)
	ff := m.caseFilter(filename)
	candidates := []string{}
	for _, key := range m.sortedKeys() {
		value := m.vm[key]
		if !filepath.IsAbs(value) {
			continue
		}
		parent := value
		for depth := 0; ; depth++ {
			if depth > 0 && pathHasPrefix(ff, m.caseFilter(parent)) {
				if candidate, ok := encodeNearCandidate(key, value, parent, filename); ok {
					candidates = append(candidates, candidate)
				}
			}
			next := filepath.Dir(parent)
			if next == parent {
				break
			}
			parent = next
		}
	}
	return candidates
}

//----------

func (m *HomeVarMap) Decode(f string) string {
	// input can be from varmap (user input)
	f = parseutil.RemoveEscapes(f, osutil.EscapeRune)
	f = m.decode2(f)
	return osutil.FilepathClean(f)
}
func (m *HomeVarMap) DecodeVars(vm VarMap) {
	for k, v := range vm {
		if strings.HasPrefix(v, "~") {
			vm[k] = m.Decode(v)
		}
	}
}
func (m *HomeVarMap) ParseAndDecodeVars(data *Data) VarMap {
	return parseVars(data, m.Decode)
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

func (m *HomeVarMap) sortedKeys() []string {
	keys := []string{}
	for k := range m.vm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func pathHasPrefix(filename, prefix string) bool {
	if filename == prefix {
		return true
	}
	prefix2 := prefix
	if !strings.HasSuffix(prefix2, string(filepath.Separator)) {
		prefix2 += string(filepath.Separator)
	}
	return strings.HasPrefix(filename, prefix2)
}

func encodeNearCandidate(key, value, parent, filename string) (string, bool) {
	upRel, err := filepath.Rel(value, parent)
	if err != nil {
		return "", false
	}
	downRel, err := filepath.Rel(parent, filename)
	if err != nil {
		return "", false
	}

	parts := []string{key}
	if upRel != "." {
		parts = append(parts, strings.Split(upRel, string(filepath.Separator))...)
	}
	if downRel != "." {
		parts = append(parts, parseutil.EscapeFilename(downRel))
	}
	return strings.Join(parts, string(filepath.Separator)), true
}

func shortestString(candidates []string) string {
	shortest := ""
	for _, s := range candidates {
		if shortest == "" || len(s) < len(shortest) {
			shortest = s
		}
	}
	return shortest
}

//----------
//----------
//----------

type VarMap map[string]string // [name]value; name includes {"~","$",...}

//----------
//----------
//----------

func ParseVars(data *Data) VarMap {
	return parseVars(data, nil)
}

func parseVars(data *Data, decodePath func(string) string) VarMap {
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

		s := v.Value
		if u, err := parseutil.UnquoteString(s, '\\'); err == nil {
			s = u
		}

		// allows to reuse previously value of the same variable
		s = expandVariables(s, vm)

		if decodePath != nil && strings.HasPrefix(s, "~") {
			s = decodePath(s)
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
func expandVarRefs(src string, mapping func(string) (string, bool)) string {
	refs, err := parseVarRefs([]byte(src))
	if err != nil {
		// DEBUG
		//log.Println(err)

		return src
	}
	adjust := 0
	for _, vr := range refs {
		v, ok := mapping(vr.Name)
		if !ok {
			continue
		}
		// replace: refs are expected to be in ascending order
		pos := vr.Pos() + adjust
		end := vr.End() + adjust
		src = src[0:pos] + v + src[end:]
		adjust += len(v) - (end - pos)
	}
	return src
}
