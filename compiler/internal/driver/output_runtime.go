package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/veighnsche/can-lang/distribution"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type RuntimeArtifacts struct {
	Directory, Identity string
	Files               []OutputArtifact
}

// PrivateArtifacts reads only the distribution's closed inventory and verifies
// every original module hash. One namespace is shared by the entire generation.
func (r *Runtime) PrivateArtifacts() (RuntimeArtifacts, error) {
	read := func(name string) ([]byte, error) {
		raw, err := regularFile(r.Root, name, false)
		if err != nil {
			return nil, err
		}
		if r.manifest.Files[name] == "" || hashBytes(raw) != r.manifest.Files[name] {
			return nil, fmt.Errorf("private runtime module digest mismatch: %s", name)
		}
		return raw, nil
	}
	raw, err := read("runtime/modules.json")
	if err != nil {
		return RuntimeArtifacts{}, err
	}
	var inventory struct {
		SchemaVersion int                 `json:"schemaVersion"`
		Modules       map[string][]string `json:"modules"`
	}
	if err = decodeOutput(raw, &inventory); err != nil || inventory.SchemaVersion != 1 || len(inventory.Modules) == 0 {
		return RuntimeArtifacts{}, fmt.Errorf("invalid private runtime module inventory")
	}
	hashes := map[string]string{}
	contents := map[string][]byte{}
	for _, name := range sortedOutputKeys(inventory.Modules) {
		if err = outputPath(name); err != nil || !strings.HasSuffix(name, ".ts") {
			return RuntimeArtifacts{}, fmt.Errorf("unsafe runtime module name")
		}
		contents[name], err = read("runtime/" + name)
		if err != nil {
			return RuntimeArtifacts{}, err
		}
		hashes[name] = hashBytes(contents[name])
	}
	encoded, _ := json.Marshal(struct {
		Target         string
		TargetManifest string
		Modules        map[string]string
	}{r.manifest.TargetID, hashBytes(distribution.TargetJSON), hashes})
	id := hashBytes(encoded)
	result := RuntimeArtifacts{Directory: "runtime/r-" + id, Identity: id}
	for _, name := range sortedOutputKeys(inventory.Modules) {
		artifact := OutputArtifact{Path: result.Directory + "/" + name, Bytes: contents[name], Runtime: true}
		for _, specifier := range inventory.Modules[name] {
			if strings.HasPrefix(specifier, "node:") {
				artifact.NativeImports = append(artifact.NativeImports, specifier)
			} else {
				if !strings.HasPrefix(specifier, "./") && !strings.HasPrefix(specifier, "../") {
					return RuntimeArtifacts{}, fmt.Errorf("nonrelative private runtime import")
				}
				target := path.Join(path.Dir(name), specifier)
				if hashes[target] == "" {
					return RuntimeArtifacts{}, fmt.Errorf("missing private runtime import")
				}
				artifact.Imports = append(artifact.Imports, specifier)
			}
		}
		result.Files = append(result.Files, artifact)
	}
	return result, nil
}

// ValidateOutput invokes only the pinned distribution parser tool. Parsing never
// executes authored expressions, loads an import, or installs a package. Type-only
// edges remain in the Go inventory even when the native transpiler erases them.
func (r *Runtime) ValidateOutput(ctx context.Context, prepared *PreparedOutput) error {
	if prepared == nil {
		return fmt.Errorf("missing output")
	}
	if err := r.validateRuntimeManifest(prepared.manifest); err != nil {
		return err
	}
	type module struct {
		Path    string   `json:"path"`
		Source  string   `json:"source"`
		Imports []string `json:"imports"`
	}
	request := struct {
		SchemaVersion int      `json:"schemaVersion"`
		Modules       []module `json:"modules"`
	}{SchemaVersion: 1, Modules: []module{}}
	for _, name := range sortedOutputKeys(prepared.manifest.Imports) {
		imports := append([]string{}, prepared.manifest.Imports[name]...)
		imports = append(imports, prepared.manifest.NativeImports[name]...)
		request.Modules = append(request.Modules, module{name, string(prepared.files[name]), imports})
	}
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	var stdout, stderr bytes.Buffer
	if err = r.RunTool(ctx, "tools/runtime/output-check.ts", nil, nil, bytes.NewReader(data), &stdout, &stderr); err != nil {
		return fmt.Errorf("generated TypeScript validation failed: %w: %s", err, stderr.String())
	}
	var report struct {
		SchemaVersion int    `json:"schemaVersion"`
		Kind          string `json:"kind"`
		Modules       int    `json:"modules"`
		RequestSHA256 string `json:"requestSHA256"`
	}
	if err = decodeOutput(stdout.Bytes(), &report); err != nil || report.SchemaVersion != 1 || report.Kind != "can.output-validation" || report.Modules != len(request.Modules) || report.RequestSHA256 != hashBytes(data) {
		return fmt.Errorf("invalid native output validation report")
	}
	prepared.validated = true
	return nil
}

// RunOutput consumes a validated generation lease. fd 3 remains the private
// environment channel; fd 4 keeps the generation leased until the child exits,
// even if the launcher dies first. Generated code cannot name these host handles.
func (r *Runtime) RunOutput(ctx context.Context, lease *OutputLease, args, environment []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if lease == nil || lease.file == nil {
		return fmt.Errorf("run requires an active generation lease")
	}
	defer lease.Close()
	if err := r.validateRuntimeManifest(lease.Manifest); err != nil {
		return err
	}
	if err := validateOutputManifest(lease.Manifest, true); err != nil {
		return err
	}
	if err := noSymlinkAncestors(lease.Directory); err != nil {
		return err
	}
	root, err := os.OpenRoot(lease.Directory)
	if err != nil {
		return err
	}
	defer root.Close()
	current, err := root.Lstat("manifest.json")
	if err != nil {
		return err
	}
	locked, err := lease.file.Stat()
	if err != nil || !os.SameFile(current, locked) {
		return fmt.Errorf("leased manifest was replaced")
	}
	if err = validateOutputTree(root, ".", lease.Manifest, false, false); err != nil {
		return err
	}
	entry := filepath.Join(lease.Directory, filepath.FromSlash(lease.Manifest.Entry))
	return r.runEntry(ctx, entry, args, environment, stdin, stdout, stderr, []*os.File{lease.file})
}

func (r *Runtime) validateRuntimeManifest(manifest OutputManifest) error {
	runtime, err := r.PrivateArtifacts()
	if err != nil {
		return err
	}
	if manifest.Inputs.Runtime != runtime.Identity {
		return fmt.Errorf("generation runtime identity does not match distribution")
	}
	verified := map[string]string{}
	for _, asset := range runtime.Files {
		verified[asset.Path] = hashBytes(asset.Bytes)
		if manifest.Files[asset.Path] != verified[asset.Path] {
			return fmt.Errorf("private runtime artifact missing or changed")
		}
	}
	for name := range manifest.Files {
		if strings.HasPrefix(name, "runtime/") && verified[name] == "" {
			return fmt.Errorf("unowned private runtime artifact")
		}
	}

	return nil
}
