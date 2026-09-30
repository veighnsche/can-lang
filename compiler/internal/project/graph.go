package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

type Graph struct {
	Errors []error
	Inputs map[string][]byte
	Root   *Project
	// Projects is keyed by canonical node identity; the empty key is the
	// root. Dependency edge names are parent-local: the same edge name in
	// different parents may reach different instances.
	Projects map[string]*Project
	// Packages is keyed by canonical package identity
	// (<project-ID>/<package>). One package name may occur in many
	// instances, but only once per instance.
	Packages map[string]*Package
	Lock     Lock
	// LockSHA256 binds the exact can.lock.json bytes into verification
	// identity. It is empty when the root project carries no lock file.
	LockSHA256 string
	// Enumerated records every source-directory enumeration in walk order.
	// It is empty for graphs whose sources were never walked.
	Enumerated []EnumeratedDir
}
type Project struct {
	Key, ID, Root                string
	Lineage                      string // declared manifest lineage; empty means legacy edge-path identity
	Manifest                     Manifest
	ManifestSHA256, SourceSHA256 string
	FixturesSHA256               string
	SourceIncomplete             bool
	FixturesIncomplete           bool
	Registry                     Registry
	RegistryError                error
	FixtureErrors                map[string]error
	Dependencies                 map[string]*Project
	Packages                     []*Package
	Sources                      []*Source
	CheckedAssets                []Asset
	CheckedFixtures              []Fixture
}
type Package struct {
	Name, ID, Directory, OutputDirectory string
	Owner                                *Project
	Sources                              []*Source
}
type Source struct {
	Name, Path, RelativePath, ID, OutputPath string
	Bytes                                    []byte
	Syntax                                   *syntax.File
	Package                                  *Package
}

// SourceError reports a source file that could not be decoded or parsed. It
// carries the structured diagnostics alongside the exact message the CLI has
// always printed, so editor bridges can convert spans without changing CLI
// output by a single byte.
type SourceError struct {
	Path        string
	File        *source.File
	Diagnostics []syntax.Diagnostic
	Message     string
}

func (e *SourceError) Error() string { return e.Message }

// Load reads exactly the named local project's manifest graph. It does not
// search parent directories, download dependencies, execute hooks, or write locks.
func Load(directory string) (*Graph, error) {
	return load(context.Background(), directory, nil)
}

