package iout

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

var MkdirMode os.FileMode = 0770

func MkdirAllWriteFile(filename string, src []byte, m os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), MkdirMode); err != nil {
		return err
	}
	return ioutil.WriteFile(filename, []byte(src), m)
}

func MkdirAllCopyFile(src, dst string, m os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), MkdirMode); err != nil {
		return err
	}
	return CopyFile(src, dst, m)
}

func CopyFile(src, dst string, m os.FileMode) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()
	flags := os.O_RDWR | os.O_CREATE | os.O_TRUNC
	to, err := os.OpenFile(dst, flags, m)
	if err != nil {
		return err
	}
	defer to.Close()
	_, err = io.Copy(to, from)
	return err
}

func MkdirAllCopyFileSync(src, dst string, m os.FileMode) error {
	// must exist in src
	info1, err := os.Stat(src)
	if os.IsNotExist(err) {
		return fmt.Errorf("src not found: %v", src)
	}

	// already exists in dest with same modification time
	info2, err := os.Stat(dst)
	if !os.IsNotExist(err) {
		// compare modification time in src
		if info2.ModTime().Equal(info1.ModTime()) {
			return nil
		}
	}

	if err := MkdirAllCopyFile(src, dst, m); err != nil {
		return err
	}

	// set modtime equal to src to avoid copy next time
	t := info1.ModTime().Local()
	return os.Chtimes(dst, t, t)
}
