package driver

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/project"
)

type outputOwner struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	Project       string `json:"project"`
}
type outputCurrent struct {
	SchemaVersion  int    `json:"schemaVersion"`
	BuildID        string `json:"buildID"`
	ManifestSHA256 string `json:"manifestSHA256"`
}
type outputPending struct {
	SchemaVersion int            `json:"schemaVersion"`
	Stage         string         `json:"stage"`
	Manifest      OutputManifest `json:"manifest"`
}

// OutputStore holds the OS project-directory lock from snapshot loading through
// publication. Close releases it, including automatically on process death.
type OutputStore struct {
	Graph             *project.Graph
	projectRoot, dist *os.Root
	lock              *os.File
	snapshot          string
	// testHook injects interruption at durable publication boundaries in tests.
	testHook                     func(string) error
	sourceInput, dependencyInput string
	assetHashes                  map[string]map[string]string
}

func (s *OutputStore) Close() error {
	if s.dist != nil {
		s.dist.Close()
	}
	if s.projectRoot != nil {
		s.projectRoot.Close()
	}
	if s.lock != nil {
		return s.lock.Close()
	}
	return nil
}

func BeginOutput(directory string) (store *OutputStore, err error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	for _, part := range strings.Split(filepath.ToSlash(directory), "/") {
		if part == ".." {
			return nil, fmt.Errorf("parent traversal is forbidden")
		}
	}
	if filepath.Clean(directory) != directory {
		return nil, fmt.Errorf("project path is not normalized")
	}
	if err = noSymlinkAncestors(absolute); err != nil {
		return nil, err
	}
	if absolute == string(filepath.Separator) {
		return nil, fmt.Errorf("refuse filesystem-root project output")
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, err
	}
	s := &OutputStore{projectRoot: root}
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	s.lock, err = root.Open(".")
	if err != nil {
		return nil, err
	}
	if err = outputLock(s.lock, true); err != nil {
		return nil, fmt.Errorf("project build is already locked: %w", err)
	}
	s.Graph, err = project.Load(absolute)
	if err != nil {
		return nil, err
	}
	distPath := filepath.Join(absolute, "dist")
	for _, p := range s.Graph.Projects {
		sourcePath, e := project.ConfinedPath(p.Root, p.Manifest.SourceRoot, true)
		if e != nil {
			return nil, e
		}
		paths := []string{p.Manifest.ErrorRegistry}
		for _, asset := range p.Manifest.Assets {
			paths = append(paths, asset)
		}
		for _, name := range paths {
			input, e := project.ConfinedPath(p.Root, name, false)
			if e != nil {
				return nil, e
			}
			if withinOutput(distPath, input) {
				return nil, fmt.Errorf("dist contains configured project input")
			}
		}
		if withinOutput(distPath, p.Root) || withinOutput(distPath, sourcePath) {
			return nil, fmt.Errorf("dist overlaps a project or source root")
		}
		for _, src := range p.Sources {
			if withinOutput(distPath, src.Path) {
				return nil, fmt.Errorf("dist contains source input")
			}
		}
	}
	s.assetHashes, err = outputAssetHashes(s.Graph)
	if err != nil {
		return nil, err
	}
	s.snapshot = outputSnapshot(s.Graph, s.assetHashes)
	identities := s.captureBuildInputs("", "", "", "")
	s.sourceInput = identities.Source
	s.dependencyInput = identities.Dependencies
	if info, e := root.Lstat("dist"); os.IsNotExist(e) {
		if err = root.Mkdir("dist", 0700); err != nil {
			return nil, err
		}
	} else if e != nil {
		return nil, e
	} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("dist must be a real directory")
	}
	s.dist, err = root.OpenRoot("dist")
	if err != nil {
		return nil, err
	}
	owner := outputOwner{1, "can.output-owner", absolute}
	raw, e := readOutputRegular(s.dist, ".can-owner.json")
	if os.IsNotExist(e) {
		entries, e := outputEntries(s.dist, ".")
		if e != nil {
			return nil, e
		}
		if len(entries) != 0 {
			return nil, fmt.Errorf("nonempty dist is unowned; preserve its contents and choose an empty directory")
		}
		raw, _ = json.Marshal(owner)
		if err = writeOutputNew(s.dist, ".can-owner.json", raw); err != nil {
			return nil, err
		}
	} else if e != nil {
		return nil, e
	} else {
		var found outputOwner
		if e = decodeOutput(raw, &found); e != nil || found != owner {
			return nil, fmt.Errorf("dist owner does not match canonical project")
		}
	}
	if info, e := s.dist.Lstat("builds"); os.IsNotExist(e) {
		if err = s.dist.Mkdir("builds", 0700); err != nil {
			return nil, err
		}
	} else if e != nil {
		return nil, e
	} else if !info.IsDir() {
		return nil, fmt.Errorf("builds is not a real directory")
	}
	if err = s.checkLayout(); err != nil {
		return nil, err
	}
	if err = syncOutputDir(s.dist, "."); err != nil {
		return nil, err
	}
	if err = syncOutputDir(root, "."); err != nil {
		return nil, err
	}
	return s, nil
}
func withinOutput(root, target string) bool {
	return target == root || strings.HasPrefix(target, root+string(filepath.Separator))
}
func noSymlinkAncestors(absolute string) error {
	for current := absolute; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("output ancestor is not a real directory: %s", current)
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return nil
}
func graphSnapshot(g *project.Graph) string {
	type identity struct {
		Key, Manifest, Source string
		Registry              project.Registry
	}
	var values []identity
	for _, key := range sortedOutputKeys(g.Projects) {
		p := g.Projects[key]
		values = append(values, identity{key, p.ManifestSHA256, p.SourceSHA256, p.Registry})
	}
	raw, _ := json.Marshal(values)
	return hashBytes(raw)
}
func (s *OutputStore) BuildInputs(compiler, catalogue, runtime, options string) BuildInputs {
	return BuildInputs{Source: s.sourceInput, Dependencies: s.dependencyInput, Compiler: compiler, Catalogue: catalogue, Runtime: runtime, Options: options}
}

