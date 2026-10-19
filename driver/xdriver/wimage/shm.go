package wimage

import (
	"errors"
	"fmt"
)

func ShmOpen(size int) (shmid, addr uintptr, err error) {
	return shmOpen(size, systemShmOps())
}

func ShmClose(shmid, addr uintptr) error {
	return shmClose(shmid, addr, systemShmOps())
}

//----------

type shmOps struct {
	get    func(int) (uintptr, error)
	attach func(uintptr) (uintptr, error)
	detach func(uintptr) error
	remove func(uintptr) error
}

func shmOpen(size int, ops shmOps) (uintptr, uintptr, error) {
	if size <= 0 {
		return 0, 0, fmt.Errorf("invalid shared memory size: %d", size)
	}
	id, err := ops.get(size)
	if err != nil {
		return 0, 0, fmt.Errorf("shmget: %w", err)
	}
	addr, err := ops.attach(id)
	if err != nil {
		err = fmt.Errorf("shmat: %w", err)
		if removeErr := ops.remove(id); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("shmctl: %w", removeErr))
		}
		return 0, 0, err
	}
	return id, addr, nil
}

func shmClose(id, addr uintptr, ops shmOps) error {
	var err error
	if detachErr := ops.detach(addr); detachErr != nil {
		err = fmt.Errorf("shmdt: %w", detachErr)
	}
	if removeErr := ops.remove(id); removeErr != nil {
		err = errors.Join(err, fmt.Errorf("shmctl: %w", removeErr))
	}
	return err
}
