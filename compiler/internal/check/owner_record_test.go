package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const ownerMailPackage = `package mail
    provides [email, email_wire, make_email, email_address, load_wire]
    uses [codec, bytes]
owner record email
    str address
record email_wire
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
fn email load_verified
    emits []
    given
        email mail
    asserts
        sample: email("a@b") => ok email("verified")
    ok mail with address = "verified"
fn email_wire load_wire
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"address\":\"a@b\"}" => ok email_wire("a@b")
    match chain
        call bytes::from_utf8(text) as bytes::buffer encoded
        call codec::decode_json<email_wire>(encoded) as email_wire decoded
        codec::invalid_data
        ok => ok decoded
`

const ownerAppHeader = `package app
    provides []
    uses [mail, codec, bytes]
`

const ownerAppMain = `fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

func ownerProgram(t *testing.T, app string) (map[string]string, *Program, error) {
	t.Helper()
	files := map[string]string{"src/mail/box.can": ownerMailPackage, "src/app/main.can": ownerAppHeader + app + ownerAppMain}
	program, err := programFixture(t, files)
	return files, program, err
}

func TestOwnerRecordBoundaryPositives(t *testing.T) {
	app := `fn str describe
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
fn str adopt
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"address\":\"a@b\"}" => ok "a@b"
    match chain
        call bytes::from_utf8(text) as bytes::buffer encoded
        call codec::decode_json<mail::email_wire>(encoded) as mail::email_wire wire
        codec::invalid_data
        ok => do
            mail::email held = call mail::make_email(wire.address)
            ok call mail::email_address(held)
`
	if _, _, err := ownerProgram(t, app); err != nil {
		t.Fatalf("owner boundary positives rejected: %v", err)
	}
}

func TestOwnerRecordForeignNegatives(t *testing.T) {
	cases := map[string]struct{ app, want string }{
		"constructor": {`fn mail::email forge
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok call mail::make_email("a@b")
    ok mail::email(address)
`, "only be constructed in its declaring package"},
		"with update": {`fn mail::email rewrite
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok call mail::make_email("a@b")
    mail::email held = call mail::make_email(address)
    ok held with address = "forged"
`, "representation is confined"},
		"field read": {`fn str peek
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    mail::email held = call mail::make_email(address)
    ok held.address
`, "representation is confined"},
		"json decode": {`fn mail::email load_owned
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"address\":\"a@b\"}" => ok call mail::make_email("a@b")
    match chain
        call bytes::from_utf8(text) as bytes::buffer encoded
        call codec::decode_json<mail::email>(encoded) as mail::email decoded
        codec::invalid_data
        ok => ok decoded
`, "not codec-admissible"},
		"json encode": {`fn bytes::buffer expose
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok call bytes::from_utf8("x")
    mail::email held = call mail::make_email(address)
    ok call codec::encode_json<mail::email>(held)
`, "not codec-admissible"},
		"destructure": {`fn str unpack
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    mail::email held = call mail::make_email(address)
    match held
        mail::email(bind addr) => ok addr
`, "representation is confined"},
		"typed wire is not the owner": {`fn str smuggle
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    mail::email_wire wire = mail::email_wire(address)
    ok call mail::email_address(wire)
`, "does not fit expected type"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := ownerProgram(t, tc.app); err == nil {
				t.Fatalf("foreign %s admitted", name)
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("foreign %s diagnosis %q lacks %q", name, err.Error(), tc.want)
			} else {
				assertOwnerSpan(t, name, err)
			}
		})
	}
}

// assertOwnerSpan pins each rejection to the offending consumer file.
// Expression, constructor and codec paths carry structured spans; the
// pattern branch follows its siblings with the file in the region wrap.
func assertOwnerSpan(t *testing.T, name string, err error) {
	t.Helper()
	if located, ok := source.AsLocated(err); ok {
		if !strings.HasSuffix(located.File, "src/app/main.can") {
			t.Fatalf("foreign %s located at %q, want the consumer file", name, located.File)
		}
		return
	}
	if !strings.Contains(err.Error(), "src/app/main.can") {
		t.Fatalf("foreign %s diagnosis names no consumer span: %v", name, err)
	}
}

func TestOwnerRecordInPackageCodecRejection(t *testing.T) {
	mail := ownerMailPackage + `fn email load_owned
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"address\":\"a@b\"}" => ok email("a@b")
    match chain
        call bytes::from_utf8(text) as bytes::buffer encoded
        call codec::decode_json<email>(encoded) as email decoded
        codec::invalid_data
        ok => ok decoded
`
	files := map[string]string{"src/mail/box.can": mail, "src/app/main.can": ownerAppHeader + ownerAppMain}
	if _, err := programFixture(t, files); err == nil {
		t.Fatal("in-package generic decode of an owner record admitted")
	} else if !strings.Contains(err.Error(), "not codec-admissible") {
		t.Fatalf("in-package decode diagnosis lacks the codec contract: %v", err)
	} else if located, ok := source.AsLocated(err); !ok || !strings.HasSuffix(located.File, "src/mail/box.can") {
		t.Fatalf("in-package decode diagnosis is not located at the owner file: %v", err)
	}
}

func TestOwnerRecordFixtureAdmission(t *testing.T) {
	cases := map[string]struct{ app, want string }{
		"assertion input": {`fn mail::email pass_through
    emits []
    given
        mail::email mail
    asserts
        forged: mail::email("a@b") => ok call mail::make_email("a@b")
    ok mail
`, "only be constructed in its declaring package"},
		"supplied completion": {`fn mail::email mint
    emits []
    given
        str address
    asserts
        forged: "a@b" => ok mail::email("a@b")
    ok call mail::make_email(address)
`, "only be constructed in its declaring package"},
		"template use argument": {`fn int double
    emits []
    given
        int value
    asserts
        sample: 2 => ok 4
    ok value + value
fixture doubled for double
    given
        int base
    cases
        base => ok base + base
fn int consumer
    emits []
    asserts
        sample: => ok 4
    match call double(2)
        when
            sample: use doubled(mail::email("a@b"))
        ok int got => ok got
`, "only be constructed in its declaring package"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := ownerProgram(t, tc.app); err == nil {
				t.Fatalf("fixture %s admitted", name)
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("fixture %s diagnosis %q lacks %q", name, err.Error(), tc.want)
			}
		})
	}
}
