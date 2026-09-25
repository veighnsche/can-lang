package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const cryptoCommandsManifest = `{"source_root":"src","error_registry":"can.errors.json"}`

func writeCryptoCommandsProject(t *testing.T, root, manifest string) {
	t.Helper()
	writeCryptoCommandsText(t, root, manifest, "")
}

func writeCryptoCommandsText(t *testing.T, root, manifest, text string) {
	t.Helper()
	fixture := text
	if fixture == "" {
		sourceRoot, _ := filepath.Abs("../..")
		raw, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/crypto/commands.can"))
		if err != nil {
			t.Fatal(err)
		}
		fixture = string(raw)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", manifest)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", fixture)
}

func TestCurrentCryptoCommands(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged crypto execution")
	}

	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeCryptoCommandsProject(t, root, cryptoCommandsManifest)
	runAt := func(root, command string, args ...string) (int, string, string) {
		t.Helper()
		argv := []string{
			"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root}
		argv = append(argv, args...)
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), out.String(), diag.String()
		}
		return 0, out.String(), diag.String()
	}
	status, out, diag := runAt(root, "assert")
	if status != 0 || diag != "" {
		t.Fatalf("crypto assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 7 {
		t.Fatalf("invalid crypto report %v %s", err, out)
	}
	buildAt := func(root string) (string, string) {
		t.Helper()
		status, out, diag := runAt(root, "build")
		if status != 0 {
			t.Fatalf("build %s: %d %s %s", root, status, out, diag)
		}
		var build struct {
			BuildID   string `json:"buildID"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(out), &build); err != nil {
			t.Fatal(err)
		}
		return build.BuildID, build.Directory
	}
	firstID, firstDir := buildAt(root)
	secondID, _ := buildAt(root)
	if firstID != secondID {
		t.Fatalf("rebuild changed build identity %s %s", firstID, secondID)
	}
	moved, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeCryptoCommandsProject(t, moved, cryptoCommandsManifest)
	if movedID, _ := buildAt(moved); movedID != firstID {
		t.Fatalf("relocated build changed identity %s %s", movedID, firstID)
	}
	state, err := os.ReadFile(filepath.Join(firstDir, "program/state.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"$canPasswords=$canCreatePasswords(",
		"$canCryptoKeys=$canCreateCryptoKeys(",
		"$canCrypto=$canCreateCryptoPrimitives(",
		"$canCryptoKinds",
		"$canIsCryptoKey($canCryptoKinds[identity],value)",
	} {
		if !strings.Contains(string(state), needle) {
			t.Fatalf("missing emitted crypto wiring %s", needle)
		}
	}
	program, err := filepath.Glob(filepath.Join(firstDir, "packages/*/*.ts"))
	if err != nil || len(program) != 1 {
		t.Fatalf("expected one emitted program unit, saw %v %v", program, err)
	}
	unit, err := os.ReadFile(program[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"$canPasswords.verify", "$canCryptoKeys.generateAESKey",
		"$canCryptoKeys.importEd25519Public", "$canCrypto.hmacSha256",
		"$canCrypto.encryptAesGcmSealed", "$canCrypto.signEd25519",
	} {
		if !strings.Contains(string(unit), needle) {
			t.Fatalf("missing emitted crypto call %s", needle)
		}
	}
	if tsc := os.Getenv("CAN_TSC"); tsc != "" {
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--ignoreConfig", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(firstDir, "entry.ts"))
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v %s", err, result)
		}
	}
	fixture, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/crypto/commands.can"))
	if err != nil {
		t.Fatal(err)
	}
	negatives := []struct {
		name string
		text string
		want string
	}{
		{"argument type", strings.Replace(string(fixture), "match call bytes::from_ints(plaintext)", `match call bytes::from_ints("nope")`, 1), "expression type does not fit expected type"},
		{"missing emits", strings.Replace(string(fixture), "fn int seal_size\n    emits [codec::invalid_data, crypto::key_misuse]", "fn int seal_size\n    emits [codec::invalid_data]", 1), "undeclared escaping domain error crypto::key_misuse"},
	}
	for _, negative := range negatives {
		if negative.text == string(fixture) {
			t.Fatalf("%s replacement missed", negative.name)
		}
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		writeCryptoCommandsText(t, dir, cryptoCommandsManifest, negative.text)
		status, out, diag := runAt(dir, "build")
		if status == 0 || !strings.Contains(out+diag, negative.want) {
			t.Fatalf("%s admitted: %d %s %s", negative.name, status, out, diag)
		}
	}
	// Every command runs real native crypto with no service: entries
	// consume an empty fd-3 environment snapshot at load.
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(home, "snapshot.json")
	if err := os.WriteFile(snapshot, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (int, string, string) {
		t.Helper()
		file, err := os.Open(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{filepath.Join(firstDir, "entry.ts")}, args...)...)
		cmd.Dir = home
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
		cmd.ExtraFiles = []*os.File{file}
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), stdout.String(), stderr.String()
		}
		return 0, stdout.String(), stderr.String()
	}
	checks := 0
	ok := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(args...)
		if status != 0 || out != want || diag != "" {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, out, diag)
		}
	}
	fault := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(args...)
		if status != 1 || out != "" || !strings.Contains(diag, `"error":"`+want+`"`) {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, out, diag)
		}
	}
	ok([]string{"roundtrip"}, "[1,2,3,4]")
	ok([]string{"hmac"}, "[176,52,76,97,216,219,56,83,92,168,175,206,175,11,241,43,136,29,194,0,201,131,61,167,38,233,55,108,46,50,207,247]")
	fault([]string{"misuse"}, "crypto::key_misuse")
	fault([]string{"tamper"}, "crypto::decrypt_failed")
	ok([]string{"ask", "correct horse"}, "match")
	ok([]string{"ask", "wrong"}, "mismatch")
	t.Logf("live crypto: %d checks", checks)
}
