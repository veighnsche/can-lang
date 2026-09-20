//go:build !darwin

package driver

import "fmt"

func hostOSVersion() (string, error) {
	return "", fmt.Errorf("CAN-DIST-PLATFORM: requires macOS arm64")
}
