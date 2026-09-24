package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

// Owner values lower through the ordinary immutable record machinery: the
// package boundary is a compile-time confinement, so construction and
// with-updates in the declaring package emit the same nominal helpers,
// while foreign packages only ever pass the resulting opaque values.
func TestOwnerRecordLowersThroughNominalHelpers(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
		"src/mail/box.can": `package mail
    provides [email, make_email, email_address]
    uses []
owner record email
    str address
fn email make_email
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok email("a@b")
    ok email(address)
fn str email_address
    emits []
    given
        email mail
    asserts
        sample: email("a@b") => ok "a@b"
    ok mail.address
fn email refresh
    emits []
    given
        email mail
    asserts
        sample: email("a@b") => ok email("new")
    ok mail with address = "new"
`,
		"src/app/main.can": `package app
    provides []
    uses [mail]
fn str describe
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    mail::email held = call mail::make_email(address)
    match held
        mail::email => ok call mail::email_address(held)
fn bool same_address
    emits []
    given
        str first
        str second
    asserts
        sample: "a@b", "a@b" => ok true
    mail::email one = call mail::make_email(first)
    mail::email two = call mail::make_email(second)
    ok one is two
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`,
	}
	for name, text := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, artifact := range artifacts {
		joined += string(artifact.Bytes) + "\n"
	}
	if !strings.Contains(joined, "$canRecord(") {
		t.Fatal("owner construction did not lower through the nominal record helper")
	}
	if !strings.Contains(joined, "$canUpdate(") {
		t.Fatal("owner with-update did not lower through the update helper")
	}
	if !strings.Contains(joined, "Bun.deepEquals(") {
		t.Fatal("owner leaf equality did not lower to a deep comparison")
	}
	if !strings.Contains(joined, "$canRecordIdentity(") {
		t.Fatal("owner leaf match did not lower to a nominal identity check")
	}
}
