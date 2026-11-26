package core

import (
	"strings"
	"testing"
)

func TestShellArgsToolbarCmd(t *testing.T) {
	tests := []struct {
		args      []string
		wantPlain bool
	}{
		{[]string{"ls"}, true},
		{[]string{"ls", "-la", "./dir"}, true},
		{[]string{"printf", "%s", "a b"}, false},
		{[]string{"echo", "a|b"}, false},
		{[]string{"echo", "*.go"}, false},
		{[]string{"echo", "$HOME"}, false},
		{[]string{"echo", ""}, false},
		{[]string{"echo", "a\nb"}, false},
	}
	for _, tt := range tests {
		got := shellArgsToolbarCmd(tt.args)
		plain := !strings.HasPrefix(got, "Run -shellArgs -- ")
		if plain != tt.wantPlain {
			t.Fatalf("%q: got %q, wantPlain=%v", tt.args, got, tt.wantPlain)
		}
		if plain && got != strings.Join(tt.args, " ") {
			t.Fatalf("%q: got %q", tt.args, got)
		}
	}
}
