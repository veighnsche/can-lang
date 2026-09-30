package reference

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func packageDir(t *testing.T) string {
	t.Helper()
	dir, err := PackageDir()
	if err != nil {
		t.Fatalf("PackageDir: %v", err)
	}
	return dir
}

func seed(t *testing.T) (root, canlc string, seal Seal) {
	t.Helper()
	var err error
	root, err = SeedRoot()
	if err != nil {
		t.Fatalf("SeedRoot: %v", err)
	}
	canlc, err = ResolveCanlc(root)
	if err != nil {
		t.Fatalf("ResolveCanlc: %v", err)
	}
	seal, err = LoadSeal(root)
	if err != nil {
		t.Fatalf("LoadSeal: %v", err)
	}
	return root, canlc, seal
}

func TestSealIdentity(t *testing.T) {
	_, _, seal := seed(t)
	if len(seal.BundleFiles) != 337 {
		t.Fatalf("sealed bundle files = %d, want 337: truncated seal must not pass", len(seal.BundleFiles))
	}
	if _, ok := seal.BundleFiles[compilerEntry]; !ok {
		t.Fatalf("seal lacks compiler entry %q", compilerEntry)
	}
}

func TestCompilerDigestMatchesSeal(t *testing.T) {
	root, _, seal := seed(t)
	if err := VerifyCompilerDigest(root, seal); err != nil {
		t.Fatalf("compiler digest: %v", err)
	}
}

func TestRuntimeBundleMatchesSeal(t *testing.T) {
	root, _, seal := seed(t)
	if err := VerifyRuntimeDigest(root, seal); err != nil {
		t.Fatalf("runtime digest: %v", err)
	}
}

// TestWrongCompilerDigestDetected is the negative control for the compiler
// digest: a single wrong hex digit in the sealed expectation must fail.
func TestWrongCompilerDigestDetected(t *testing.T) {
	root, _, seal := seed(t)
	seal.BundleFiles[compilerEntry] = corruptHex(t, seal.BundleFiles[compilerEntry])
	err := VerifyCompilerDigest(root, seal)
	if err == nil {
		t.Fatal("wrong compiler digest accepted")
	}
	if !strings.Contains(err.Error(), compilerEntry) {
		t.Fatalf("err = %v, want compiler entry named", err)
	}
	t.Logf("detected: %v", err)
}

