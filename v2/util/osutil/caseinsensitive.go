package osutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// TODO: not used yet in the editor (would need to write a tmp file on startup)
// TODO: needs tests

//----------

func FilesystemFilename(filename string) (string, error) {
	inSens, err := IsCaseInsensitiveFileSystemCached()
	if err != nil {
		return "", err
	}
	if inSens {
		return findFsFilename(filename)
	}
	// sensitive, use filename as is
	return filename, nil
}

func findFsFilename(f string) (string, error) {
	if !filepath.IsAbs(f) {
		return "", fmt.Errorf("filename not absolute")
	}
	vol := filepath.VolumeName(f)
	names := []string{}
	for {
		fi, err := os.Stat(f)
		if err != nil {
			return "", err
		}
		names = append(names, fi.Name())
		oldf := f
		f = filepath.Dir(f)
		isRoot := oldf == f
		if isRoot {
			break
		}
	}
	// reverse names
	for i := 0; i < len(names)/2; i++ {
		k := len(names) - 1 - i
		names[i], names[k] = names[k], names[i]
	}
	return vol + filepath.Join(names...), nil
}

//----------

var fsCase = struct {
	checked       bool
	isInsensitive bool
	err           error
}{}

func IsCaseInsensitiveFileSystemCached() (bool, error) {
	if !fsCase.checked {
		fsCase.checked = true
		v, err := isCaseInsensitiveFileSystem()
		fsCase.isInsensitive = v
		fsCase.err = err
	}
	return fsCase.isInsensitive, fsCase.err
}

func isCaseInsensitiveFileSystem() (bool, error) {
	tf := NewTmpFiles("test_case_sensitivity")
	defer tf.RemoveAll()
	// write a file to the filesystem
	name := "a.txt"
	p1, err := tf.WriteFileInTmp(name, []byte("a"))
	if err != nil {
		return false, err
	}
	// check if the file exists with the uppercase name
	p2 := p1[:len(p1)-len(name)] + "A.txt"
	if _, err := os.Stat(p2); os.IsNotExist(err) {
		return false, nil // sensitive
	}
	return true, nil // insensitive
}
