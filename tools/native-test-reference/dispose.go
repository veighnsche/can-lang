package reference

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Receipt records one disposal: what was removed, when, and by which owner.
// Staging workspaces (pristine checkouts, temp dirs) are disposed with a
// receipt immediately after the seal; the staged seed itself is disposed
// only at retirement through DisposeSeed.
type Receipt struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Path          string `json:"path"`
	RemovedUTC    string `json:"removed_utc"`
	OwnerUID      int    `json:"owner_uid"`
	OwnerGID      int    `json:"owner_gid"`
	Note          string `json:"note,omitempty"`
}

func dispose(path, kind, note string, uid, gid int) (Receipt, error) {
	if path == "" {
		return Receipt{}, fmt.Errorf("reference: refuse to dispose empty path")
	}
	if err := os.RemoveAll(path); err != nil {
		return Receipt{}, fmt.Errorf("reference: dispose %s: %w", path, err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return Receipt{}, fmt.Errorf("reference: %s survives disposal", path)
	}
	return Receipt{
		SchemaVersion: 1,
		Kind:          kind,
		Path:          path,
		RemovedUTC:    time.Now().UTC().Format(time.RFC3339),
		OwnerUID:      uid,
		OwnerGID:      gid,
		Note:          note,
	}, nil
}

// DisposeWorktree removes a staging checkout and returns its receipt.
func DisposeWorktree(path string, uid, gid int) (Receipt, error) {
	return dispose(path, "can.native-test.seed-worktree-disposal", "pristine selection checkout", uid, gid)
}

// DisposeSeed removes a staged seed root and returns its receipt. It is the
// only supported way to retire a sealed seed.
func DisposeSeed(path string, uid, gid int) (Receipt, error) {
	return dispose(path, "can.native-test.seed-disposal", "staged seed root", uid, gid)
}

// WriteReceipt writes receipt as indented JSON to path.
func WriteReceipt(path string, receipt Receipt) error {
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(path, encoded, 0644); err != nil {
		return fmt.Errorf("reference: write receipt: %w", err)
	}
	return nil
}
