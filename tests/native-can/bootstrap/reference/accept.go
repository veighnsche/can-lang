package reference

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Staged seed identity. The directory name is pinned exactly: R is the P04
// materialization, and any other directory is not R.
const (
	// StagedSeedDir is the fixed staged seed directory, relative to the
	// repository root.
	StagedSeedDir = "run/acceptance/can-r-seed-626bf037-bun-1.4.2-darwin-arm64-v1"
	// SealKind and SealVersion are the only accepted seal identity values.
	SealKind    = "can.native-test.seed"
	SealVersion = "r-seed-626bf037"
	// SeedEnvOverride selects an explicit alternate seed root. It never
	// falls back to PATH or to a built candidate.
	SeedEnvOverride = "CAN_R_SEED"
	// compilerEntry is the sealed compiler digest entry.
	compilerEntry = "bin/canlc"
	// sealFile is the P04 acceptance manifest inside the staged seed.
	sealFile = "seed-manifest.json"
	// sealedBundleCount pins the sealed bundle size: 333 runtime inputs plus
	// tsconfig, target manifest, Bun executable and canlc. A shorter seal is
	// a truncated identity, never R.
	sealedBundleCount = 337
	// committedManifest is the reviewed trust anchor: the seal copy committed
	// as P04 evidence, relative to the repository root. The staged seal must
	// match it exactly, or the staged seed authenticates only itself.
	committedManifest = "docs/implementation/native-can-tests-plan-2026-09-30/evidence/P04-manifest.json"
)

// PackageDir returns the absolute path of this control package, so tests
// resolve fixtures and observations without depending on the process
// working directory.
func PackageDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("reference: locate package")
	}
	return filepath.Dir(file), nil
}

// SeedRoot resolves the staged seed R. An explicit CAN_R_SEED override wins;
// otherwise the fixed staged directory under the repository root is used.
// The repository root is found by walking up from this package to go.mod, so
// resolution never depends on the caller working directory, PATH, or any
// candidate build output.
func SeedRoot() (string, error) {
	if override := strings.TrimSpace(os.Getenv(SeedEnvOverride)); override != "" {
		return checkSeedRoot(override)
	}
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	return checkSeedRoot(filepath.Join(root, StagedSeedDir))
}

func checkSeedRoot(root string) (string, error) {
	for _, need := range []string{filepath.Join("bin", "canlc"), sealFile} {
		info, err := os.Stat(filepath.Join(root, need))
		if err != nil {
			return "", fmt.Errorf("reference: seed root %s lacks %s: %w", root, need, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("reference: seed root %s: %s is a directory", root, need)
		}
	}
	return root, nil
}

// ResolveCanlc returns the absolute staged compiler path. It never consults
// PATH: acceptance runs the sealed binary only.
func ResolveCanlc(seedRoot string) (string, error) {
	abs, err := filepath.Abs(filepath.Join(seedRoot, "bin", "canlc"))
	if err != nil {
		return "", fmt.Errorf("reference: resolve canlc: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("reference: staged canlc: %w", err)
	}
	if info.IsDir() || info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("reference: staged canlc %s is not executable", abs)
	}
	return abs, nil
}

// Seal is the P04 acceptance manifest subset these controls verify.
type Seal struct {
	Kind        string            `json:"kind"`
	Version     string            `json:"version"`
	BundleFiles map[string]string `json:"bundle_files"`
}

// LoadSeal reads and validates the seal inside the staged seed root.
func LoadSeal(seedRoot string) (Seal, error) {
	raw, err := os.ReadFile(filepath.Join(seedRoot, sealFile))
	if err != nil {
		return Seal{}, fmt.Errorf("reference: read seal: %w", err)
	}
	var seal Seal
	if err := json.Unmarshal(raw, &seal); err != nil {
		return Seal{}, fmt.Errorf("reference: decode seal: %w", err)
	}
	return seal, validateSeal(seal)
}

// validateSeal enforces seal identity: exact kind, version and bundle size.
func validateSeal(seal Seal) error {
	if seal.Kind != SealKind {
		return fmt.Errorf("reference: seal kind %q, want %q", seal.Kind, SealKind)
	}
	if seal.Version != SealVersion {
		return fmt.Errorf("reference: seal version %q, want %q", seal.Version, SealVersion)
	}
	if len(seal.BundleFiles) != sealedBundleCount {
		return fmt.Errorf("reference: seal holds %d bundle files, want %d: truncated seal must not pass", len(seal.BundleFiles), sealedBundleCount)
	}
	return nil
}

// repoRoot walks up from the control package to the checkout holding go.mod.
func repoRoot() (string, error) {
	pkg, err := PackageDir()
	if err != nil {
		return "", err
	}
	dir := pkg
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("reference: repository root not found above %s", pkg)
		}
		dir = parent
	}
}

