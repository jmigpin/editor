package osutil

import (
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func HomeEnvVar() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

//----------

func FilepathHasDirPrefix(s, prefix string) bool {
	// ensure it ends in separator
	sep := string(filepath.Separator)
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}

	return strings.HasPrefix(s, prefix)
}

// Result does not start with separator.
func FilepathSplitAt(s string, n int) string {
	if n > len(s) {
		return ""
	}
	for ; n < len(s); n++ {
		if s[n] != filepath.Separator {
			break
		}
	}
	return s[n:]
}

func FilepathClean(s string) string {
	return filepath.Clean(s)
}

//----------

func GetFreeTcpPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	p := l.Addr().(*net.TCPAddr).Port
	return p, nil
}

func RandomPort(simpleSeed, min, max int) int {
	seed := time.Now().UnixNano() + int64(os.Getpid()+simpleSeed)
	ra := rand.New(rand.NewSource(int64(seed)))
	return min + ra.Intn(max-min)
}

//----------

// doesn't wait for the cmd to end
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		c := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		return c.Start()
	case "darwin":
		c := exec.Command("open", url)
		return c.Start()
	default:
		c := exec.Command("xdg-open", url)
		return c.Start()
	}
}

// doesn't wait for the cmd to end
func OpenFilemanager(filename string) error {
	switch runtime.GOOS {
	case "windows":
		c := exec.Command("explorer", "/select,"+filename)
		return c.Start()
	case "darwin":
		c := exec.Command("open", filename)
		return c.Start()
	default:
		c := exec.Command("xdg-open", filename)
		return c.Start()
	}
}
