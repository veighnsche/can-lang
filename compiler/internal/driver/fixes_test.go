package driver

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
)

const missingArmMain = `package app
    provides []
    uses [codec]
fn str motto
    emits []
    asserts
        sample: => ok "𝄞"
    ok "𝄞"
fn int number
    emits [codec::invalid_data]
    asserts
        sample: => ok 1
    ok 1
fn int handle
    emits [codec::invalid_data]
    asserts
        sample: => ok 1
    match call number()
        ok int value => ok value
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

// The suggested arm insertion validates against an isolated re-check:
// acceptance is the re-check passing, not the fix being proposed.
func TestValidateFixMissingArm(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": missingArmMain})
	open := canonical(t, filepath.Join(root, "src/main.can"))
	overlay := project.NewOverlay()
	snapshot, err := CheckSnapshot(root, open, overlay)
	if err != nil {
		t.Fatal(err)
	}
	fixes := SuggestedFixes(snapshot, open)
	if len(fixes) != 1 {
		t.Fatalf("missing-arm diagnosis produced %d fixes", len(fixes))
	}
	if err := ValidateFix(root, snapshot, overlay, fixes[0]); err != nil {
		t.Fatalf("forward fix rejected: %v", err)
	}

	doctored := fixes[0]
	doctored.NewText += "        bogus arm line\n"
	if err := ValidateFix(root, snapshot, overlay, doctored); err == nil {
		t.Fatal("doctored fix accepted")
	} else if !strings.Contains(err.Error(), doctored.Title) {
		t.Fatalf("doctored rejection is not diagnosable: %v", err)
	}

	deleting := fixes[0]
	deleting.End = deleting.Start + 1
	if err := ValidateFix(root, snapshot, overlay, deleting); err == nil {
		t.Fatal("deleting fix accepted")
	} else if !strings.Contains(err.Error(), "not an insertion") {
		t.Fatalf("deletion rejection misnamed: %v", err)
	}

	stale := project.NewOverlay()
	if err := stale.Set(open, 7, missingArmMain+"# edited\n"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFix(root, snapshot, stale, fixes[0]); err == nil {
		t.Fatal("stale fix accepted")
	} else if !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale rejection misnamed: %v", err)
	}

	// An unsaved overlay buffer validates at its own version: the isolated
	// re-check reads the overlay bytes, never the disk text.
	edited := project.NewOverlay()
	if err := edited.Set(open, 3, missingArmMain); err != nil {
		t.Fatal(err)
	}
	overlaySnapshot, err := CheckSnapshot(root, open, edited)
	if err != nil {
		t.Fatal(err)
	}
	overlayFixes := SuggestedFixes(overlaySnapshot, open)
	if len(overlayFixes) != 1 || overlayFixes[0].Version != 3 {
		t.Fatalf("overlay diagnosis lost its version: %+v", overlayFixes)
	}
	if err := ValidateFix(root, overlaySnapshot, edited, overlayFixes[0]); err != nil {
		t.Fatalf("overlay fix rejected: %v", err)
	}
}
