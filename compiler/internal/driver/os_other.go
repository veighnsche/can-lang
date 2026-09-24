//go:build !darwin && !linux

package driver

import "fmt"

func hostOSVersion() (string, error) {
	return "", fmt.Errorf("CAN-DIST-PLATFORM: unsupported host; requires darwin/arm64 or linux/amd64")
}