func (s *OutputStore) captureBuildInputs(compiler, catalogue, runtime, options string) BuildInputs {
	root := s.Graph.Root
	rootBytes, _ := json.Marshal(struct {
		Manifest, Source string
		Registry         project.Registry
		Assets           map[string]string
	}{root.ManifestSHA256, root.SourceSHA256, root.Registry, s.assetHashes[""]})
	dependencies := map[string]any{}
	for key, p := range s.Graph.Projects {
		if key != "" {
			dependencies[key] = struct {
				Manifest, Source string
				Registry         project.Registry
				Assets           map[string]string
			}{p.ManifestSHA256, p.SourceSHA256, p.Registry, s.assetHashes[key]}
		}
	}
	dependencyBytes, _ := json.Marshal(dependencies)
	return BuildInputs{Source: hashBytes(rootBytes), Dependencies: hashBytes(dependencyBytes), Compiler: compiler, Catalogue: catalogue, Runtime: runtime, Options: options}
}

func decodeOutput(raw []byte, value any) error {
	if err := uniqueOutputJSON(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("trailing output metadata")
	}
	return nil
}
func readOutputRegular(root *os.Root, name string) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("output is not a regular file: %s", name)
	}
	file, err := outputOpen(root, name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil || !os.SameFile(info, actual) {
		return nil, fmt.Errorf("output changed while opening")
	}
	return io.ReadAll(file)
}
func outputEntries(root *os.Root, name string) ([]os.DirEntry, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadDir(-1)
}
func syncOutputDir(root *os.Root, name string) error {
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}
func writeOutputNew(root *os.Root, name string, data []byte) error {
	file, err := outputOpen(root, name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func (s *OutputStore) checkLayout() error {
	raw, err := readOutputRegular(s.dist, ".can-owner.json")
	if err != nil {
		return err
	}
	var owner outputOwner
	if err = decodeOutput(raw, &owner); err != nil || owner != (outputOwner{1, "can.output-owner", s.projectRoot.Name()}) {
		return fmt.Errorf("dist owner changed or does not match project")
	}
	if err = noSymlinkAncestors(s.projectRoot.Name()); err != nil {
		return err
	}
	actual, err := s.projectRoot.Lstat("dist")
	if err != nil || !actual.IsDir() {
		return fmt.Errorf("dist is no longer a real directory")
	}
	opened, err := s.dist.Stat(".")
	if err != nil || !os.SameFile(actual, opened) {
		return fmt.Errorf("dist directory was replaced")
	}
	entries, err := outputEntries(s.dist, ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		switch e.Name() {
		case "builds":
			if !e.IsDir() {
				return fmt.Errorf("invalid builds directory")
			}
		case ".can-owner.json", "current.json", "pending.json", ".current.tmp", ".pending.tmp":
			if !e.Type().IsRegular() {
				return fmt.Errorf("output metadata is not regular")
			}
		default:
			return fmt.Errorf("unknown dist entry %s; preserve it before cleanup", e.Name())
		}
	}
	return nil
}
func (s *OutputStore) atomicMetadata(name, temp string, value any) error {
	if _, err := s.dist.Lstat(temp); !os.IsNotExist(err) {
		return fmt.Errorf("unrecovered output metadata temporary %s", temp)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err = writeOutputNew(s.dist, temp, data); err != nil {
		return err
	}
	if _, err = s.dist.Lstat(name); err == nil {
		if _, err = readOutputRegular(s.dist, name); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err = s.dist.Rename(temp, name); err != nil {
		return err
	}
	return syncOutputDir(s.dist, ".")
}
func (s *OutputStore) hook(point string) error {
	if s.testHook != nil {
		return s.testHook(point)
	}
	return nil
}

func (s *OutputStore) Publish(prepared *PreparedOutput) (string, error) {
	if prepared == nil || !prepared.validated {
		return "", fmt.Errorf("generation has not passed native syntax validation")
	}
	var err error
	if err = s.Recover(); err != nil {
		return "", err
	}
	actual := s.BuildInputs("", "", "", "")
	if prepared.manifest.Inputs.Source != actual.Source || prepared.manifest.Inputs.Dependencies != actual.Dependencies {
		return "", fmt.Errorf("generation does not bind this project snapshot")
	}
	id := prepared.manifest.BuildID
	final := "builds/" + id
	if _, err = s.dist.Lstat(final); err == nil {
		if _, _, err = s.generation(id, false); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	} else {
		var random [16]byte
		if _, err = rand.Read(random[:]); err != nil {
			return "", err
		}
		stage := "builds/.stage-" + hex.EncodeToString(random[:])
		pending := outputPending{1, stage, prepared.manifest}
		if err = s.atomicMetadata("pending.json", ".pending.tmp", pending); err != nil {
			return "", err
		}
		if err = s.hook("reserved"); err != nil {
			return "", err
		}
		if err = s.dist.Mkdir(stage, 0700); err != nil {
			return "", err
		}
		if err = writeOutputNew(s.dist, stage+"/manifest.json", prepared.manifestBytes); err != nil {
			return "", err
		}
		for _, name := range sortedOutputKeys(prepared.files) {
			if err = s.dist.MkdirAll(path.Dir(stage+"/"+name), 0700); err != nil {
				return "", err
			}
			if err = writeOutputNew(s.dist, stage+"/"+name, prepared.files[name]); err != nil {
				return "", err
			}
			if err = s.hook("file:" + name); err != nil {
				return "", err
			}
		}
		if err = validateOutputTree(s.dist, stage, prepared.manifest, false, false); err != nil {
			return "", err
		}
		if err = syncOutputTree(s.dist, stage); err != nil {
			return "", err
		}
		if err = s.hook("staged"); err != nil {
			return "", err
		}
		if err = s.dist.Rename(stage, final); err != nil {
			return "", err
		}
		if err = syncOutputDir(s.dist, "builds"); err != nil {
			return "", err
		}
		if err = s.hook("generation"); err != nil {
			return "", err
		}
	}
	current := outputCurrent{1, id, hashBytes(prepared.manifestBytes)}
	if err = s.atomicMetadata("current.json", ".current.tmp", current); err != nil {
		return "", err
	}
	if err = s.hook("published"); err != nil {
		return "", err
	}
	if err = s.dist.Remove("pending.json"); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err = syncOutputDir(s.dist, "."); err != nil {
		return "", err
	}
	return filepath.Join(s.Graph.Root.Root, "dist", filepath.FromSlash(final)), nil
}

func (s *OutputStore) generation(id string, partial bool) (OutputManifest, []byte, error) {
	var manifest OutputManifest
	if !digestPattern.MatchString(id) {
		return manifest, nil, fmt.Errorf("invalid generation name")
	}
	raw, err := readOutputRegular(s.dist, "builds/"+id+"/manifest.json")
	if err != nil {
		return manifest, nil, err
	}
	if err = decodeOutput(raw, &manifest); err != nil {
		return manifest, nil, err
	}
	if err = validateOutputManifest(manifest, true); err != nil || manifest.BuildID != id {
		return manifest, nil, fmt.Errorf("invalid owned generation manifest")
	}
	if err = validateOutputTree(s.dist, "builds/"+id, manifest, partial, false); err != nil {
		return manifest, nil, err
	}
	return manifest, raw, nil
}
func validateOutputTree(root *os.Root, base string, manifest OutputManifest, partial, staging bool) error {
	info, err := root.Lstat(base)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("generation root is not a real directory")
	}
	expected := map[string]bool{"manifest.json": true}
	dirs := map[string]bool{".": true}
	for name := range manifest.Files {
		expected[name] = true
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			dirs[parent] = true
		}
	}
	seen := map[string]bool{}
	err = fs.WalkDir(root.FS(), base, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel := strings.TrimPrefix(name, base+"/")
		if name == base {
			rel = "."
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in output tree: %s", name)
		}
		if entry.IsDir() {
			if !dirs[rel] {
				return fmt.Errorf("unknown output directory: %s", name)
			}
			return nil
		}
		if !entry.Type().IsRegular() || !expected[rel] {
			return fmt.Errorf("unknown or nonregular output file: %s", name)
		}
		seen[rel] = true
		if rel != "manifest.json" && !staging {
			data, err := readOutputRegular(root, name)
			if err != nil {
				return err
			}
			if hashBytes(data) != manifest.Files[rel] {
				return fmt.Errorf("modified output file: %s", name)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !partial {
		for name := range expected {
			if !seen[name] {
				return fmt.Errorf("missing output file: %s", name)
			}
		}
	}
	return nil
}
func syncOutputTree(root *os.Root, base string) error {
	return fs.WalkDir(root.FS(), base, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return syncOutputDir(root, name)
		}
		return nil
	})
}
func deleteOutputTree(root *os.Root, base string, manifest OutputManifest, staging bool) error {
	// Revalidate, then unlink only manifest-listed names. A concurrently added
	// unknown file is never collected as a deletion candidate; it prevents rmdir.
	if err := validateOutputTree(root, base, manifest, true, staging); err != nil {
		return err
	}
	dirs := map[string]bool{}
	for _, name := range sortedOutputKeys(manifest.Files) {
		full := base + "/" + name
		if _, err := readOutputRegular(root, full); err == nil {
			if err = root.Remove(full); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			dirs[parent] = true
		}
	}
	ordered := sortedOutputKeys(dirs)
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	for _, name := range ordered {
		if err := root.Remove(base + "/" + name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if _, err := readOutputRegular(root, base+"/manifest.json"); err == nil {
		if err = root.Remove(base + "/manifest.json"); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return root.Remove(base)
}

func (s *OutputStore) Recover() error {
	if err := s.ensureFresh(); err != nil {
		return err
	}
	if err := s.checkLayout(); err != nil {
		return err
	}
	raw, err := readOutputRegular(s.dist, "pending.json")
	if err == nil {
		var pending outputPending
		if err = decodeOutput(raw, &pending); err != nil {
			return err
		}
		if pending.SchemaVersion != 1 || !strings.HasPrefix(pending.Stage, "builds/.stage-") || len(strings.TrimPrefix(pending.Stage, "builds/.stage-")) != 32 {
			return fmt.Errorf("invalid staging reservation")
		}
		suffix := strings.TrimPrefix(pending.Stage, "builds/.stage-")
		if _, err = hex.DecodeString(suffix); err != nil {
			return err
		}
		if err = validateOutputManifest(pending.Manifest, true); err != nil {
			return err
		}
		if info, e := s.dist.Lstat(pending.Stage); e == nil {
			if !info.IsDir() {
				return fmt.Errorf("staging path is not a directory")
			}
			if err = validateOutputTree(s.dist, pending.Stage, pending.Manifest, true, true); err != nil {
				return err
			}
			if err = deleteOutputTree(s.dist, pending.Stage, pending.Manifest, true); err != nil {
				return err
			}
		} else if !os.IsNotExist(e) {
			return e
		}
		if err = s.dist.Remove("pending.json"); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	// These fixed metadata temporaries are explicitly owned by the owner schema.
	// They are never followed and never promoted after an interrupted write.
	for _, name := range []string{".current.tmp", ".pending.tmp"} {
		if _, err = readOutputRegular(s.dist, name); err == nil {
			if err = s.dist.Remove(name); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return syncOutputDir(s.dist, ".")
}

type OutputLease struct {
	Directory string
	Manifest  OutputManifest
	file      *os.File
}

func (l *OutputLease) Close() error { return l.file.Close() }

// File must be inherited by the generated child and retained until its exit.
// Inheritance preserves the shared kernel lease if the launching parent dies.
func (l *OutputLease) File() *os.File { return l.file }
func (s *OutputStore) AcquireCurrent() (*OutputLease, error) {
	if err := s.checkLayout(); err != nil {
		return nil, err
	}
	raw, err := readOutputRegular(s.dist, "current.json")
	if err != nil {
		return nil, err
	}
	var current outputCurrent
	if err = decodeOutput(raw, &current); err != nil || current.SchemaVersion != 1 {
		return nil, fmt.Errorf("invalid current manifest")
	}
	manifest, encoded, err := s.generation(current.BuildID, false)
	if err != nil {
		return nil, err
	}
	if hashBytes(encoded) != current.ManifestSHA256 {
		return nil, fmt.Errorf("current manifest digest mismatch")
	}
	file, err := outputOpen(s.dist, "builds/"+current.BuildID+"/manifest.json", os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	if err = outputLock(file, false); err != nil {
		file.Close()
		return nil, err
	}
	return &OutputLease{Directory: filepath.Join(s.Graph.Root.Root, "dist", "builds", current.BuildID), Manifest: manifest, file: file}, nil
}

// Prune removes only inactive, non-current manifest-owned generations. A clean
// build retains ownership metadata; callers can safely rebuild afterward.
func (s *OutputStore) Prune() error { return s.prune(false) }

// Clean unpublishes current, removes inactive owned generations, and keeps live leases.
func (s *OutputStore) Clean() error { return s.prune(true) }

func (s *OutputStore) prune(clean bool) error {
	if err := s.Recover(); err != nil {
		return err
	}
	currentID := ""
	if raw, err := readOutputRegular(s.dist, "current.json"); err == nil {
		var current outputCurrent
		if err = decodeOutput(raw, &current); err != nil {
			return err
		}
		_, encoded, e := s.generation(current.BuildID, false)
		if e != nil {
			return e
		}
		if current.SchemaVersion != 1 || hashBytes(encoded) != current.ManifestSHA256 {
			return fmt.Errorf("invalid current manifest identity")
		}
		currentID = current.BuildID
	} else if !os.IsNotExist(err) {
		return err
	}
	entries, err := outputEntries(s.dist, "builds")
	if err != nil {
		return err
	}
	type candidate struct {
		name     string
		file     *os.File
		manifest OutputManifest
	}
	var candidates []candidate
	defer func() {
		for _, c := range candidates {
			c.file.Close()
		}
	}()
	// Preflight every candidate before deleting anything. Unknown user files in
	// any generation cause a refusal, not a partially destructive cleanup.
	for _, entry := range entries {
		if !entry.IsDir() || !digestPattern.MatchString(entry.Name()) {
			return fmt.Errorf("unknown build directory %s", entry.Name())
		}
		if entry.Name() == currentID && !clean {
			continue
		}
		manifest, _, e := s.generation(entry.Name(), true)
		if e != nil {
			return e
		}
		file, e := outputOpen(s.dist, "builds/"+entry.Name()+"/manifest.json", os.O_RDONLY, 0)
		if e != nil {
			return e
		}
		if e = outputLock(file, true); e != nil {
			file.Close()
			if !outputLockBusy(e) {
				return e
			}
			continue
		}
		candidates = append(candidates, candidate{entry.Name(), file, manifest})
	}
	if clean && currentID != "" {
		if err = s.dist.Remove("current.json"); err != nil {
			return err
		}
		if err = syncOutputDir(s.dist, "."); err != nil {
			return err
		}
	}
	for _, c := range candidates {
		if err = deleteOutputTree(s.dist, "builds/"+c.name, c.manifest, false); err != nil {
			return err
		}
	}
	return syncOutputDir(s.dist, "builds")
}

func (s *OutputStore) ensureFresh() error {
	fresh, err := project.Load(s.projectRoot.Name())
	if err != nil {
		return fmt.Errorf("project inputs changed during build; retry from a fresh snapshot")
	}
	assets, err := outputAssetHashes(fresh)
	if err != nil || outputSnapshot(fresh, assets) != s.snapshot {
		return fmt.Errorf("project inputs changed during build; retry from a fresh snapshot")
	}
	return nil
}
