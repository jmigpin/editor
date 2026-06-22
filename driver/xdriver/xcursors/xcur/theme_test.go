package xcur

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadThemeIgnoresMissingInherits(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "default")
	cursorDir := filepath.Join(themeDir, "cursors")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "index.theme"), []byte("[Icon Theme]\nInherits=missing-theme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "hand2"), makeTestCursorFile(), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XCURSOR_PATH", dir)
	theme, err := LoadTheme("")
	if err != nil {
		t.Fatal(err)
	}
	if theme.Cursors["hand2"] == nil {
		t.Fatal("missing hand2 cursor")
	}
}
