package driver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// Bounds for the staging runtime-tree hash: the emitted suite imports the
// runtime tree at execution time, so changed runtime content must invalidate
// the staging lease. The caps keep the hash bounded and deterministic.
const (
	maxStageRuntimeFiles = 2000
	maxStageRuntimeBytes = 64 << 20
)

// StageRunner executes one staged entry file and returns its stdout. The
// production runner shells to bun; tests inject fakes plus one real bun run.
type StageRunner func(ctx context.Context, bunPath, entryFile string, args []string) ([]byte, error)

// StageOptions configures one suite staging.
type StageOptions struct {
	// RuntimeDir is the real runtime tree: it is hashed into the staging
	// key so changed runtime content invalidates the lease.
	RuntimeDir string
	// EmitPrefix is the bundle-namespace prefix for runtime imports
	// (production: runtime/r-<identity>). Dependencies must use it.
	EmitPrefix string
	// Dependencies are extra artifacts linked into the bundle.
	Dependencies []ir.Artifact
	// BunPath is the bun executable for bootstrap runs. Empty means the
	// path lookup is the caller's responsibility (see DefaultStageRunner).
	BunPath string
	// Runner executes bootstrap runs. Nil selects DefaultStageRunner.
	Runner StageRunner
	// Validate marks prepared output validated through the pinned
	// distribution parser (production: Runtime.ValidateOutput). Nil fails
	// closed: staging never marks output validated itself.
	Validate func(context.Context, *PreparedOutput) error
	// EncodeSourceMaps seals diagnostics into the staging bundle the same
	// way production builds do (production: Runtime.encodeSourceMaps).
	// The staged entry configures diagnostics at load, which requires
	// diagnostics/source-index.json. Nil fails closed.
	EncodeSourceMaps func(context.Context, *check.Program, []ir.Artifact) ([]ir.Artifact, error)
	// TimeoutMs bounds each bootstrap run under the P15.1 budget rules.
	TimeoutMs int
}

