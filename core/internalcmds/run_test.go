package internalcmds

import (
	"strconv"
	"strings"
	"testing"

	"github.com/jmigpin/editor/core/toolbarparser"
)

func TestParseRunScriptMultiline(t *testing.T) {
	for _, script := range []string{
		"printf 'a|b'\ncd /tmp\npwd",
		strings.Repeat("x", 4000) + "\npwd",
	} {
		data := toolbarparser.Parse("Run -script " + strconv.Quote(script))
		if len(data.Parts) != 1 {
			t.Fatalf("parts: got %d, want 1", len(data.Parts))
		}
		opts, err := parseRunOptions(data.Parts[0])
		if err != nil {
			t.Fatal(err)
		}
		if opts.script != script {
			t.Fatalf("script: got %q, want %q", opts.script, script)
		}
	}
}
