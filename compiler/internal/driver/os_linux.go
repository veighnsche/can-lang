//go:build linux

package driver

import (
	"fmt"
	"os"
	"strings"
)

// hostOSVersion reports the Debian version from /etc/os-release. The Linux
// target admits Debian only: other distributions, including derivatives,
// refuse rather than run on an unqualified libc.
func hostOSVersion() (string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("CAN-DIST-OS: cannot read OS release: %w", err)
	}
	fields := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		fields[key] = strings.Trim(value, `"`)
	}
	if fields["ID"] != "debian" {
		return "", fmt.Errorf("CAN-DIST-PLATFORM: requires Debian, found %q", fields["ID"])
	}
	if fields["VERSION_ID"] == "" {
		return "", fmt.Errorf("CAN-DIST-OS: Debian VERSION_ID missing")
	}
	return fields["VERSION_ID"], nil
}
