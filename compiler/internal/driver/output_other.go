//go:build !darwin && !linux

package driver

import (
	"fmt"
	"os"
)

func outputLock(file *os.File, exclusive bool) error {
	return fmt.Errorf("safe output locking is unsupported on this platform")
}
func outputOpen(root *os.Root, name string, flags int, mode os.FileMode) (*os.File, error) {
	return nil, fmt.Errorf("safe output access is unsupported on this platform")
}

func outputLockBusy(err error) bool { return false }
