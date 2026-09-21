package distribution

import "syscall"

func hostOSVersion() (string, error) { return syscall.Sysctl("kern.osproductversion") }
