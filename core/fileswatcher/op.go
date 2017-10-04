// +build !darwin

package fileswatcher

import (
	"fmt"
	"strings"

	"golang.org/x/sys/unix"
)

type Op uint32

func GetCreateOp() Op {
	return Op(unix.IN_CREATE)
}
func GetDeleteOp() Op {
	return Op(unix.IN_DELETE)
}

func (op Op) HasDelete() bool {
	return op&unix.IN_DELETE_SELF+op&unix.IN_DELETE+op&unix.IN_MOVED_FROM+op&unix.IN_MOVE_SELF > 0
}
func (op Op) HasCreate() bool {
	return op&unix.IN_CREATE+op&unix.IN_MOVED_TO > 0
}
func (op Op) HasModify() bool {
	return op&unix.IN_MODIFY > 0
}
func (op Op) HasIgnored() bool {
	return op&unix.IN_IGNORED > 0
}
func (op Op) HasIsDir() bool {
	return op&unix.IN_ISDIR > 0
}

func (op Op) String() string {
	var u []string
	for _, um := range unixMasks {
		if uint32(op)&um.k > 0 {
			u = append(u, um.v)
			op = Op(uint32(op) - um.k)
		}
	}
	if op > 0 {
		u = append(u, fmt.Sprintf("(%v=?)", uint32(op)))
	}
	return strings.Join(u, "|")
}

var unixMasks = []KV{
	KV{unix.IN_CREATE, "create"},
	KV{unix.IN_DELETE, "delete"},
	KV{unix.IN_DELETE_SELF, "deleteSelf"},
	KV{unix.IN_MODIFY, "modify"},
	KV{unix.IN_MOVE_SELF, "moveSelf"},
	KV{unix.IN_MOVED_FROM, "movedFrom"},
	KV{unix.IN_MOVED_TO, "movedTo"},
	KV{unix.IN_IGNORED, "ignored"},
	KV{unix.IN_ISDIR, "isDir"},
	KV{unix.IN_Q_OVERFLOW, "qOverflow"},
}

type KV struct {
	k uint32
	v string
}
