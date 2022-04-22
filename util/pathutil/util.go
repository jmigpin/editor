package pathutil

import "path/filepath"

func ReplaceExt(filename, ext string) string {
	ext2 := filepath.Ext(filename)
	tmp := filename[:len(filename)-len(ext2)] // remove ext
	return tmp + ext                          // add ext
}
