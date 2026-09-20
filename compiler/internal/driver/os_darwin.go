//go:build darwin

package driver

import "syscall"

func hostOSVersion() (string, error) { return syscall.Sysctl("kern.osproductversion") }
