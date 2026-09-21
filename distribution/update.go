package distribution

import (
	"context"
	"fmt"
)

// Update installs a release archive as a new version beside the running one
// and atomically selects it. The previous selection is read before anything
// is staged; install never touches the selection until the new version is
// fully verified, so any failure preserves the previous version by
// structure, and this function re-checks that fact before reporting. Old
// version directories are never modified or removed, and no updater ever
// rewrites a live runtime in place.
func Update(ctx context.Context, archive, shaPath, root string) (previous, current string, err error) {
	previous, err = Selection(root)
	if err != nil {
		return "", "", err
	}
	installed, err := Install(ctx, archive, shaPath, root)
	if err != nil {
		after, checkErr := Selection(root)
		if checkErr != nil {
			return "", "", fmt.Errorf("update release: %v; selection now unreadable: %v", err, checkErr)
		}
		if after != previous {
			return "", "", fmt.Errorf("update release: %v; previous selection lost", err)
		}
		return "", "", err
	}
	current, err = Selection(root)
	if err != nil {
		return "", "", err
	}
	if current != installed {
		return "", "", fmt.Errorf("update release: selection does not point at the installed version")
	}
	return previous, current, nil
}
