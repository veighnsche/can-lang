package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOSMinimum(t *testing.T) {
	for _, row := range []struct {
		actual string
		want   bool
	}{{"12.9", false}, {"13", true}, {"13.0", true}, {"13.0.1", true}, {"27.0", true}, {"unknown", false}} {
		if got := atLeast(row.actual, "13.0"); got != row.want {
			t.Errorf("%s: got %v", row.actual, got)
		}
	}
}

func TestAssetPathRefusals(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "valid"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../valid", "/valid", "a/../valid", "link/valid", "."} {
		if _, err := regularFile(root, name, false); err == nil || !strings.Contains(err.Error(), "CAN-DIST-PATH") {
			t.Errorf("accepted %q: %v", name, err)
		}
	}
	if _, err := regularFile(root, "absent", false); err == nil || !strings.Contains(err.Error(), "CAN-DIST-MISSING") {
		t.Fatal(err)
	}
	if _, err := regularFile(root, "valid", true); err == nil || !strings.Contains(err.Error(), "CAN-DIST-PERMISSION") {
		t.Fatal(err)
	}
}
