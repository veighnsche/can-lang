package syntax

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func TestOwnerRecordDeclaration(t *testing.T) {
	program := testHeader + `owner record email
    str address
owner record seat<item>
    item value
    int count
record plain
    int value
`
	file := parseFile(t, program)
	if len(file.Declarations) != 3 {
		t.Fatalf("declarations: got %d, want 3", len(file.Declarations))
	}
	first, ok := file.Declarations[0].(*RecordDecl)
	if !ok || !first.Owner || first.Name.Text != "email" || len(first.Fields) != 1 {
		t.Fatalf("owner record shape: %#v", file.Declarations[0])
	}
	second, ok := file.Declarations[1].(*RecordDecl)
	if !ok || !second.Owner || len(second.Parameters) != 1 || len(second.Fields) != 2 {
		t.Fatalf("generic owner record shape: %#v", file.Declarations[1])
	}
	third, ok := file.Declarations[2].(*RecordDecl)
	if !ok || third.Owner {
		t.Fatalf("plain record gained owner marking: %#v", file.Declarations[2])
	}
	formatted := Format(file)
	if !strings.Contains(formatted, "owner record email\n") || !strings.Contains(formatted, "owner record seat<item>\n") {
		t.Fatalf("owner prefix lost in rendering:\n%s", formatted)
	}
	if strings.Contains(formatted, "owner record plain") {
		t.Fatalf("plain record rendered as owner:\n%s", formatted)
	}
}

func TestOwnerWordStaysAnOrdinaryName(t *testing.T) {
	program := testHeader + "int owner = 1\n"
	file := parseFile(t, program)
	decl, ok := file.Declarations[0].(*ValueDecl)
	if !ok || decl.Binding.Name.Text != "owner" {
		t.Fatalf("owner binding misparsed: %#v", file.Declarations[0])
	}
}

func TestOwnerRecordRequiresRecordKeyword(t *testing.T) {
	file, err := source.New("bad.can", testHeader+"owner email\n    str address\n")
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(file)
	if result.OK() {
		t.Fatal("lone owner declaration parsed")
	}
}
