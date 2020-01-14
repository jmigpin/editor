package osutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type TmpFiles struct {
	Dir string
}

func NewTmpFiles(prefix string) *TmpFiles {
	f := &TmpFiles{}
	f.setupDir(prefix)
	return f
}

func (tf *TmpFiles) setupDir(prefix string) error {
	d, err := ioutil.TempDir(os.TempDir(), prefix)
	if err != nil {
		return err
	}
	tf.Dir = d
	return nil
}

func (tf *TmpFiles) MkdirInTmp(path string) (string, error) {
	fp := filepath.Join(tf.Dir, path)
	if err := os.MkdirAll(fp, 0700); err != nil {
		return "", err
	}
	return fp, nil
}

func (tf *TmpFiles) WriteFileInTmp(path string, src []byte) (string, error) {
	fp := filepath.Join(tf.Dir, path)
	baseDir := filepath.Dir(fp)
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(fp, src, 0600); err != nil {
		return "", err
	}
	return fp, nil
}

func (tf *TmpFiles) RemoveAll() error {
	return os.RemoveAll(tf.Dir)
}

//----------

func (tf *TmpFiles) MkdirInTmpOrPanic(path string) string {
	s, err := tf.MkdirInTmp(path)
	if err != nil {
		panic(err)
	}
	return s
}

func (tf *TmpFiles) WriteFileInTmpOrPanic(path string, src []byte) string {
	s, err := tf.WriteFileInTmp(path, src)
	if err != nil {
		panic(err)
	}
	return s
}

func (tf *TmpFiles) WriteFileInTmp2OrPanic(path string, src string) string {
	return tf.WriteFileInTmpOrPanic(path, []byte(src))
}
