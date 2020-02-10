package osutil

import (
	"os"
	"path/filepath"
	"strings"
)

func HomeEnvVar() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

//----------

func FilepathHasDirPrefix(s, prefix string) bool {
	// ensure it ends in separator
	sep := string(filepath.Separator)
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}

	return strings.HasPrefix(s, prefix)
}

// Result does not start with separator.
func FilepathSplitAt(s string, n int) string {
	if n > len(s) {
		return ""
	}
	for ; n < len(s); n++ {
		if s[n] != filepath.Separator {
			break
		}
	}
	return s[n:]
}

func FilepathClean(s string) string {
	return filepath.Clean(s)
}