// VerifySealMatchesCommitted requires the staged seal to equal the reviewed
// committed manifest on kind, version and every bundle digest. Without this
// anchor a tampered seed could attest itself: modified bytes plus an edited
// staged seal would verify against nothing but the attacker's own values.
func VerifySealMatchesCommitted(seal Seal) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(root, committedManifest))
	if err != nil {
		return fmt.Errorf("reference: read committed manifest: %w", err)
	}
	var committed Seal
	if err := json.Unmarshal(raw, &committed); err != nil {
		return fmt.Errorf("reference: decode committed manifest: %w", err)
	}
	if seal.Kind != committed.Kind || seal.Version != committed.Version {
		return fmt.Errorf("reference: staged seal %s/%s drifts from committed %s/%s",
			seal.Kind, seal.Version, committed.Kind, committed.Version)
	}
	if len(seal.BundleFiles) != len(committed.BundleFiles) {
		return fmt.Errorf("reference: staged seal holds %d files, committed holds %d",
			len(seal.BundleFiles), len(committed.BundleFiles))
	}
	for name, want := range committed.BundleFiles {
		got, ok := seal.BundleFiles[name]
		if !ok {
			return fmt.Errorf("reference: staged seal drops committed entry %s", name)
		}
		if got != want {
			return fmt.Errorf("reference: staged seal entry %s drifts from committed digest", name)
		}
	}
	return nil
}

// VerifyCompilerDigest re-hashes the staged compiler binary and compares it
// to the sealed digest. A wrong compiler digest fails acceptance.
func VerifyCompilerDigest(seedRoot string, seal Seal) error {
	want, ok := seal.BundleFiles[compilerEntry]
	if !ok {
		return fmt.Errorf("reference: seal lacks compiler entry %q", compilerEntry)
	}
	return checkEntry(seedRoot, compilerEntry, want)
}

