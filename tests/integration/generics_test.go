package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCurrentBundledGenerics(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline generic execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	run := func(command string, args ...string) (int, string, string) {
		t.Helper()
		argv := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root}
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

	for _, name := range []string{"definitions.can", "helper.can", "main.can"} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/generics", name))
		if err != nil {
			t.Fatal(err)
		}
		write("src/"+name, string(data))
	}
	status, out, diag := run("assert")
	if status != 0 || diag != "" {
		t.Fatalf("generic assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 6 {
		t.Fatalf("invalid generic report: %v %s", err, out)
	}
	status, out, diag = run("run")
	if status != 0 || out != "" || diag != "" {
		t.Fatalf("generic execution: %d %s %s", status, out, diag)
	}
	// A reachable instance must check a branch that this invocation never takes.
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/generics/definitions.can"))
	if err != nil {
		t.Fatal(err)
	}
	invalid := strings.Replace(string(data), "    ok value\n", "    match true\n        true => ok value\n        false => ok false\n", 1)
	write("src/definitions.can", invalid)
	status, out, diag = run("build")
	if status == 0 || !strings.Contains(diag, "concrete function") {
		t.Fatalf("invalid unused branch admitted: %d %s %s", status, out, diag)
	}
	for _, name := range []string{"definitions.can", "helper.can"} {
		if err := os.Remove(filepath.Join(root, "src", name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"finite-transition", "finite-field-chain", "spread-inferred"} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/generics", name+".can"))
		if err != nil {
			t.Fatal(err)
		}
		variants := []string{string(data)}
		if name == "finite-transition" {
			variants = append(variants, strings.Replace(string(data), "        sample: 1, 0 => ok 0", "        sample: 1, 0 => ok 0\n        array: [1], 0 => ok 0", 1))
			variants = append(variants, strings.Replace(string(data), "fixed<int[]>([1]", "fixed([1]", 1))
		}
		if name == "finite-field-chain" {
			row := "        sample: seed([]), 0 => ok 0\n"
			terminal := "        terminal: done<step<seed>>([]), 0 => ok 0\n"
			variants = append(variants, strings.Replace(string(data), row, terminal+row, 1), strings.Replace(string(data), row, row+terminal, 1))
		}
		for index, source := range variants {
			write("src/main.can", source)
			status, out, diag = run("assert")
			if status != 0 || diag != "" || !strings.Contains(out, `"passed":true`) {
				t.Fatalf("%s variant %d assertions: %d %s %s", name, index, status, out, diag)
			}
			status, out, diag = run("run")
			if status != 0 || out != "" || diag != "" {
				t.Fatalf("%s variant %d execution: %d %s %s", name, index, status, out, diag)
			}
		}
	}

}

// TestCurrentBundledGenericChain proves UP17 end to end: public generic
// composition across a locked dependency succeeds through check, emission
// and execution with an operated archive. The vendor packages carry the
// core/helpers/mail chain fixtures; the root drives nested<int> through
// two syntactic paths plus its assertion row, relays an error completion
// through two generic instances, and passes an owner value through an
// owner-typed instance. Canonical instances deduplicate: fourteen emitted
// functions serve every request site.
func TestCurrentBundledGenericChain(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline generic-chain execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	stage := func(t *testing.T) (string, func(name, text string)) {
		t.Helper()
		root, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		write := func(name, text string) {
			t.Helper()
			p := filepath.Join(root, filepath.FromSlash(name))
			if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(p, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return root, write
	}
	runAt := func(root, command string, args ...string) (int, string, string) {
		t.Helper()
		argv := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command}
		if command == "build" {
			argv = append(argv, args...)
			argv = append(argv, root)
		} else {
			argv = append(argv, root)
			argv = append(argv, args...)
		}
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
	fixture := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/generics", name+".can"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	root, write := stage(t)
	write("can.project.json", `{"source_root":"src","dependencies":{"vendor":"vendor"},"error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	mainSource := fixture("chain-main")
	write("src/app/main.can", mainSource)
	vendorManifest := `{"source_root":"src","error_registry":"can.errors.json"}`
	vendorRegistry := `{"active":["core::denied"],"retired":[]}`
	write("vendor/can.project.json", vendorManifest)
	write("vendor/can.errors.json", vendorRegistry)
	vendorSources := map[string]string{
		"core/core.can":       fixture("chain-core"),
		"helpers/helpers.can": fixture("chain-helpers"),
		"mail/mail.can":       fixture("chain-mail"),
	}
	for name, text := range vendorSources {
		write("vendor/src/"+name, text)
	}
	// The lock below is computed independently (see captureTreeDigest),
	// so staging cross-checks the compiler's digest verification.
	manifestSum := sha256.Sum256([]byte(vendorManifest))
	lock, err := json.Marshal(map[string]any{
		"edges": map[string]any{"vendor": map[string]any{
			"target": "can.project.dependency/vendor", "path": "vendor"}},
		"projects": map[string]any{"can.project.dependency/vendor": map[string]any{
			"lineage": "", "manifest_sha256": hex.EncodeToString(manifestSum[:]),
			"source_sha256":   captureTreeDigest(t, "can-source-tree-v1", vendorSources),
			"fixtures_sha256": captureTreeDigest(t, "can-fixture-tree-v1", map[string]string{}),
			"error_registry":  map[string]any{"active": []any{"core::denied"}, "retired": []any{}},
			"edges":           map[string]any{}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	write("can.lock.json", string(lock))

	status, out, diag := runAt(root, "assert")
	if status != 0 || diag != "" {
		t.Fatalf("chain assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid chain report %v %s", err, out)
	}
	if report["passed"] != true || report["scope"] != "full" {
		t.Fatalf("chain report not a full pass: %s", out)
	}
	rows, ok := report["assertions"].([]any)
	if !ok || len(rows) != 10 {
		t.Fatalf("chain report has %d roots, want 10: %s", len(rows), out)
	}
	roots := map[string]bool{}
	for _, row := range rows {
		entry := row.(map[string]any)
		if entry["passed"] != true {
			t.Fatalf("chain root failed: %v", row)
		}
		real := false
		for _, evidence := range entry["evidence"].([]any) {
			real = real || evidence == "real-can"
		}
		if !real {
			t.Fatalf("chain root lacks real-can evidence: %v", row)
		}
		root := entry["root"].(map[string]any)
		roots[root["declaration"].(string)+" "+root["name"].(string)] = true
	}
	for _, want := range []string{
		"can.project.dependency/vendor/core::identity number",
		"can.project.dependency/vendor/core::refuse sample",
		"can.project.dependency/vendor/helpers::nested number",
		"can.project.dependency/vendor/helpers::pass number",
		"can.project.dependency/vendor/helpers::probe denied",
		"can.project.dependency/vendor/helpers::wrap number",
		"can.project.dependency/vendor/mail::email_address sample",
		"can.project.dependency/vendor/mail::make_email sample",
		"can.project.root/app::main empty",
		"can.project.root/app::top number",
	} {
		if !roots[want] {
			t.Fatalf("chain report misses root %q: %v", want, roots)
		}
	}
	assertGens, err := os.ReadDir(filepath.Join(root, "dist/builds"))
	if err != nil || len(assertGens) != 1 {
		t.Fatalf("assert staged %v, want one generation", assertGens)
	}
	assertGen := filepath.Join(root, "dist/builds", assertGens[0].Name())

	// Execution proves the runtime behavior: both nested<int> values, the
	// direct identity<box<int>> value, the top<int> value, the exact
	// relayed core::denied completion and the owner round-trip. Any
	// divergence divides by zero.
	status, out, diag = runAt(root, "run")
	if status != 0 || out != "" || diag != "" {
		t.Fatalf("chain execution: %d %s %s", status, out, diag)
	}

	status, out, diag = runAt(root, "build")
	if status != 0 || diag != "" {
		t.Fatalf("chain build: %d %s %s", status, out, diag)
	}
	var build map[string]any
	if err = json.Unmarshal([]byte(out), &build); err != nil {
		t.Fatalf("invalid chain build %v %s", err, out)
	}
	if build["kind"] != "can.build" || build["target"] != "bun" || build["validation"] != "verified" {
		t.Fatalf("chain build not verified: %s", out)
	}
	counts := build["assertions"].(map[string]any)
	if counts["roots"] != float64(10) || counts["passed"] != float64(10) || counts["failed"] != float64(0) {
		t.Fatalf("chain build assertions wrong: %s", out)
	}
	if build["inputs"].(map[string]any)["dependencies"] == "" {
		t.Fatalf("chain build omits locked dependency identity: %s", out)
	}
	prodGen := build["directory"].(string)

	// Production emission: fourteen canonical instances across four
	// authored modules. nested<int> serves four request sites with one
	// function; identity<box<int>> serves the nested<int> body and the
	// root's direct inferred call with one function.
	exported := regexp.MustCompile(`export async function \$canFunction\d+`)
	moduleCounts := map[string]int{}
	var helpersMap map[string]any
	err = filepath.WalkDir(filepath.Join(prodGen, "packages"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path + ".map")
		if err != nil {
			return err
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		source := decoded["sources"].([]any)[0].(string)
		moduleCounts[source] = len(exported.FindAll(data, -1))
		if strings.Contains(source, "/helpers/") {
			helpersMap = decoded
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	wantModules := map[string]int{
		"can.project.dependency/vendor/core/core/core.can":          4,
		"can.project.dependency/vendor/helpers/helpers/helpers.can": 6,
		"can.project.dependency/vendor/mail/mail/mail.can":          2,
		"can.project.root/app/app/main.can":                         2,
	}
	if len(moduleCounts) != len(wantModules) {
		t.Fatalf("production modules = %v, want %v", moduleCounts, wantModules)
	}
	for source, want := range wantModules {
		if moduleCounts[source] != want {
			t.Fatalf("production module %s emits %d functions, want %d", source, moduleCounts[source], want)
		}
	}
	// The nested<int> body call to identity<box<int>> is emitted with an
	// exact source-map entry at its call-site bytes.
	call := "core::identity<box<item>>(box(value))"
	helpersSource := vendorSources["helpers/helpers.can"]
	start := strings.Index(helpersSource, call)
	if start < 0 {
		t.Fatalf("helpers fixture lost the nested call site")
	}
	wantCall := "call:" + strconv.Itoa(start) + ":" + strconv.Itoa(start+len(call))
	foundCall := false
	for _, name := range helpersMap["names"].([]any) {
		foundCall = foundCall || name == wantCall
	}
	if !foundCall {
		t.Fatalf("helpers map lacks nested call entry %s: %v", wantCall, helpersMap["names"])
	}

	// Every authored, state, assertion and entry module in both staged
	// generations carries a sibling source map plus a trailer, and every
	// map parses with exact mappings back to .can sources. The staged
	// runtime mirror is owned by distribution tests, not this suite.
	for _, gen := range []string{assertGen, prodGen} {
		checked := 0
		for _, dir := range []string{"packages", "program", "assertions"} {
			root := filepath.Join(gen, dir)
			if _, err := os.Stat(root); err != nil {
				continue
			}
			err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
				if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
					return err
				}
				// The state module is fully synthesized, so its map is
				// structurally complete but carries no .can mappings.
				checkSourceMap(t, path, filepath.Base(path) != "state.ts")
				checked++
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		checkSourceMap(t, filepath.Join(gen, "entry.ts"), false)
		if checked == 0 {
			t.Fatalf("generation %s has no mapped modules", gen)
		}
	}

	// Complete assertion emission: one case module per root, each
	// importing all fourteen concrete instance functions.
	entries, err := os.ReadDir(filepath.Join(assertGen, "assertions"))
	if err != nil {
		t.Fatal(err)
	}
	cases := 0
	instanceRef := regexp.MustCompile(`\$canFunction\d+`)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ts") {
			continue
		}
		cases++
		data, err := os.ReadFile(filepath.Join(assertGen, "assertions", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, match := range instanceRef.FindAll(data, -1) {
			seen[string(match)] = true
		}
		if len(seen) != 14 {
			t.Fatalf("assertion case %s references %d instances, want 14", entry.Name(), len(seen))
		}
	}
	if cases != 10 {
		t.Fatalf("assertion generation has %d cases, want 10", cases)
	}

	// Strict TypeScript over both staged generations. The only tolerated
	// error is the known cross-lane state.ts invalidQuery excess-property
	// error: the UP11 emitter passes the contract but this branch's base
	// predates the UP13 runtime/browser.ts Contracts update (fixed on
	// main at dbeb7bbf), and runtime/platform is B-lane owned and
	// explicitly off-limits to UP17. Authored, assertion and entry
	// modules must be strict-clean unconditionally.
	if tsc := os.Getenv("CAN_TSC"); tsc != "" {
		for _, gen := range []string{assertGen, prodGen} {
			args := []string{tsc, "--noEmit", "--ignoreConfig", "--strict", "--skipLibCheck",
				"--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler",
				"--allowImportingTsExtensions",
				"--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"),
				"--types", "bun,node", filepath.Join(gen, "entry.ts")}
			output, err := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), args...).CombinedOutput()
			if err == nil {
				continue
			}
			for _, line := range strings.Split(string(output), "\n") {
				if !strings.Contains(line, "error TS") {
					continue
				}
				if !strings.Contains(line, "program/state.ts") ||
					!strings.Contains(line, "TS2353") || !strings.Contains(line, "invalidQuery") {
					t.Fatalf("strict generated TypeScript: %v\n%s", err, output)
				}
			}
			t.Logf("generation %s strict-clean except the known UP13 state.ts invalidQuery error", gen)
		}
	}

	// The execution proof is nonvacuous: corrupting one expected
	// composed value fails verification instead of running silent.
	mutated := strings.Replace(mainSource, "direct.value is 9", "direct.value is 8", 1)
	if mutated == mainSource {
		t.Fatal("chain-main fixture lost the direct-value guard")
	}
	write("src/app/main.can", mutated)
	status, out, diag = runAt(root, "run")
	if status == 0 || !strings.Contains(diag, "outcome mismatch") {
		t.Fatalf("mutated chain executed: %d %s %s", status, out, diag)
	}

	// A stale locked dependency fails before any check, emission or
	// execution: the recorded digest no longer matches the bytes.
	stale, staleWrite := stage(t)
	staleWrite("can.project.json", `{"source_root":"src","dependencies":{"vendor":"vendor"},"error_registry":"can.errors.json"}`)
	staleWrite("can.errors.json", `{"active":[],"retired":[]}`)
	staleWrite("src/app/main.can", fixture("chain-main"))
	staleWrite("vendor/can.project.json", vendorManifest)
	staleWrite("vendor/can.errors.json", vendorRegistry)
	for name, text := range vendorSources {
		staleWrite("vendor/src/"+name, text+"\n// stale\n")
	}
	staleWrite("can.lock.json", string(lock))
	status, out, diag = runAt(stale, "assert")
	if status == 0 || !strings.Contains(diag, `stale dependency digest for "can.project.dependency/vendor"`) {
		t.Fatalf("stale lock admitted: %d %s %s", status, out, diag)
	}

	// A failed symbolic component publishes no proof: the dependent
	// declaration cannot reuse it, and the diagnosis is the root
	// arithmetic failure at ::b, never a consumer-side false proof.
	failed, failedWrite := stage(t)
	failedWrite("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	failedWrite("can.errors.json", `{"active":[],"retired":[]}`)
	failedWrite("src/app/main.can", "package app\n    provides [a, b, c]\n    uses []\n"+
		"fn item a<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call b<item>(value, true)\n        true => ok value\n"+
		"fn item b<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call a<item>(value, true)\n        true => ok value + value\n"+
		"fn item c<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 3\n    ok call a<item>(value, true)\n"+
		"fn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n")
	for _, command := range []string{"assert", "build"} {
		status, out, diag = runAt(failed, command)
		if status == 0 || !strings.Contains(diag, "exported generic function") ||
			!strings.Contains(diag, "app::b") || !strings.Contains(diag, "operator + is not defined") {
			t.Fatalf("failed component reused by %s: %d %s %s", command, status, out, diag)
		}
		if strings.Contains(diag, "app::c") {
			t.Fatalf("false proof names the consumer instead of the root: %d %s %s", status, out, diag)
		}
	}

	// Owner negatives: a public generic can pass an owner value through
	// (proven above) but can neither construct the owner record outside
	// its declaring package nor feed an opaque variable to the owner
	// factory's concrete parameter.
	for _, negative := range []struct {
		name string
		body string
		want []string
	}{
		{"construction", "    ok mail::email(\"forged\")\n",
			[]string{"exported generic function", "only be constructed in its declaring package"}},
		{"factory", "    ok call mail::make_email(value)\n",
			[]string{"exported generic function", "explicit named callable or dictionary inputs"}},
	} {
		neg, negWrite := stage(t)
		negWrite("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
		negWrite("can.errors.json", `{"active":[],"retired":[]}`)
		negWrite("src/mail/mail.can", fixture("chain-mail"))
		negWrite("src/app/main.can", "package app\n    provides [suspect]\n    uses [mail]\n"+
			"fn mail::email suspect<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok call mail::make_email(\"a@b\")\n"+
			negative.body+
			"fn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n")
		status, out, diag = runAt(neg, "assert")
		if status == 0 {
			t.Fatalf("owner %s admitted: %d %s %s", negative.name, status, out, diag)
		}
		for _, want := range negative.want {
			if !strings.Contains(diag, want) {
				t.Fatalf("owner %s misdiagnosed, want %q: %d %s %s", negative.name, want, status, out, diag)
			}
		}
	}
}

// checkSourceMap requires a sibling .map plus trailer for one emitted
// module. When mapped, the map must also carry exact mappings back to
// .can sources; synthesized modules (state, entry) only need the
// structural sibling and trailer.
func checkSourceMap(t *testing.T, path string, mapped bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path + ".map")
	if err != nil {
		t.Fatalf("emitted module %s lacks a source map", path)
	}
	base := filepath.Base(path)
	if !strings.HasSuffix(string(data), "//# sourceMappingURL="+base+".map\n") {
		t.Fatalf("emitted module %s lacks an exact source-map trailer", path)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("invalid source map for %s: %v", path, err)
	}
	if !mapped {
		return
	}
	sources, ok := decoded["sources"].([]any)
	if !ok || len(sources) == 0 || decoded["mappings"] == "" {
		t.Fatalf("source map for %s lacks exact mappings: %s", path, raw)
	}
	for _, source := range sources {
		if !strings.Contains(source.(string), ".can") {
			t.Fatalf("source map for %s escapes .can sources: %s", path, raw)
		}
	}
}
