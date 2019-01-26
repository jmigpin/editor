package fswatcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGWatcher1(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewGWatcher(mustNewFsnWatcher(t))
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")

	mustAddWatch(t, w, dir4)
	mustMkdirAll(t, dir4)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == dir4 && ev.Op.HasAny(Create)
	})

	mustRemoveAll(t, dir2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == dir4 && ev.Op.HasAny(Remove)
	})

	mustRemoveWatch(t, w, dir4)
	mustAddWatch(t, w, file1)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Create)
	})
}

func TestGWatcher2(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewGWatcher(mustNewFsnWatcher(t))
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")
	file2 := filepath.Join(dir4, "file2.txt")

	mustAddWatch(t, w, file2)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)
	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Create)
	})

	mustRemoveWatch(t, w, file2)

	mustRemoveAll(t, dir2)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	mustAddWatch(t, w, file2)
	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Create|Rename)
	})
}

func TestGWatcher3(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewGWatcher(mustNewFsnWatcher(t))
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")

	mustAddWatch(t, w, file1)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Create)
	})

	mustWriteFile(t, file1)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Modify)
	})
}

func TestGWatcher4(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewGWatcher(mustNewFsnWatcher(t))
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")
	file2 := filepath.Join(dir4, "file2.txt")

	dir3_ := filepath.Join(dir2, "dir3_")

	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)
	mustCreateFile(t, file2)
	mustAddWatch(t, w, file1)
	mustAddWatch(t, w, file2)
	mustRenameFile(t, dir3, dir3_)

	readEvent(t, w, true, func(ev *Event) bool {
		return (ev.Name == file1 || ev.Name == file2) &&
			ev.Op.HasAny(Remove)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return (ev.Name == file1 || ev.Name == file2) &&
			ev.Op.HasAny(Remove)
	})

	mustRenameFile(t, dir3_, dir3)

	readEvent(t, w, true, func(ev *Event) bool {
		return (ev.Name == file1 || ev.Name == file2) &&
			ev.Op.HasAny(Create)
	})
	readEvent(t, w, true, func(ev *Event) bool {
		return (ev.Name == file1 || ev.Name == file2) &&
			ev.Op.HasAny(Create)
	})
}

func TestGWatcher5(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewGWatcher(mustNewFsnWatcher(t))
	defer w.Close()

	dir := tmpDir
	dir2 := filepath.Join(dir, "dir2")
	dir3 := filepath.Join(dir2, "dir3")
	dir4 := filepath.Join(dir3, "dir4")
	file1 := filepath.Join(dir4, "file1.txt")

	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	mustAddWatch(t, w, file1)
	mustAddWatch(t, w, file1)
	mustAddWatch(t, w, file1)

	mustRemoveWatch(t, w, file1)

	mustRemoveAll(t, dir2)

	// should have no "remove" events here (watcher added more then once)

	mustAddWatch(t, w, file1)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Create)
	})
}
