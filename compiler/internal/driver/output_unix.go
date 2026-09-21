//go:build darwin || linux

package driver

import (
	"errors"
	"os"
	"syscall"
)

func outputLock(file *os.File, exclusive bool) error {
	mode := syscall.LOCK_SH
	if exclusive {
		mode = syscall.LOCK_EX
	}
	return syscall.Flock(int(file.Fd()), mode|syscall.LOCK_NB)
}
func outputOpen(root *os.Root, name string, flags int, mode os.FileMode) (*os.File, error) {
	return root.OpenFile(name, flags|syscall.O_NOFOLLOW, mode)
}

func outputLockBusy(err error) bool { return errors.Is(err, syscall.EWOULDBLOCK) }