// TestWrongRuntimeDigestDetected is the negative control for runtime
// digests: one wrong hex digit in one runtime entry must fail and name it.
func TestWrongRuntimeDigestDetected(t *testing.T) {
	root, _, seal := seed(t)
	names := make([]string, 0, len(seal.BundleFiles))
	for name := range seal.BundleFiles {
		if name != compilerEntry {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	victim := names[0]
	seal.BundleFiles[victim] = corruptHex(t, seal.BundleFiles[victim])
	err := VerifyRuntimeDigest(root, seal)
	if err == nil {
		t.Fatal("wrong runtime digest accepted")
	}
	if !strings.Contains(err.Error(), victim) {
		t.Fatalf("err = %v, want runtime entry %q named", err, victim)
	}
	t.Logf("detected: %v", err)
}

// TestMissingBundleFileDetected requires a sealed-but-absent file to fail:
// changed or missing artifacts fail identity.
func TestMissingBundleFileDetected(t *testing.T) {
	root, _, seal := seed(t)
	seal.BundleFiles["runtime/no-such-file.ts"] = strings.Repeat("0", 64)
	if err := VerifyRuntimeDigest(root, seal); err == nil {
		t.Fatal("missing bundle file accepted")
	} else {
		t.Logf("detected: %v", err)
	}
}

func corruptHex(t *testing.T, sum string) string {
	t.Helper()
	if len(sum) != 64 {
		t.Fatalf("sealed digest %q is not a SHA-256 hex string", sum)
	}
	last := sum[63]
	flipped := byte('0')
	if last == '0' {
		flipped = '1'
	}
	return sum[:63] + string(flipped)
}

func TestFixedNativeObservations(t *testing.T) {
	dir := packageDir(t)
	_, canlc, _ := seed(t)
	obs := filepath.Join(dir, "observations")
	cases := []struct {
		name    string
		fixture string
		args    []string
	}{
		{"pass.parse", "pass", []string{"parse", "src/probe.can"}},
		{"pass.render", "pass", []string{"parse", "-render", "src/probe.can"}},
		{"pass.project", "pass", []string{"inspect-project", "."}},
		{"pass.types", "pass", []string{"inspect-types", "."}},
		// The failing fixture must still be observed by R exactly: the
		// seed observes assertions, and the semantic failure is detected
		// by the transitional host check, not by an observation fault.
		{"fail.parse", "fail", []string{"parse", "src/probe.can"}},
		{"fail.render", "fail", []string{"parse", "-render", "src/probe.can"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := filepath.Join(dir, "fixtures", tc.fixture)
			if err := FixtureSelfContained(fixture); err != nil {
				t.Fatalf("fixture: %v", err)
			}
			got, err := Run(canlc, fixture, tc.args...)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if err := CheckObservation(got, obs, tc.name); err != nil {
				t.Fatalf("observation: %v", err)
			}
		})
	}
}

// TestWrongObservationDetected is the negative control for observations: the
// same native output compared against the wrong fixed record must fail.
func TestWrongObservationDetected(t *testing.T) {
	dir := packageDir(t)
	_, canlc, _ := seed(t)
	obs := filepath.Join(dir, "observations")
	fixture := filepath.Join(dir, "fixtures", "pass")
	got, err := Run(canlc, fixture, "parse", "src/probe.can")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := CheckObservation(got, obs, "pass.render"); err == nil {
		t.Fatal("parse output accepted as render observation")
	} else {
		t.Logf("detected: %v", err)
	}
}

// TestSingleByteObservationDriftDetected proves the comparison is
// byte-exact: one flipped byte in the fixed record must fail.
func TestSingleByteObservationDriftDetected(t *testing.T) {
	dir := packageDir(t)
	_, canlc, _ := seed(t)
	fixture := filepath.Join(dir, "fixtures", "pass")
	got, err := Run(canlc, fixture, "parse", "src/probe.can")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	drift := t.TempDir()
	for _, suffix := range []string{"stdout.txt", "stderr.txt", "exit.txt"} {
		raw, err := os.ReadFile(filepath.Join(dir, "observations", "pass.parse."+suffix))
		if err != nil {
			t.Fatal(err)
		}
		if suffix == "stdout.txt" {
			raw = append([]byte(nil), raw...)
			raw[len(raw)-2]++ // flip one byte, keep the trailing newline
		}
		if err := os.WriteFile(filepath.Join(drift, "drift."+suffix), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckObservation(got, drift, "drift"); err == nil {
		t.Fatal("single-byte observation drift accepted")
	} else {
		t.Logf("detected: %v", err)
	}
}

func fixtureSource(t *testing.T, name string) (source, render string) {
	t.Helper()
	dir := packageDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, "fixtures", name, "src", "probe.can"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	rendered, err := os.ReadFile(filepath.Join(dir, "observations", name+".render.stdout.txt"))
	if err != nil {
		t.Fatalf("read fixed render: %v", err)
	}
	return string(raw), string(rendered)
}

func TestLiteralAssertsPass(t *testing.T) {
	source, render := fixtureSource(t, "pass")
	if err := EvalLiteralAsserts(source, render); err != nil {
		t.Fatalf("passing asserts rejected: %v", err)
	}
}

// TestFailingAssertionDetected is the negative control for assertions: the
// fail fixture's row (want ok 43, body ok 42) must be reported, never
// passed.
func TestFailingAssertionDetected(t *testing.T) {
	source, render := fixtureSource(t, "fail")
	err := EvalLiteralAsserts(source, render)
	if err == nil {
		t.Fatal("failing assertion accepted")
	}
	if !strings.Contains(err.Error(), "failing assertion") || !strings.Contains(err.Error(), "value") {
		t.Fatalf("err = %v, want the failing row named", err)
	}
	t.Logf("detected: %v", err)
}

// TestUnobservedAssertRowDetected requires every evaluated row to appear in
// the staged render: R must observe what the judge will execute.
func TestUnobservedAssertRowDetected(t *testing.T) {
	source, _ := fixtureSource(t, "pass")
	err := EvalLiteralAsserts(source, "package probe\n")
	if err == nil {
		t.Fatal("unobserved assert row accepted")
	}
	if !strings.Contains(err.Error(), "not observed") {
		t.Fatalf("err = %v, want observation fault", err)
	}
	t.Logf("detected: %v", err)
}

// TestOutOfScopeAssertRejected requires the transitional check to fail
// closed on assertions outside its literal bound instead of passing them.
func TestOutOfScopeAssertRejected(t *testing.T) {
	source := `package probe
    provides [answer]
    uses []

fn int answer
    emits {}
    asserts
        value: 1 => ok call other(1)
    ok 42
`
	if err := EvalLiteralAsserts(source, source); err == nil {
		t.Fatal("non-literal assert accepted by the literal check")
	} else if !strings.Contains(err.Error(), "out of scope") {
		t.Fatalf("err = %v, want out-of-scope refusal", err)
	} else {
		t.Logf("detected: %v", err)
	}
}

// TestSeedResolutionIgnoresPathDecoy proves acceptance never resolves a
// compiler via PATH: with a decoy canlc earlier on PATH, the staged
// observation must still match exactly.
func TestSeedResolutionIgnoresPathDecoy(t *testing.T) {
	dir := packageDir(t)
	_, canlc, _ := seed(t)
	decoy := t.TempDir()
	script := filepath.Join(decoy, "canlc")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho decoy-candidate; exit 42\n"), 0755); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(dir, "fixtures", "pass")
	env := append([]string{}, os.Environ()...)
	replaced := false
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			env[i] = "PATH=" + decoy
			replaced = true
		}
	}
	if !replaced {
		env = append(env, "PATH="+decoy)
	}
	got, err := RunEnv(canlc, fixture, env, "parse", "src/probe.can")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := CheckObservation(got, filepath.Join(dir, "observations"), "pass.parse"); err != nil {
		t.Fatalf("PATH decoy affected acceptance: %v", err)
	}
}

// TestRunRejectsBareCommandName forbids PATH lookup at the API level: only
// absolute staged paths run.
func TestRunRejectsBareCommandName(t *testing.T) {
	dir := packageDir(t)
	if _, err := Run("canlc", filepath.Join(dir, "fixtures", "pass"), "parse", "src/probe.can"); err == nil {
		t.Fatal("bare command name accepted")
	}
}

// TestSeedOverrideMustCarrySeal keeps CAN_R_SEED honest: an override without
// the seal and compiler is not a seed.
func TestSeedOverrideMustCarrySeal(t *testing.T) {
	t.Setenv(SeedEnvOverride, t.TempDir())
	if _, err := SeedRoot(); err == nil {
		t.Fatal("seal-less seed override accepted")
	}
}

// TestAcceptanceSeparateFromCandidateAndSuite pins the separation contract:
// the default seed is the one fixed staged directory, fixtures are
// self-contained, and every input the controls read lives either under the
// staged seed or under this package.
func TestAcceptanceSeparateFromCandidateAndSuite(t *testing.T) {
	t.Setenv(SeedEnvOverride, "")
	root, err := SeedRoot()
	if err != nil {
		t.Fatalf("SeedRoot: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(root), StagedSeedDir) {
		t.Fatalf("default seed = %s, want fixed staged %s", root, StagedSeedDir)
	}
	dir := packageDir(t)
	for _, fixture := range []string{"pass", "fail"} {
		if err := FixtureSelfContained(filepath.Join(dir, "fixtures", fixture)); err != nil {
			t.Fatalf("fixture %s: %v", fixture, err)
		}
	}
}

// TestTruncatedSealRejected pins the bundle count in the checker itself: a
// seal missing one entry must fail validation, not just this test's count.
func TestTruncatedSealRejected(t *testing.T) {
	_, _, seal := seed(t)
	for name := range seal.BundleFiles {
		delete(seal.BundleFiles, name)
		break
	}
	if err := validateSeal(seal); err == nil {
		t.Fatal("truncated seal accepted by the checker")
	} else if !strings.Contains(err.Error(), "truncated seal") {
		t.Fatalf("err = %v, want truncation fault", err)
	}
}

func TestSealEntryTraversalRejected(t *testing.T) {
	root, _, _ := seed(t)
	for _, name := range []string{"../evil.ts", "/abs/evil.ts", "runtime/../../evil.ts"} {
		if err := checkEntry(root, name, strings.Repeat("0", 64)); err == nil {
			t.Fatalf("traversal entry %q accepted", name)
		} else if !strings.Contains(err.Error(), "escapes the seed root") {
			t.Fatalf("err = %v, want escape fault", err)
		}
	}
}

// TestSealMatchesCommittedManifest anchors the staged seal to the reviewed
// P04 evidence: without this check the staged seed authenticates itself.
func TestSealMatchesCommittedManifest(t *testing.T) {
	_, _, seal := seed(t)
	if err := VerifySealMatchesCommitted(seal); err != nil {
		t.Fatalf("staged seal drifts from committed manifest: %v", err)
	}
}

// TestTamperedStagedSealDetected mutates a copy of the staged seal (the
// staged seed itself is never touched): one edited digest must fail the
// committed comparison even though nothing else changed.
func TestTamperedStagedSealDetected(t *testing.T) {
	_, _, seal := seed(t)
	seal.BundleFiles[compilerEntry] = corruptHex(t, seal.BundleFiles[compilerEntry])
	if err := VerifySealMatchesCommitted(seal); err == nil {
		t.Fatal("tampered staged seal matches committed manifest")
	} else {
		t.Logf("detected: %v", err)
	}
}

func TestDuplicateAssertRowRejected(t *testing.T) {
	source := `package probe
    provides [answer]
    uses []

fn int answer
    emits {}
    asserts
        value: => ok 43
        value: => ok 42
    ok 42
`
	if err := EvalLiteralAsserts(source, source); err == nil {
		t.Fatal("duplicate assert row accepted")
	} else if !strings.Contains(err.Error(), "duplicate assert row") {
		t.Fatalf("err = %v, want duplicate fault", err)
	}
}

func TestSkippedAssertRowRejected(t *testing.T) {
	source := `package probe
    provides [answer]
    uses []

fn int answer
    emits {}
    asserts
        value: => ok 42
    /// stray line ends row parsing
        skipped: => ok 43
    ok 42
`
	if err := EvalLiteralAsserts(source, source); err == nil {
		t.Fatal("post-block assert row silently skipped")
	} else if !strings.Contains(err.Error(), "outside the asserts block") {
		t.Fatalf("err = %v, want skip fault", err)
	}
}
