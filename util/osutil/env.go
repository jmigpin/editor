package osutil

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jmigpin/editor/util/strconvutil"
)

func GetEnv(env []string, key string) string {
	for i := len(env) - 1; i >= 0; i-- { // last entry has precedence
		s := env[i]
		k, v, ok := splitEnvVar(s)
		if !ok {
			continue
		}
		if k == key {
			return v
		}
	}
	return ""
}

func SetEnv(env []string, key, value string) []string {
	w := append(env, keyValStr(key, value))
	w = clearDuplicatesFavorLast(w)
	return w
}
func SetEnv2(env *[]string, key, value string) {
	*env = SetEnv(*env, key, value)
}

func AppendEnv(env []string, addEnv []string) []string {
	w := append(env, addEnv...)
	w = clearDuplicatesFavorLast(w)
	return w
}

//----------

func UnquoteEnvValues(env []string) []string {
	w := []string{}
	for _, s := range env {
		k, v, ok := splitEnvVar(s)
		if !ok {
			continue
		}
		// NOTE: strconv.Unquote() fails on singlequotes with len>6 runes
		if v2, ok := strconvutil.BasicUnquote(v); ok {
			w = append(w, keyValStr(k, v2))
		} else {
			w = append(w, s)
		}
	}
	return w
}

//----------

func keyValStr(key, value string) string {
	return fmt.Sprintf("%v=%v", key, value)
}

func splitEnvVar(s string) (string, string, bool) {
	return strings.Cut(s, "=")
}

func clearDuplicatesFavorLast(env []string) []string {
	w := []string{}
	seen := map[string]bool{}
	for i := len(env) - 1; i >= 0; i-- { // bottom up, keep last
		s := env[i]
		k, _, ok := splitEnvVar(s)
		if !ok {
			continue
		}
		if seen[k] {
			continue
		}
		seen[k] = true
		w = append(w, s)
	}
	slices.Reverse(w)
	return w
}
