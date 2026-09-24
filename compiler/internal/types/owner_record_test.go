package types

import (
	"strings"
	"testing"
)

func TestOwnerMarkingAndSchemaRejection(t *testing.T) {
	b, file := buildSource(t, sourceHeader+`owner record email
    str address
record wire
    str address
`)
	if err := b.SeedDeclarations(); err != nil {
		t.Fatal(err)
	}
	email, err := b.Resolve(file, annotation(t, "email"), map[string]*Type{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !email.Owner() || email.OwnerPackage() == "" {
		t.Fatalf("owner marking missing: owner=%t package=%q", email.Owner(), email.OwnerPackage())
	}
	wire, err := b.Resolve(file, annotation(t, "wire"), map[string]*Type{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if wire.Owner() {
		t.Fatal("plain record marked as owner")
	}
	if _, err := b.Finish(); err != nil {
		t.Fatal(err)
	}
	if _, err := Schema(email); err == nil {
		t.Fatal("codec schema admitted an owner record root")
	} else if !strings.Contains(err.Error(), "owner record") || !strings.Contains(err.Error(), "owner factory") {
		t.Fatalf("codec diagnosis does not name the owner contract: %v", err)
	}
	if _, err := Form(email); err == nil {
		t.Fatal("form schema admitted an owner record root")
	}
	if _, err := LLMSchema(email); err == nil {
		t.Fatal("LLM schema admitted an owner record root")
	}
	if _, err := SQLSchemaOf(email); err == nil {
		t.Fatal("SQL schema admitted an owner record root")
	}
	if _, err := SQLFieldOf(email); err == nil {
		t.Fatal("SQL field projection admitted an owner record")
	}
	if _, err := Schema(wire); err != nil {
		t.Fatalf("codec schema rejected the plain wire record: %v", err)
	}
}

func TestOwnerReachableThroughNestingIsRejected(t *testing.T) {
	b, file := buildSource(t, sourceHeader+`owner record email
    str address
record holder
    email mail
record other
    int value
variant choice
    holder
    other
`)
	if err := b.SeedDeclarations(); err != nil {
		t.Fatal(err)
	}
	nodes := map[string]*Type{}
	for _, name := range []string{"holder", "choice", "email[]"} {
		node, err := b.Resolve(file, annotation(t, name), map[string]*Type{}, false)
		if err != nil {
			t.Fatal(err)
		}
		nodes[name] = node
	}
	if _, err := b.Finish(); err != nil {
		t.Fatal(err)
	}
	for name, node := range nodes {
		if _, err := Schema(node); err == nil {
			t.Fatalf("codec schema admitted owner record through %s", name)
		}
	}
	if _, err := LLMSchema(nodes["holder"]); err == nil {
		t.Fatal("LLM schema admitted owner record through a field")
	}
}
