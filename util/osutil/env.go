package osutil

import (
	"fmt"
	"strconv"
	"strings"
)

//godebug:annotatefile

func GetEnv(env []string, key string) string {
	for _, s := range env {
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

func UnquoteEnvValues(env []string) []string {
	for i, s := range env {
		k, v, ok := splitEnvVar(s)
		if ok {
			if v2, err := strconv.Unquote(v); err == nil {
				env[i] = keyvalStr(k, v2)
			}
		}
	}
	return env
}

//----------

func SetEnv(env []string, key, value string) []string {
	entry := keyvalStr(key, value)
	set := false
	for i, s := range env {
		k, _, ok := splitEnvVar(s)
		if !ok {
			continue
		}
		if k == key {
			if set {
				env[i] = "" // clear entry
			} else {
				env[i] = entry
				set = true // clear following entries
			}
		}
	}

	// clear empty entries
	env2 := []string{}
	for _, s := range env {
		if s != "" {
			env2 = append(env2, s)
		}
	}
	env = env2

	if !set {
		return append(env, entry)
	}
	return env
}

func keyvalStr(key, value string) string {
	return fmt.Sprintf("%v=%v", key, value)
}

func SetEnvs(env []string, addEnv []string) []string {
	for _, s := range addEnv {
		k, v, ok := splitEnvVar(s)
		if !ok {
			continue
		}
		env = SetEnv(env, k, v)
	}
	return env
}

func splitEnvVar(s string) (string, string, bool) {
	u := strings.SplitN(s, "=", 2)
	if len(u) != 2 {
		return "", "", false
	}
	return u[0], u[1], true
}
