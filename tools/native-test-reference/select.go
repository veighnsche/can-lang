package reference

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// proposalInputs is the P03 reference-seed.json subset this package verifies.
type proposalInputs struct {
	Status string `json:"status"`
	Head   string `json:"head"`
	Inputs struct {
		Catalogue struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"catalogue"`
		RuntimeTree struct {
			Roots []string `json:"roots"`
			Files int      `json:"files"`
			Bytes int64    `json:"bytes"`
		} `json:"runtime_tree"`
		BunExecutable struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"bun_executable"`
		BunArchive struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"bun_archive"`
		TargetManifest struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"target_manifest"`
	} `json:"inputs"`
}

// ignoredStrays is the proposal-documented count of ignored files included in
// the runtime aggregate counts. Ignored files are never selection identity:
// a pristine checkout holds proposal.files-ignoredStrays tracked files.
const ignoredStrays = 3

// FileHash is one sealed input file.
type FileHash struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// Selection is a verified proposal binding: exact source, pins and hashes.
type Selection struct {
	Head         string     `json:"head"`
	SourceDir    string     `json:"source_dir"`
	PinsDir      string     `json:"pins_dir"`
	ProposalPath string     `json:"proposal_path"`
	Catalogue    FileHash   `json:"catalogue"`
	Target       FileHash   `json:"target_manifest"`
	BunArchive   FileHash   `json:"bun_archive"`
	BunBinary    FileHash   `json:"bun_executable"`
	RuntimeFiles []FileHash `json:"runtime_files"`
	RuntimeCount int        `json:"runtime_count"`
	RuntimeBytes int64      `json:"runtime_bytes"`
}

func hashFile(path string) (FileHash, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileHash{}, err
	}
	sum := sha256.Sum256(data)
	return FileHash{Path: path, SHA256: hex.EncodeToString(sum[:]), Bytes: int64(len(data))}, nil
}

func checkHash(got FileHash, want, what string) error {
	if got.SHA256 != want {
		return fmt.Errorf("reference: %s identity mismatch: got %s want %s", what, got.SHA256, want)
	}
	return nil
}

// SelectFiles verifies proposal file hashes and runtime aggregates against
// sourceDir (pristine selection checkout) and pinsDir (hash-verified local
// pins root, e.g. the live tree holding .local-deps). It does not check git
// state; pair it with VerifyPristine for full selection identity.
func SelectFiles(sourceDir, pinsDir, proposalPath string) (Selection, error) {
	raw, err := os.ReadFile(proposalPath)
	if err != nil {
		return Selection{}, fmt.Errorf("reference: read proposal: %w", err)
	}
	var proposal proposalInputs
	if err := json.Unmarshal(raw, &proposal); err != nil {
		return Selection{}, fmt.Errorf("reference: decode proposal: %w", err)
	}
	if proposal.Head == "" {
		return Selection{}, fmt.Errorf("reference: proposal has no head")
	}
	sel := Selection{Head: proposal.Head, SourceDir: sourceDir, PinsDir: pinsDir, ProposalPath: proposalPath}
	catalogue, err := hashFile(filepath.Join(sourceDir, proposal.Inputs.Catalogue.Path))
	if err != nil {
		return Selection{}, fmt.Errorf("reference: catalogue: %w", err)
	}
	if err := checkHash(catalogue, proposal.Inputs.Catalogue.SHA256, "catalogue"); err != nil {
		return Selection{}, err
	}
	sel.Catalogue = catalogue
	target, err := hashFile(filepath.Join(sourceDir, proposal.Inputs.TargetManifest.Path))
	if err != nil {
		return Selection{}, fmt.Errorf("reference: target manifest: %w", err)
	}
	if err := checkHash(target, proposal.Inputs.TargetManifest.SHA256, "target manifest"); err != nil {
		return Selection{}, err
	}
	sel.Target = target
	archive, err := hashFile(filepath.Join(pinsDir, proposal.Inputs.BunArchive.Path))
	if err != nil {
		return Selection{}, fmt.Errorf("reference: bun archive: %w", err)
	}
	if err := checkHash(archive, proposal.Inputs.BunArchive.SHA256, "bun archive"); err != nil {
		return Selection{}, err
	}
	sel.BunArchive = archive
	binary, err := hashFile(filepath.Join(pinsDir, proposal.Inputs.BunExecutable.Path))
	if err != nil {
		return Selection{}, fmt.Errorf("reference: bun executable: %w", err)
	}
	if err := checkHash(binary, proposal.Inputs.BunExecutable.SHA256, "bun executable"); err != nil {
		return Selection{}, err
	}
	sel.BunBinary = binary
	for _, root := range proposal.Inputs.RuntimeTree.Roots {
		base := filepath.Join(sourceDir, root)
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return err
			}
			hashed, err := hashFile(path)
			if err != nil {
				return err
			}
			hashed.Path = filepath.ToSlash(rel)
			sel.RuntimeFiles = append(sel.RuntimeFiles, hashed)
			sel.RuntimeBytes += hashed.Bytes
			return nil
		})
		if err != nil {
			return Selection{}, fmt.Errorf("reference: runtime tree %s: %w", root, err)
		}
	}
	sel.RuntimeCount = len(sel.RuntimeFiles)
	wantCount := proposal.Inputs.RuntimeTree.Files - ignoredStrays
	if sel.RuntimeCount != wantCount {
		return Selection{}, fmt.Errorf("reference: runtime file count %d, want proposal %d minus %d ignored strays",
			sel.RuntimeCount, proposal.Inputs.RuntimeTree.Files, ignoredStrays)
	}
	if sel.RuntimeBytes > proposal.Inputs.RuntimeTree.Bytes {
		return Selection{}, fmt.Errorf("reference: runtime bytes %d exceed proposal aggregate %d",
			sel.RuntimeBytes, proposal.Inputs.RuntimeTree.Bytes)
	}
	return sel, nil
}

// VerifyPristine requires dir to be a clean checkout at wantHead. Any local
// change, staged or not, fails selection: mutation after selection
// invalidates.
func VerifyPristine(dir, wantHead string) error {
	head, err := gitOutput(dir, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("reference: rev-parse: %w", err)
	}
	if head != wantHead {
		return fmt.Errorf("reference: source head %s, want selection %s", head, wantHead)
	}
	status, err := gitOutput(dir, "status", "--porcelain=v1")
	if err != nil {
		return fmt.Errorf("reference: status: %w", err)
	}
	if status != "" {
		return fmt.Errorf("reference: source tree is dirty:\n%s", status)
	}
	return nil
}

// DistbuildDriftFree requires the bootstrap tool inputs to be byte-identical
// between head and the current checkout, so running the current tree's
// distbuild is equivalent to running the selection's reviewed tools.
func DistbuildDriftFree(gitDir, head string) error {
	cmd := exec.Command("git", "-C", gitDir, "diff", "--quiet", head, "--",
		"tools/distbuild", "distribution", "compiler/internal/catalogue")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reference: bootstrap tools drifted since selection: %w", err)
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", full...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}