// load shares Load's implementation with overlay substitution: when
// substitute reports bytes for a walked source path, those bytes parse
// instead of the file on disk.
func load(ctx context.Context, directory string, overlay *Overlay) (*Graph, error) {
	var substitute func(string) ([]byte, bool)
	if overlay != nil {
		substitute = overlay.bytesFor
	}
	root, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return nil, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	g := &Graph{Inputs: map[string][]byte{}, Projects: map[string]*Project{}, Packages: map[string]*Package{}, Lock: Lock{Edges: map[string]LockEdge{}, Projects: map[string]LockEntry{}}}
	read := func(directory, name string) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := NormalizePath(name); err != nil {
			return nil, err
		}
		target := filepath.Join(directory, filepath.FromSlash(name))
		real, err := ConfinedPath(directory, name, false)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			if _, statErr := os.Lstat(target); statErr == nil {
				return nil, err
			} // dangling symlink
			real, err = canonicalPath(target)
			if err != nil {
				return nil, err
			}
			if !Contains(directory, real) {
				return nil, &GraphError{Kind: KindEscape, Path: filepath.Join(directory, "can.project.json"), Msg: fmt.Sprintf("path %q escapes its manifest directory", name)}
			}
		}
		g.Inputs[real] = nil
		if substitute != nil {
			if data, ok := substitute(real); ok {
				g.Inputs[real] = data
				return data, nil
			}
		}
		data, err := os.ReadFile(real)
		g.Inputs[real] = data
		return data, err
	}
	lockPath := filepath.Join(root, "can.lock.json")
	_, lockErr := os.Lstat(lockPath)
	lockOverlay := false
	if overlay != nil {
		_, lockOverlay = overlay.Get(lockPath)
	}
	lockValid := true
	g.Inputs[lockPath] = nil
	if lockErr == nil || lockOverlay {
		data, err := read(root, "can.lock.json")
		if err != nil {
			return nil, err
		}
		g.Lock, err = ParseLock(data)
		if err != nil {
			lockValid = false
			g.Errors = append(g.Errors, configError(lockPath, data, err))
		}
		g.LockSHA256 = Digest(data)
	} else if !os.IsNotExist(lockErr) {
		return g, lockErr
	}
	loading := map[string]bool{}
	nodes := map[string]*Project{}
	lineages := map[string]*Project{}
	outputs := map[string]string{}
	var load func([]string, string, int) (*Project, error)
	load = func(edgePath []string, directory string, depth int) (*Project, error) {
		if depth > 256 {
			return nil, &GraphError{Kind: KindTooDeep, Path: filepath.Join(directory, "can.project.json"), Msg: "dependency graph exceeds 256 levels"}
		}
		if loading[directory] {
			return nil, &GraphError{Kind: KindCycle, Path: filepath.Join(directory, "can.project.json"), Msg: fmt.Sprintf("dependency manifest cycle at %s", directory)}
		}
		// Identical real paths intern to one instance however many edges
		// reach them. The cycle check above runs first so a manifest
		// cycle still fails instead of resolving to a half-loaded node.
		if existing, ok := nodes[directory]; ok {
			return existing, nil
		}
		loading[directory] = true
		defer delete(loading, directory)
		data, err := read(directory, "can.project.json")
		if err != nil {
			return nil, err
		}
		manifest, err := ParseManifest(data)
		if err != nil {
			manifestErr := configError(filepath.Join(directory, "can.project.json"), data, err)
			if manifest.SourceRoot == "" || manifest.Invalid["source_root"] != nil || manifest.Invalid["project"] != nil || manifest.Invalid["syntax"] != nil {
				return nil, manifestErr
			}
			g.Errors = append(g.Errors, manifestErr)
		}
		var registryData []byte
		registryErr := manifest.Invalid["error_registry"]
		if registryErr == nil {
			registryData, registryErr = read(directory, manifest.ErrorRegistry)
		}
		var registry Registry
		if registryErr != nil {
			registryErr = configError(filepath.Join(directory, "can.project.json"), data, jsonFieldError(data, registryErr, "error_registry"))
		} else {
			registry, registryErr = ParseRegistry(registryData)
			if registryErr != nil {
				registryErr = configError(filepath.Join(directory, manifest.ErrorRegistry), registryData, registryErr)
			}
		}
		if registryErr != nil {
			g.Errors = append(g.Errors, registryErr)
		}

		if manifest.Project != "" {
			if owner, exists := lineages[manifest.Project]; exists {
				return nil, &GraphError{Kind: KindLineage, Path: filepath.Join(directory, "can.project.json"), Msg: fmt.Sprintf("project lineage %q names divergent instances at %s and %s", manifest.Project, owner.Root, directory)}
			}
		}
		identity := "can.project.root"
		key := ""
		if edgePath != nil {
			key = "can.project.dependency/" + strings.Join(edgePath, "/")
			identity = key
			if manifest.Project != "" {
				identity = "can.project.lineage/" + manifest.Project
				key = identity
			}
		}
		project := &Project{Key: key, ID: identity, Root: directory, Lineage: manifest.Project, Manifest: manifest, ManifestSHA256: Digest(data), Registry: registry, RegistryError: registryErr, Dependencies: map[string]*Project{}}
		g.Projects[key] = project
		nodes[directory] = project
		if manifest.Project != "" {
			lineages[manifest.Project] = project
		}
		for _, name := range sortedKeys(manifest.Assets) {
			assets, assetErr := Snapshot(directory, map[string]string{name: manifest.Assets[name]})
			assetPath := filepath.Join(directory, manifest.Assets[name])
			g.Inputs[assetPath] = nil
			if assetErr != nil {
				g.Errors = append(g.Errors, configError(filepath.Join(directory, "can.project.json"), data, jsonFieldError(data, assetErr, "assets", name)))
				continue
			}
			for _, asset := range assets {
				g.Inputs[assetPath] = asset.Bytes
			}
			project.CheckedAssets = append(project.CheckedAssets, assets...)
		}
		for i := range project.CheckedAssets {
			project.CheckedAssets[i].Project = key
		}
		for _, name := range sortedKeys(manifest.Dependencies) {
			depDir, err := ConfinedPath(directory, manifest.Dependencies[name], true)
			if err != nil {
				g.Inputs[filepath.Join(directory, manifest.Dependencies[name], "can.project.json")] = nil
				g.Errors = append(g.Errors, &DependencyError{Edge: name, Dir: filepath.Join(directory, manifest.Dependencies[name]), Err: configError(filepath.Join(directory, "can.project.json"), data, jsonFieldError(data, err, "dependencies", name))})
				continue
			}
			child := append(append([]string{}, edgePath...), name)
			dep, err := load(child, depDir, depth+1)
			if err != nil {
				g.Errors = append(g.Errors, &DependencyError{Edge: name, Dir: depDir, Err: err})
				continue
			}
			project.Dependencies[name] = dep
		}
		if err := g.readSources(ctx, project, outputs, overlay); err != nil {
			g.Errors = append(g.Errors, err)
		}
		// Registry agreement needs the complete declaration inventory; raw
		// fixture capture remains independent of unrelated source failures.
		syntaxComplete := !project.SourceIncomplete
		for _, file := range project.Sources {
			if file.Syntax == nil || len(file.Syntax.Invalid) != 0 {
				syntaxComplete = false
			}
		}
		if registryErr == nil && syntaxComplete {
			if err := verifySourceRegistry(project); err != nil {
				typed := contentOrRaw(KindRegistryMismatch, filepath.Join(project.Root, project.Manifest.ErrorRegistry), err)
				g.Errors = append(g.Errors, typed)
				project.RegistryError = typed
			}
		}
		project.FixturesIncomplete = !syntaxComplete
		if err := project.captureFixtures(g.Inputs); err != nil {
			project.FixturesIncomplete = true
			g.Errors = append(g.Errors, err)
		}

		return project, nil
	}
	g.Root, err = load(nil, root, 0)
	if err != nil {
		g.Errors = append(g.Errors, err)
		return g, errors.Join(g.Errors...)
	}
	if lockValid {
		if err := g.verifyLock(); err != nil {
			g.Errors = append(g.Errors, err)
		}
	}
	return g, errors.Join(g.Errors...)
}

