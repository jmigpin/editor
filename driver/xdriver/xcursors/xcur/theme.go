package xcur

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrThemeNotFound = errors.New("theme not found")

const DefaultSize = 24

var defaultLibraryPaths = []string{
	"~/.local/share/icons",
	"~/.icons",
	"/usr/share/icons",
	"/usr/share/pixmaps",
	"/usr/X11R6/lib/X11/icons",
}

func LoadTheme(name string) (*Theme, error) {
	if name == "" {
		name = "default"
	}
	theme := &Theme{
		Name:    name,
		Cursors: map[string]*Cursor{},
	}
	seen := map[string]bool{}
	if err := theme.load(name, seen); err != nil {
		return nil, err
	}
	return theme, nil
}

func LoadThemeFromEnv() (*Theme, error) {
	return LoadTheme(os.Getenv("XCURSOR_THEME"))
}

func LoadThemeFromDir(path string) (*Theme, error) {
	theme := &Theme{
		Name:    filepath.Base(path),
		Cursors: map[string]*Cursor{},
	}
	if err := theme.loadDir(filepath.Join(path, "cursors")); err != nil {
		return nil, err
	}
	return theme, nil
}

func (t *Theme) load(name string, seen map[string]bool) error {
	if seen[name] {
		return nil
	}
	seen[name] = true

	for _, base := range libraryPaths() {
		dir := filepath.Join(expandHome(base), name)
		if _, err := os.Stat(dir); err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
				continue
			}
			return fmt.Errorf("stat theme: %w", err)
		}

		n := len(t.Cursors)
		if err := t.loadDir(filepath.Join(dir, "cursors")); err != nil {
			switch {
			case errors.Is(err, fs.ErrNotExist):
			case errors.Is(err, fs.ErrPermission):
				continue
			default:
				return fmt.Errorf("load theme dir: %w", err)
			}
		}
		loaded := len(t.Cursors) != n

		inherits, err := loadInherits(filepath.Join(dir, "index.theme"))
		if err != nil {
			switch {
			case errors.Is(err, fs.ErrNotExist):
			case errors.Is(err, fs.ErrPermission) && !loaded:
				continue
			case errors.Is(err, fs.ErrPermission):
			default:
				return fmt.Errorf("load inherits: %w", err)
			}
		}
		for _, inherit := range inherits {
			if err := t.load(inherit, seen); err != nil && !errors.Is(err, ErrThemeNotFound) {
				return fmt.Errorf("load inherited theme %q: %w", inherit, err)
			}
		}
		return nil
	}

	return fmt.Errorf("%w: %q", ErrThemeNotFound, name)
}

func (t *Theme) loadDir(path string) error {
	ents, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, ent := range ents {
		name := ent.Name()
		if _, ok := t.Cursors[name]; ok {
			continue
		}
		if typ := ent.Type().Type(); !typ.IsRegular() && typ != fs.ModeSymlink {
			continue
		}
		cur, err := DecodeFile(filepath.Join(path, name))
		if err != nil {
			if errors.Is(err, ErrBadMagic) {
				continue
			}
			return fmt.Errorf("decode %q: %w", name, err)
		}
		t.Cursors[name] = cur
	}
	return nil
}

func (t *Theme) Images(names []string, size int) []*Image {
	for _, name := range names {
		cur := t.Cursors[name]
		if cur == nil {
			continue
		}
		bestSize := cur.BestSize(size)
		if bestSize == 0 {
			continue
		}
		if imgs := cur.Images[bestSize]; len(imgs) > 0 {
			return imgs
		}
	}
	return nil
}

func SizeFromEnv() int {
	v := os.Getenv("XCURSOR_SIZE")
	if v == "" {
		return DefaultSize
	}
	size, err := strconv.Atoi(v)
	if err != nil || size <= 0 {
		return DefaultSize
	}
	return size
}

func libraryPaths() []string {
	if v := os.Getenv("XCURSOR_PATH"); v != "" {
		return strings.Split(v, string(filepath.ListSeparator))
	}
	return append([]string(nil), defaultLibraryPaths...)
}

func loadInherits(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "Inherits") {
			continue
		}
		_, after, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		parts := strings.FieldsFunc(after, func(ru rune) bool {
			return ru == ',' || ru == ':' || ru == ';'
		})
		inherits := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				inherits = append(inherits, part)
			}
		}
		return inherits, nil
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