// DefaultStageRunner runs entryFile with bun under ctx. ctx must already
// carry the run deadline; a missing bun binary fails the run, never skips it.
// Every run provisions fd 3 with an empty launcher environment snapshot:
// emitted entries import the environment adapter at load, and staging
// bootstrap has no caller environment to snapshot. Richer snapshots arrive
// with the controller path (P19) through a custom StageRunner.
func DefaultStageRunner(ctx context.Context, bunPath, entryFile string, args []string) ([]byte, error) {
	if bunPath == "" {
		return nil, fmt.Errorf("driver: staging bootstrap needs a bun path")
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	if _, err := writer.Write([]byte("{}")); err != nil {
		writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, bunPath, append([]string{entryFile}, args...)...)
	cmd.ExtraFiles = []*os.File{reader} // first extra file is fd 3
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("driver: staging bootstrap %v failed: %w: %s", args, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// stageRoot is one listed assertion identity. It mirrors the emit-side shape;
// the bootstrap --list run must print exactly len(program.Assertions) of them.
type stageRoot struct {
	Package     string `json:"package"`
	Declaration string `json:"declaration"`
	Name        string `json:"name"`
}

// hashStageRuntime binds the runtime tree content into the staging key, so a
// changed runtime invalidates the lease even when the runtime path is stable.
// Files walk in lexical order under the staging caps; symlinks are refused.
func hashStageRuntime(dir string) (string, error) {
	var names []string
	var total int64
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("driver: staging runtime tree must not contain symlinks: %s", path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if total > maxStageRuntimeBytes {
			return fmt.Errorf("driver: staging runtime tree exceeds %d bytes", maxStageRuntimeBytes)
		}
		names = append(names, path)
		if len(names) > maxStageRuntimeFiles {
			return fmt.Errorf("driver: staging runtime tree exceeds %d files", maxStageRuntimeFiles)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(names)
	sum := sha256.New()
	for _, name := range names {
		rel, err := filepath.Rel(dir, name)
		if err != nil {
			return "", err
		}
		content, err := os.ReadFile(name)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(sum, "%s\x00%d\x00", rel, len(content))
		sum.Write(content)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// StageSuite stages one immutable live-suite generation without publishing:
// check all source, emit the staging bundle, execute every assertion root
// (one root=<index> run each) plus the --list control for bootstrap, then
// Fill one P27 key and return
// the generation lease for controller/workers. Production publication state
// (current.json) is untouched: the path Stages but never Publishes. A failed
// bootstrap fails the Fill, so failed suites never lease. Key rendering is
// the caller's contract (Go never parses keys); changed source, runtime or
// entry content must arrive as a different key, which misses and rebuilds.
func StageSuite(ctx context.Context, store *OutputStore, cache *ReuseCache, key string, opts StageOptions) (*ReuseLease, error) {
	if store == nil {
		return nil, fmt.Errorf("driver: suite staging needs a store")
	}
	if cache == nil {
		return nil, fmt.Errorf("driver: suite staging needs a reuse cache")
	}
	if opts.RuntimeDir == "" {
		return nil, fmt.Errorf("driver: suite staging needs a runtime directory")
	}
	if opts.EmitPrefix == "" {
		return nil, fmt.Errorf("driver: suite staging needs an emit prefix")
	}
	if opts.Validate == nil {
		return nil, fmt.Errorf("driver: suite staging needs a validator")
	}
	if opts.EncodeSourceMaps == nil {
		return nil, fmt.Errorf("driver: suite staging needs a source map encoder")
	}
	if _, err := CheckAssertTimeoutMs(opts.TimeoutMs); err != nil {
		return nil, err
	}
	runner := opts.Runner
	if runner == nil {
		runner = DefaultStageRunner
	}
	program, err := check.CheckAssertionProgram(store.Graph)
	if err != nil {
		return nil, err
	}
	if len(program.Assertions) == 0 {
		return nil, fmt.Errorf("driver: suite staging needs at least one assertion root")
	}
	runtimeHash, err := hashStageRuntime(opts.RuntimeDir)
	if err != nil {
		return nil, err
	}
	return cache.Fill(ctx, key, func(fillCtx context.Context) (*PreparedOutput, error) {
		artifacts, err := emit.StagingModules(program, opts.EmitPrefix, opts.Dependencies)
		if err != nil {
			return nil, err
		}
		artifacts, err = opts.EncodeSourceMaps(fillCtx, program, artifacts)
		if err != nil {
			return nil, err
		}
		var entryBytes []byte
		for _, artifact := range artifacts {
			if artifact.Path == "entry.ts" {
				entryBytes = artifact.Bytes
			}
		}
		if entryBytes == nil {
			return nil, fmt.Errorf("driver: staging bundle has no entry.ts")
		}
		entryDigest := sha256.Sum256(entryBytes)
		compilerID := sha256.Sum256(append([]byte("can-stage-suite-v1\x00"), entryDigest[:]...))
		options, _ := json.Marshal(struct {
			Schema        int
			Staging       bool
			Roots         int
			EntryDigest   string
			RuntimeHash   string
			CatalogueHash string
		}{1, true, len(program.Assertions), hex.EncodeToString(entryDigest[:]), runtimeHash, catalogue.SourceHash()})
		optionsID := sha256.Sum256(options)
		inputs := store.BuildInputs(hex.EncodeToString(compilerID[:]), catalogue.SourceHash(), runtimeHash, hex.EncodeToString(optionsID[:]))
		prepared, err := PrepareOutput(inputs, "entry.ts", artifacts)
		if err != nil {
			return nil, err
		}
		if err := opts.Validate(fillCtx, prepared); err != nil {
			return nil, err
		}
		stagingDir, err := os.MkdirTemp("", "can-stage-suite-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(stagingDir)
		for name, data := range prepared.files {
			target := filepath.Join(stagingDir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return nil, err
			}
		}
		entryFile := filepath.Join(stagingDir, "entry.ts")
		for i := range program.Assertions {
			selector := fmt.Sprintf("root=%d", i)
			if _, err := runner(fillCtx, opts.BunPath, entryFile, []string{selector}); err != nil {
				return nil, fmt.Errorf("driver: staging bootstrap run %s failed: %w", selector, err)
			}
		}
		listed, err := runner(fillCtx, opts.BunPath, entryFile, []string{"--list"})
		if err != nil {
			return nil, fmt.Errorf("driver: staging bootstrap list failed: %w", err)
		}
		var roots []stageRoot
		if err := json.Unmarshal(bytes.TrimSpace(listed), &roots); err != nil {
			return nil, fmt.Errorf("driver: staging bootstrap list is not root JSON: %w", err)
		}
		if len(roots) != len(program.Assertions) {
			return nil, fmt.Errorf("driver: staging bootstrap listed %d roots for %d assertions", len(roots), len(program.Assertions))
		}
		return prepared, nil
	})
}
