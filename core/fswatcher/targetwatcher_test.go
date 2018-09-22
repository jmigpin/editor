package fswatcher

import (
	"os"
	"path"
	"testing"
)

func init() {
	//log.SetFlags(0)
}

func TestTargetWatcher1(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewTargetWatcher(mustNew(t))
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	mustAdd(t, w, dir4)
	mustMkdirAll(t, dir4)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == dir4 && ev.Op.HasAny(Create)
	})

	mustRemoveAllFile(t, dir2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == dir4 && ev.Op.HasAny(Remove)
	})

	mustRemove(t, w, dir4)
	mustAdd(t, w, file1)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Create)
	})
}

func TestTargetWatcher2(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewTargetWatcher(mustNew(t))
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")
	file2 := path.Join(dir4, "file2.txt")

	mustAdd(t, w, file2)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)
	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Create)
	})

	mustRemove(t, w, file2)

	mustRemoveAllFile(t, dir2)
	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)

	mustAdd(t, w, file2)
	mustRenameFile(t, file1, file2)

	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file2 && ev.Op.HasAny(Create|Rename)
	})
}

func TestTargetWatcher3(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewTargetWatcher(mustNew(t))
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	mustAdd(t, w, file1)
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

func TestTargetWatcher4(t *testing.T) {
	tmpDir := tmpDir()
	defer os.RemoveAll(tmpDir)

	w := NewTargetWatcher(mustNew(t))
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")
	file2 := path.Join(dir4, "file2.txt")

	dir3_ := path.Join(dir2, "dir3_")
	//dir4_ := path.Join(dir3_, "dir4")
	//file1_ := path.Join(dir4_, "file1.txt")

	mustMkdirAll(t, dir4)
	mustCreateFile(t, file1)
	mustCreateFile(t, file2)
	mustAdd(t, w, file1)
	mustAdd(t, w, file2)
	mustRenameFile(t, dir3, dir3_)
	//mustWriteFile(t, file1_)

	// TODO
	readEvent(t, w, true, func(ev *Event) bool {
		return ev.Name == file1 && ev.Op.HasAny(Create)
	})

	//mustWriteFile(t, file1)

	//readEvent(t, w, true, func(ev *Event) bool {
	//	return ev.Name == file1 && ev.Op.HasAny(Modify)
	//})
}
