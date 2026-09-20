package distribution

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRejectsBeforePublishing(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "invalid.zip")
	if err := os.WriteFile(archive, []byte("not the pinned archive"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"../escape", "ok"} {
		_, err := Build(context.Background(), ".", filepath.Join(root, "output"), archive, version)
		if err == nil {
			t.Fatal("accepted bad input")
		}
		if version == "ok" && !strings.Contains(err.Error(), "archive size or SHA-256 mismatch") {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "output")); !os.IsNotExist(err) {
		t.Fatal("published output for invalid input")
	}
}