func readConfined(root, name string) ([]byte, error) {
	file, err := ConfinedPath(root, name, false)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(file)
}

func (g *Graph) readSources(ctx context.Context, project *Project, outputs map[string]string, overlay *Overlay) error {
	var substitute func(string) ([]byte, bool)
	if overlay != nil {
		substitute = overlay.bytesFor
	}
	root, err := ConfinedPath(project.Root, project.Manifest.SourceRoot, true)
	if err != nil {
		project.SourceIncomplete = true
		return err
	}
	var sourceBytes []SourceBytes
	directories := map[string]bool{}
	files := map[string]bool{}
	packages := map[string]*Package{}
	addSource := func(real, logical string, data []byte) error {
		g.Inputs[real] = data
		sourceBytes = append(sourceBytes, SourceBytes{Path: logical, Bytes: data})
		file, err := source.New(real, string(data))
		if err != nil {
			return &SourceError{Path: real, Message: err.Error()}
		}
		parsed := syntax.Parse(file)
		if !parsed.OK() {
			g.Errors = append(g.Errors, &SourceError{Path: real, File: file, Diagnostics: parsed.Diagnostics, Message: parsed.Diagnostics[0].Format(file)})
		}
		if parsed.File.Header.Name.Text == "" {
			project.Sources = append(project.Sources, &Source{Name: path.Base(logical), Path: real, RelativePath: logical, Bytes: data, Syntax: parsed.File})
			return nil
		}
		packageName := parsed.File.Header.Name.Text
		// A source symlink cannot move an internal package into a public
		// directory. Package ownership follows the canonical source folder.
		packageDir := filepath.Dir(real)
		pkg := packages[packageDir]
		if pkg == nil {
			if err := catalogue.Builtin().CheckProjectPackage(packageName); err != nil {
				return err
			}
			id := project.ID + "/" + packageName
			if existing := g.Packages[id]; existing != nil {
				return &GraphError{Kind: KindPackageConflict, Path: real, Msg: fmt.Sprintf("package name %q occurs in more than one directory of %s", packageName, project.ID)}
			}
			output := "packages/p-" + Digest([]byte("can-package-path-v1\x00"+id))
			if err := claimOutput(outputs, output, id); err != nil {
				return err
			}
			pkg = &Package{Name: packageName, ID: id, Directory: packageDir, OutputDirectory: output, Owner: project}
			packages[packageDir] = pkg
			project.Packages = append(project.Packages, pkg)
			g.Packages[id] = pkg
		} else if pkg.Name != packageName {
			return &GraphError{Kind: KindPackageConflict, Path: real, Msg: fmt.Sprintf("source folder %s contains different package names", packageDir)}
		}
		// Identity follows the exact logical path included in the source
		// digest. Canonical locations govern ownership/security only.
		id := pkg.ID + "/" + logical
		output := pkg.OutputDirectory + "/s-" + Digest([]byte("can-source-path-v1\x00"+id)) + ".ts"
		if err := claimOutput(outputs, output, id); err != nil {
			return err
		}
		src := &Source{Name: path.Base(logical), Path: real, RelativePath: logical, ID: id, OutputPath: output, Bytes: data, Syntax: parsed.File, Package: pkg}
		pkg.Sources = append(pkg.Sources, src)
		project.Sources = append(project.Sources, src)
		return nil
	}
	var walkProblems []error
	reportPath := func(file string, err error) {
		project.SourceIncomplete = true
		g.Inputs[file] = nil
		walkProblems = append(walkProblems, fmt.Errorf("source %s: %w", file, err))
	}
	var walk func(string, string, int) error
	walk = func(realDir, relative string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if depth > 256 {
			return &GraphError{Kind: KindTooDeep, Path: realDir, Msg: "source tree exceeds 256 levels"}
		}
		if directories[realDir] {
			return &GraphError{Kind: KindCycle, Path: realDir, Msg: fmt.Sprintf("source directory alias or symlink cycle at %s", relative)}
		}
		directories[realDir] = true
		entries, err := os.ReadDir(realDir)
		if err != nil {
			return err
		}
		g.Enumerated = append(g.Enumerated, EnumeratedDir{Dir: realDir, Entries: len(entries)})
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			name := entry.Name()
			if !utf8.ValidString(name) {
				reportPath(filepath.Join(realDir, name), &GraphError{Kind: KindUTF8, Path: filepath.Join(realDir, name), Msg: "source path is not UTF-8"})
				continue
			}
			logical := path.Join(relative, name)
			entryPath := filepath.Join(realDir, name)
			real, err := filepath.EvalSymlinks(entryPath)
			if err != nil {
				reportPath(entryPath, err)
				continue
			}
			if !Contains(project.Root, real) {
				reportPath(entryPath, &GraphError{Kind: KindEscape, Path: entryPath, Msg: fmt.Sprintf("source symlink %q escapes its manifest directory", logical)})
				continue
			}
			info, err := os.Stat(real)
			if err != nil {
				reportPath(real, err)
				continue
			}
			if info.IsDir() {
				if err := walk(real, logical, depth+1); err != nil {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					reportPath(real, err)
				}
				continue
			}
			if !strings.HasSuffix(name, ".can") {
				continue
			}
			if !info.Mode().IsRegular() {
				reportPath(real, fmt.Errorf("source %q is not a regular file", logical))
				continue
			}
			if files[real] {
				reportPath(entryPath, fmt.Errorf("source file alias at %s", logical))
				continue
			}
			files[real] = true
			data, err := os.ReadFile(real)
			if err != nil {
				reportPath(real, err)
				continue
			}
			if substitute != nil {
				if overlaid, ok := substitute(real); ok {
					data = overlaid
				}
			}
			if err := addSource(real, logical, data); err != nil {
				g.Errors = append(g.Errors, err)
			}
		}
		return nil
	}
	if err := walk(root, "", 0); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		reportPath(root, err)
	}
	if overlay != nil {
		entries := overlay.Snapshot()
		for _, real := range sortedKeys(entries) {
			if files[real] || !strings.HasSuffix(real, ".can") || !Contains(root, real) {
				continue
			}
			logical, err := filepath.Rel(root, real)
			if err != nil {
				return err
			}
			if err := addSource(real, filepath.ToSlash(logical), []byte(entries[real].Text)); err != nil {
				g.Errors = append(g.Errors, err)
			}
		}
	}

	sort.Slice(project.Sources, func(i, j int) bool { return project.Sources[i].RelativePath < project.Sources[j].RelativePath })
	sort.Slice(project.Packages, func(i, j int) bool { return project.Packages[i].Name < project.Packages[j].Name })
	project.SourceSHA256, err = SourceDigest(sourceBytes)
	if err != nil {
		project.SourceIncomplete = true
		walkProblems = append(walkProblems, err)
	}
	return errors.Join(walkProblems...)
}

