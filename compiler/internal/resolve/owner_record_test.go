package resolve

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

func ownerWorld(t *testing.T) *World {
	t.Helper()
	world, err := buildFiles(t, map[string]string{
		"src/mail/box.can": header("mail", "email, plain", "") + "owner record email\n    str address\nrecord plain\n    int value\n",
		"src/app/main.can": header("app", "", "mail") + "record holder\n    mail::email mail\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	return world
}

func TestOwnerRecordConstructorConfinedToDeclaringPackage(t *testing.T) {
	world := ownerWorld(t)
	owner := fileNamed(world, "mail", "box.can")
	foreign := fileNamed(world, "app", "main.can")
	if owner == nil || foreign == nil {
		t.Fatal("owner or consumer file missing")
	}
	owned, err := owner.Lookup(nil, syntax.QualifiedName{Name: "email"}, ConstructorUse)
	if err != nil {
		t.Fatalf("declaring package cannot construct owner record: %v", err)
	}
	if !owned.Owner || owned.Kind != Record {
		t.Fatalf("owner symbol shape: %+v", owned)
	}
	if _, err := foreign.Lookup(nil, syntax.QualifiedName{Package: "mail", Name: "email"}, ConstructorUse); err == nil {
		t.Fatal("foreign package constructed an owner record")
	} else if !strings.Contains(err.Error(), "only be constructed in its declaring package") {
		t.Fatalf("unexpected constructor diagnosis: %v", err)
	}
}

func TestOwnerRecordExportedProjectionStaysNametypeable(t *testing.T) {
	world := ownerWorld(t)
	foreign := fileNamed(world, "app", "main.can")
	symbol, err := foreign.Lookup(nil, syntax.QualifiedName{Package: "mail", Name: "email"}, TypeUse)
	if err != nil {
		t.Fatalf("exported owner record is not nameable as a type: %v", err)
	}
	if !symbol.Owner || !symbol.Public {
		t.Fatalf("projected symbol shape: %+v", symbol)
	}
	if _, err := foreign.Lookup(nil, syntax.QualifiedName{Package: "mail", Name: "plain"}, ConstructorUse); err != nil {
		t.Fatalf("plain exported record constructor rejected: %v", err)
	}
}