// VerifyRuntimeDigest re-hashes every sealed bundle file except the compiler
// and compares each to its sealed digest. A wrong or missing runtime digest
// fails acceptance. All mismatches are reported together.
func VerifyRuntimeDigest(seedRoot string, seal Seal) error {
	names := make([]string, 0, len(seal.BundleFiles))
	for name := range seal.BundleFiles {
		if name == compilerEntry {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return fmt.Errorf("reference: seal holds no runtime entries")
	}
	var faults []string
	for _, name := range names {
		if err := checkEntry(seedRoot, name, seal.BundleFiles[name]); err != nil {
			faults = append(faults, err.Error())
		}
	}
	if len(faults) > 0 {
		return fmt.Errorf("reference: runtime digest mismatch:\n%s", strings.Join(faults, "\n"))
	}
	return nil
}

func checkEntry(seedRoot, name, want string) error {
	rel := filepath.FromSlash(name)
	if rel == "" || filepath.IsAbs(rel) {
		return fmt.Errorf("reference: bundle file %q escapes the seed root", name)
	}
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if seg == ".." {
			return fmt.Errorf("reference: bundle file %q escapes the seed root", name)
		}
	}
	data, err := os.ReadFile(filepath.Join(seedRoot, rel))
	if err != nil {
		return fmt.Errorf("reference: bundle file %s unreadable: %w", name, err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("reference: bundle file %s digest got %s want %s", name, got, want)
	}
	return nil
}

// Result is one staged compiler invocation record.
type Result struct {
	Exit   int
	Stdout []byte
	Stderr []byte
}

// Run executes the staged compiler with an absolute path only. A bare
// command name is rejected so acceptance can never resolve a candidate or
// PATH entry by accident.
func Run(canlc, dir string, args ...string) (Result, error) {
	return RunEnv(canlc, dir, nil, args...)
}

// RunEnv is Run with an explicit extra environment (nil inherits).
func RunEnv(canlc, dir string, env []string, args ...string) (Result, error) {
	if !filepath.IsAbs(canlc) {
		return Result{}, fmt.Errorf("reference: refusing non-absolute compiler path %q", canlc)
	}
	cmd := exec.Command(canlc, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err == nil {
		return result, nil
	}
	if exit, ok := err.(*exec.ExitError); ok {
		result.Exit = exit.ExitCode()
		return result, nil
	}
	return Result{}, fmt.Errorf("reference: run staged canlc: %w", err)
}

// CheckObservation compares a native observation byte-for-byte against the
// fixed record name.{stdout,stderr,exit}.txt under wantDir. Stdout, stderr
// and exit code must all match; any deviation is a wrong observation.
func CheckObservation(got Result, wantDir, name string) error {
	wantOut, err := os.ReadFile(filepath.Join(wantDir, name+".stdout.txt"))
	if err != nil {
		return fmt.Errorf("reference: read fixed %s stdout: %w", name, err)
	}
	wantErr, err := os.ReadFile(filepath.Join(wantDir, name+".stderr.txt"))
	if err != nil {
		return fmt.Errorf("reference: read fixed %s stderr: %w", name, err)
	}
	rawExit, err := os.ReadFile(filepath.Join(wantDir, name+".exit.txt"))
	if err != nil {
		return fmt.Errorf("reference: read fixed %s exit: %w", name, err)
	}
	wantExit, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(string(rawExit), "exit=")))
	if err != nil {
		return fmt.Errorf("reference: decode fixed %s exit: %w", name, err)
	}
	if got.Exit != wantExit {
		return fmt.Errorf("reference: wrong observation %s: exit %d, want %d", name, got.Exit, wantExit)
	}
	if !bytes.Equal(got.Stdout, wantOut) {
		return fmt.Errorf("reference: wrong observation %s stdout (%d bytes, want %d): first difference at byte %d",
			name, len(got.Stdout), len(wantOut), commonPrefix(got.Stdout, wantOut))
	}
	if !bytes.Equal(got.Stderr, wantErr) {
		return fmt.Errorf("reference: wrong observation %s stderr (%d bytes, want %d): first difference at byte %d",
			name, len(got.Stderr), len(wantErr), commonPrefix(got.Stderr, wantErr))
	}
	return nil
}

