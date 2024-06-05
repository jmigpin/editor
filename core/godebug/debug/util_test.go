package debug

import (
	"io"
	"os"
	"testing"
)

func verboseStdout() io.Writer {
	if testing.Verbose() {
		return os.Stdout
	}
	return nil
}