func claimOutput(claims map[string]string, path, identity string) error {
	key := strings.ToLower(path)
	if owner, exists := claims[key]; exists && owner != identity {
		return &GraphError{Kind: KindOutputCollision, Msg: fmt.Sprintf("output path collision between %q and %q", owner, identity)}
	}
	claims[key] = identity
	return nil
}

func verifySourceRegistry(project *Project) error {
	if err := project.Registry.Validate(); err != nil {
		return err
	}
	active := []string{}
	for _, file := range project.Sources {
		for _, decl := range file.Syntax.Declarations {
			errDecl, ok := decl.(*syntax.ErrorDecl)
			if !ok {
				continue
			}
			active = append(active, file.Package.Name+"::"+errDecl.Name.Text)
		}
	}
	sort.Strings(active)
	if !reflect.DeepEqual(active, project.Registry.Active) {
		return fmt.Errorf("%s error registry differs from source declarations", project.ID)
	}
	live := map[string]bool{}
	for _, kind := range active {
		live[kind] = true
	}
	for _, name := range project.Registry.Retired {
		if live[name] {
			return fmt.Errorf("%s reuses retired error %q", project.ID, name)
		}
	}
	return nil
}

// ResolveErrorReport attributes an archived terminal-report identity to
// its owning project and current active kind, following that owner's
// supplied predecessor chains. Catalogue identities resolve to
// themselves with a nil owner; retired names without a chain do not
// resolve.
func (g *Graph) ResolveErrorReport(identity string) (owner *Project, kind string, ok bool) {
	declaration, found := strings.CutPrefix(identity, ReportIdentityVersion+":")
	if !found || declaration == "" {
		return nil, "", false
	}
	if catalogueName, isCatalogue := catalogueErrorName(declaration); isCatalogue {
		return nil, catalogueName, true
	}
	for _, key := range sortedKeys(g.Projects) {
		project := g.Projects[key]
		rest, found := strings.CutPrefix(declaration, project.ID+"/")
		if !found || !qualifiedKind(rest) {
			continue
		}
		current, resolved := project.Registry.ResolveKind(rest)
		if !resolved {
			return nil, "", false
		}
		return project, current, true
	}
	return nil, "", false
}