func commonPrefix(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

var (
	fnHeader   = regexp.MustCompile(`^fn\s+\S+\s+(\w+)\s*$`)
	assertRow  = regexp.MustCompile(`^\s{8}(\w+):\s*(.*?)\s*=>\s*ok\s+(-?\d+)\s*$`)
	assertLike = regexp.MustCompile(`^\s{8}\w+:`)
	bodyResult = regexp.MustCompile(`^    ok\s+(-?\d+)\s*$`)
	whitespace = regexp.MustCompile(`\s+`)
)

// EvalLiteralAsserts is the transitional host check for assert rows. The
// staged seed parses and renders assertions but does not execute them, so
// this check evaluates the bounded literal subset the P05 fixtures use:
// argument-free rows of the form `name: => ok <int>` on a function whose
// body is a single `ok <int>` line. Every row must also appear in the
// staged render output, proving R observed the exact assertion the future
// judge must execute.
//
// Anything outside the literal subset fails closed as out of scope: this
// check never silently passes an assertion it cannot evaluate. A row whose
// expected value differs from the body value is a failing assertion.
func EvalLiteralAsserts(source, render string) error {
	type fn struct {
		name string
		rows map[string]int
		got  *int
	}
	var fns []*fn
	current := func() *fn {
		if len(fns) == 0 {
			return nil
		}
		return fns[len(fns)-1]
	}
	inAsserts := false
	seenAsserts := false
	for i, line := range sourceLines(source) {
		if m := fnHeader.FindStringSubmatch(line); m != nil {
			fns = append(fns, &fn{name: m[1], rows: map[string]int{}})
			inAsserts = false
			seenAsserts = false
			continue
		}
		fn := current()
		if fn == nil {
			continue
		}
		if strings.TrimSpace(line) == "asserts" {
			inAsserts = true
			seenAsserts = true
			continue
		}
		if inAsserts {
			if assertLike.MatchString(line) {
				m := assertRow.FindStringSubmatch(line)
				if m == nil || strings.TrimSpace(m[2]) != "" {
					return fmt.Errorf("reference: out of scope for transitional check (line %d): non-literal assert row %q", i+1, strings.TrimSpace(line))
				}
				if _, dup := fn.rows[m[1]]; dup {
					return fmt.Errorf("reference: out of scope for transitional check (line %d): duplicate assert row %q", i+1, m[1])
				}
				want, _ := strconv.Atoi(m[3])
				fn.rows[m[1]] = want
				continue
			}
			inAsserts = false
		} else if seenAsserts && assertLike.MatchString(line) {
			return fmt.Errorf("reference: out of scope for transitional check (line %d): assert row outside the asserts block would be skipped", i+1)
		}
		if m := bodyResult.FindStringSubmatch(line); m != nil {
			got, _ := strconv.Atoi(m[1])
			if fn.got != nil {
				return fmt.Errorf("reference: out of scope for transitional check: function %q has more than one body result line", fn.name)
			}
			fn.got = &got
		}
	}
	checked := 0
	fold := func(s string) string { return whitespace.ReplaceAllString(s, " ") }
	foldedRender := fold(render)
	for _, fn := range fns {
		for row, want := range fn.rows {
			checked++
			probe := fold(fmt.Sprintf("%s: => ok %d", row, want))
			if !strings.Contains(foldedRender, probe) {
				return fmt.Errorf("reference: assert row %s.%s not observed in staged render", fn.name, row)
			}
			if fn.got == nil {
				return fmt.Errorf("reference: out of scope for transitional check: function %q has no literal body result", fn.name)
			}
			if want != *fn.got {
				return fmt.Errorf("reference: failing assertion %s.%s: want ok %d, body ok %d", fn.name, row, want, *fn.got)
			}
		}
	}
	if checked == 0 {
		return fmt.Errorf("reference: no assert rows found; check would be vacuous")
	}
	return nil
}

func sourceLines(source string) []string {
	return strings.Split(strings.TrimRight(source, "\n"), "\n")
}

// FixtureSelfContained requires a fixture directory to carry its own project
// inputs: the project manifest, the error registry and at least one source
// file. Controls must never reach outside the fixture for suite sources.
func FixtureSelfContained(dir string) error {
	for _, need := range []string{"can.project.json", "can.errors.json"} {
		if info, err := os.Stat(filepath.Join(dir, need)); err != nil || info.IsDir() {
			return fmt.Errorf("reference: fixture %s lacks %s", dir, need)
		}
	}
	sources, err := filepath.Glob(filepath.Join(dir, "src", "*.can"))
	if err != nil {
		return fmt.Errorf("reference: fixture %s sources: %w", dir, err)
	}
	if len(sources) == 0 {
		return fmt.Errorf("reference: fixture %s has no sources", dir)
	}
	return nil
}
