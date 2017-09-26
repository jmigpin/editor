package toolbardata

import "strings"

// Replaces prefixes by defined vars.
type HomeVars struct {
	vars []*HVEntry
}

type HVEntry struct {
	k, v string
}

func (h *HomeVars) Append(k, v string) {
	// don't use empty entries
	if k == "" || v == "" {
		return
	}

	h.vars = append(h.vars, &HVEntry{k, v})
}
func (h *HomeVars) Delete(k string) {
	var u []*HVEntry
	for _, e := range h.vars {
		if e.k != k {
			u = append(u, e)
		}
	}
	h.vars = u
}

func (h *HomeVars) Encode(s string) string {
	for _, e := range h.vars {
		if strings.HasPrefix(s, e.v) {
			s = e.k + s[len(e.v):]
		}
	}
	return s
}
func (h *HomeVars) Decode(s string) string {
	for i := len(h.vars) - 1; i >= 0; i-- {
		e := h.vars[i]
		if strings.HasPrefix(s, e.k) {
			s = e.v + s[len(e.k):]
		}
	}
	return s
}