// catalogueErrorName maps a catalogue declaration identity back to its
// short error name. Distribution errors carry no predecessor chains.
func catalogueErrorName(declaration string) (string, bool) {
	for _, error := range catalogue.Builtin().Inventory().Errors {
		if error.Identity == declaration {
			return error.Name, true
		}
	}
	return "", false
}

func (g *Graph) verifyLock() error {
	visited := map[string]bool{}
	complete := true
	var problems []error
	var visit func(*Project, map[string]LockEdge)
	visit = func(parent *Project, edges map[string]LockEdge) {
		if parent.Manifest.Invalid["dependencies"] != nil {
			complete = false
		}
		for _, name := range sortedKeys(parent.Manifest.Dependencies) {
			edge, exists := edges[name]
			manifestPath := filepath.Join(parent.Root, "can.project.json")
			if !exists {
				problems = append(problems, &GraphError{Kind: KindLockMismatch, Path: manifestPath, Msg: fmt.Sprintf("missing dependency lock edge %q of %s", name, parent.ID)})
				complete = false
				continue
			}
			if edge.Path != parent.Manifest.Dependencies[name] {
				problems = append(problems, &GraphError{Kind: KindLockMismatch, Path: manifestPath, Msg: fmt.Sprintf("dependency lock path mismatch for edge %q of %s", name, parent.ID)})
			}
			child := parent.Dependencies[name]
			if child == nil {
				complete = false
				continue
			}
			if child.ID != edge.Target {
				problems = append(problems, &GraphError{Kind: KindLockMismatch, Path: manifestPath, Msg: fmt.Sprintf("dependency lock target mismatch for edge %q of %s", name, parent.ID)})
			}
			if visited[child.ID] {
				continue
			}
			visited[child.ID] = true
			entry, exists := g.Lock.Projects[child.ID]
			if !exists {
				problems = append(problems, &GraphError{Kind: KindLockMismatch, Path: filepath.Join(parent.Root, "can.project.json"), Msg: fmt.Sprintf("missing dependency lock entry for %q", child.ID)})
				complete = false
				continue
			}
			if entry.Lineage != child.Lineage {
				problems = append(problems, &GraphError{Kind: KindLockMismatch, Path: filepath.Join(child.Root, "can.project.json"), Msg: fmt.Sprintf("dependency lock lineage mismatch for %q", child.ID)})
			}
			if entry.ManifestSHA256 != child.ManifestSHA256 || !child.SourceIncomplete && entry.SourceSHA256 != child.SourceSHA256 || !child.FixturesIncomplete && entry.FixturesSHA256 != child.FixturesSHA256 {
				problems = append(problems, &GraphError{Kind: KindLockMismatch, Path: filepath.Join(child.Root, "can.project.json"), Msg: fmt.Sprintf("stale dependency digest for %q", child.ID)})
			}
			if child.RegistryError == nil && !reflect.DeepEqual(entry.ErrorRegistry, child.Registry) {
				problems = append(problems, &GraphError{Kind: KindLockMismatch, Path: filepath.Join(child.Root, "can.project.json"), Msg: fmt.Sprintf("dependency registry snapshot mismatch for %q", child.ID)})
			}
			visit(child, entry.Edges)
		}
		for _, name := range sortedKeys(edges) {
			if _, exists := parent.Manifest.Dependencies[name]; !exists && parent.Manifest.Invalid["dependencies"] == nil {
				problems = append(problems, &GraphError{Kind: KindLockMismatch, Path: filepath.Join(parent.Root, "can.project.json"), Msg: fmt.Sprintf("unused dependency lock edge %q of %s", name, parent.ID)})
			}
		}
	}
	visit(g.Root, g.Lock.Edges)
	if complete {
		for _, id := range sortedKeys(g.Lock.Projects) {
			if !visited[id] {
				problems = append(problems, &GraphError{Kind: KindLockMismatch, Path: filepath.Join(g.Root.Root, "can.project.json"), Msg: fmt.Sprintf("unused dependency lock entry for %q", id)})
			}
		}
	}
	return errors.Join(problems...)
}
