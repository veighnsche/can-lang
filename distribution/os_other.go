//go:build !darwin

package distribution

import "fmt"

func hostOSVersion() (string, error) {
	return "", fmt.Errorf("verify bundle: OS version only available on darwin")
}
