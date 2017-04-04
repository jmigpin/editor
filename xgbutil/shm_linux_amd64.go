package xgbutil

// taken from
// https://raw.githubusercontent.com/golang/exp/master/shiny/driver/x11driver/shm_linux_amd64.go

import (
	"fmt"
	"syscall"
)

// These constants are from /usr/include/linux/ipc.h
const (
	ipcPrivate = 0
	ipcCreat   = 0x1000
	ipcRmID    = 0
)

func ShmOpen(size int) (shmid uintptr, addr uintptr, err error) {
	// ipcCreat|0777 ?
	shmid, _, errno0 := syscall.RawSyscall(syscall.SYS_SHMGET, ipcPrivate, uintptr(size), ipcCreat|0600)
	if errno0 != 0 {
		return 0, 0, fmt.Errorf("shmget: %v", errno0)
	}
	p, _, errno1 := syscall.RawSyscall(syscall.SYS_SHMAT, shmid, 0, 0)
	if errno1 != 0 {
		return 0, 0, fmt.Errorf("shmat: %v", errno1)
	}
	_, _, errno2 := syscall.RawSyscall(syscall.SYS_SHMCTL, shmid, ipcRmID, 0)
	if errno2 != 0 {
		return 0, 0, fmt.Errorf("shmctl: %v", errno2)
	}
	return shmid, p, nil
}

func ShmClose(p uintptr) error {
	_, _, errno := syscall.RawSyscall(syscall.SYS_SHMDT, p, 0, 0)
	if errno != 0 {
		return fmt.Errorf("shmdt: %v", errno)
	}
	return nil
}
