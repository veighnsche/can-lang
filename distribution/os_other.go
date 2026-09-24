//go:build !darwin && !linux

package distribution

import "fmt"

func hostOSVersion() (string, error) {
	return "", fmt.Errorf("verify bundle: OS version only available on darwin and linux")
}
