package osutil

import (
	"context"
	"os"
	"os/exec"
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

//----------

func ExecCmdCtxWithAttr(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	SetupExecCmdSysProcAttr(cmd)
	return cmd
}
