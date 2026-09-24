package project

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

type Graph struct {
	Root *Project
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
}
type Project struct {
	Key, ID, Root                string
	Lineage                      string // declared manifest lineage; empty means legacy edge-path identity
	Manifest                     Manifest
	ManifestSHA256, SourceSHA256 string
	FixturesSHA256               string
	Registry                     Registry
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
	return load(directory, nil)
}

// load shares Load's implementation with overlay substitution: when
// substitute reports bytes for a walked source path, those bytes parse
// instead of the file on disk.
func load(directory string, substitute func(real string) ([]byte, bool)) (*Graph, error) {
	root, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return nil, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	g := &Graph{Projects: map[string]*Project{}, Packages: map[string]*Package{}, Lock: Lock{Edges: map[string]LockEdge{}, Projects: map[string]LockEntry{}}}
	lockPath := filepath.Join(root, "can.lock.json")
	if _, err := os.Lstat(lockPath); err == nil {
		data, err := readConfined(root, "can.lock.json")
		if err != nil {
			return nil, err
		}
		g.Lock, err = ParseLock(data)
		if err != nil {
			return nil, fmt.Errorf("can.lock.json: %w", err)
		}
		g.LockSHA256 = Digest(data)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	loading := map[string]bool{}
	nodes := map[string]*Project{}
	lineages := map[string]*Project{}
	outputs := map[string]string{}
	var load func([]string, string, int) (*Project, error)
	load = func(edgePath []string, directory string, depth int) (*Project, error) {
		if depth > 256 {
			return nil, fmt.Errorf("dependency graph exceeds 256 levels")
		}
		if loading[directory] {
			return nil, fmt.Errorf("dependency manifest cycle at %s", directory)
		}
		// Identical real paths intern to one instance however many edges
		// reach them. The cycle check above runs first so a manifest
		// cycle still fails instead of resolving to a half-loaded node.
		if existing, ok := nodes[directory]; ok {
			return existing, nil
		}
		loading[directory] = true
		defer delete(loading, directory)
		data, err := readConfined(directory, "can.project.json")
		if err != nil {
			return nil, err
		}
		manifest, err := ParseManifest(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", directory, err)
		}
		registryData, err := readConfined(directory, manifest.ErrorRegistry)
		if err != nil {
			return nil, err
		}
		registry, err := ParseRegistry(registryData)
		if err != nil {
			return nil, fmt.Errorf("%s registry: %w", directory, err)
		}
		if manifest.Project != "" {
			if owner, exists := lineages[manifest.Project]; exists {
				return nil, fmt.Errorf("project lineage %q names divergent instances at %s and %s", manifest.Project, owner.Root, directory)
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
		project := &Project{Key: key, ID: identity, Root: directory, Lineage: manifest.Project, Manifest: manifest, ManifestSHA256: Digest(data), Registry: registry, Dependencies: map[string]*Project{}}
		g.Projects[key] = project
		nodes[directory] = project
		if manifest.Project != "" {
			lineages[manifest.Project] = project
		}
		project.CheckedAssets, err = Snapshot(directory, manifest.Assets)
		if err != nil {
			return nil, err
		}
		for i := range project.CheckedAssets {
			project.CheckedAssets[i].Project = key
		}
		for _, name := range sortedKeys(manifest.Dependencies) {
			depDir, err := ConfinedPath(directory, manifest.Dependencies[name], true)
			if err != nil {
				return nil, err
			}
			child := append(append([]string{}, edgePath...), name)
			dep, err := load(child, depDir, depth+1)
			if err != nil {
				return nil, err
			}
			project.Dependencies[name] = dep
		}
		if err := g.readSources(project, outputs, substitute); err != nil {
			return nil, err
		}
		if err := verifySourceRegistry(project); err != nil {
			return nil, err
		}
		if err := project.captureFixtures(); err != nil {
			return nil, err
		}
		return project, nil
	}
	g.Root, err = load(nil, root, 0)
	if err != nil {
		return nil, err
	}
	if err := g.verifyLock(); err != nil {
		return nil, err
	}
	if err := g.verifyRegistryGraph(); err != nil {
		return nil, err
	}
	return g, nil
}

func readConfined(root, name string) ([]byte, error) {
	file, err := ConfinedPath(root, name, false)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(file)
}

func (g *Graph) readSources(project *Project, outputs map[string]string, substitute func(real string) ([]byte, bool)) error {
	root, err := ConfinedPath(project.Root, project.Manifest.SourceRoot, true)
	if err != nil {
		return err
	}
	directories := map[string]bool{}
	files := map[string]bool{}
	packages := map[string]*Package{}
	var walk func(string, string, int) error
	walk = func(realDir, relative string, depth int) error {
		if depth > 256 {
			return fmt.Errorf("source tree exceeds 256 levels")
		}
		if directories[realDir] {
			return fmt.Errorf("source directory alias or symlink cycle at %s", relative)
		}
		directories[realDir] = true
		entries, err := os.ReadDir(realDir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			name := entry.Name()
			if !utf8.ValidString(name) {
				return fmt.Errorf("source path is not UTF-8")
			}
			logical := path.Join(relative, name)
			real, err := filepath.EvalSymlinks(filepath.Join(realDir, name))
			if err != nil {
				return err
			}
			if !Contains(project.Root, real) {
				return fmt.Errorf("source symlink %q escapes its manifest directory", logical)
			}
			info, err := os.Stat(real)
			if err != nil {
				return err
			}
			if info.IsDir() {
				if err := walk(real, logical, depth+1); err != nil {
					return err
				}
				continue
			}
			if !strings.HasSuffix(name, ".can") {
				continue
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("source %q is not a regular file", logical)
			}
			if files[real] {
				return fmt.Errorf("source file alias at %s", logical)
			}
			files[real] = true
			data, err := os.ReadFile(real)
			if err != nil {
				return err
			}
			if substitute != nil {
				if overlaid, ok := substitute(real); ok {
					data = overlaid
				}
			}
			file, err := source.New(real, string(data))
			if err != nil {
				return &SourceError{Path: real, Message: err.Error()}
			}
			parsed := syntax.Parse(file)
			if !parsed.OK() {
				return &SourceError{Path: real, File: file, Diagnostics: parsed.Diagnostics, Message: parsed.Diagnostics[0].Format(file)}
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
					return fmt.Errorf("package name %q occurs in more than one directory of %s", packageName, project.ID)
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
				return fmt.Errorf("source folder %s contains different package names", packageDir)
			}
			// Identity follows the exact logical path included in the source
			// digest. Canonical locations govern ownership/security only.
			id := pkg.ID + "/" + logical
			output := pkg.OutputDirectory + "/s-" + Digest([]byte("can-source-path-v1\x00"+id)) + ".ts"
			if err := claimOutput(outputs, output, id); err != nil {
				return err
			}
			src := &Source{Name: name, Path: real, RelativePath: logical, ID: id, OutputPath: output, Bytes: data, Syntax: parsed.File, Package: pkg}
			pkg.Sources = append(pkg.Sources, src)
			project.Sources = append(project.Sources, src)
		}
		return nil
	}
	if err := walk(root, "", 0); err != nil {
		return err
	}
	sort.Slice(project.Sources, func(i, j int) bool { return project.Sources[i].RelativePath < project.Sources[j].RelativePath })
	sort.Slice(project.Packages, func(i, j int) bool { return project.Packages[i].Name < project.Packages[j].Name })
	sources := make([]SourceBytes, len(project.Sources))
	for i, s := range project.Sources {
		sources[i] = SourceBytes{Path: s.RelativePath, Bytes: s.Bytes}
	}
	project.SourceSHA256, err = SourceDigest(sources)
	return err
}

func claimOutput(claims map[string]string, path, identity string) error {
	key := strings.ToLower(path)
	if owner, exists := claims[key]; exists && owner != identity {
		return fmt.Errorf("output path collision between %q and %q", owner, identity)
	}
	claims[key] = identity
	return nil
}

func verifySourceRegistry(project *Project) error {
	active := []ErrorAllocation{}
	for _, file := range project.Sources {
		for _, decl := range file.Syntax.Declarations {
			errDecl, ok := decl.(*syntax.ErrorDecl)
			if !ok {
				continue
			}
			id, err := strconv.ParseUint(errDecl.ID.Text, 0, 31)
			if err != nil || id < 1000000 {
				return fmt.Errorf("application source error ID out of range: %s", errDecl.ID.Text)
			}
			active = append(active, ErrorAllocation{ID: id, Kind: file.Package.Name + "::" + errDecl.Name.Text})
		}
	}
	sort.Slice(active, func(i, j int) bool { return active[i].ID < active[j].ID })
	if !reflect.DeepEqual(active, project.Registry.Active) {
		return fmt.Errorf("%s error registry differs from source declarations", project.ID)
	}
	return nil
}

func (g *Graph) verifyLock() error {
	visited := map[string]bool{}
	var visit func(parent *Project, edges map[string]LockEdge) error
	visit = func(parent *Project, edges map[string]LockEdge) error {
		if len(edges) != len(parent.Manifest.Dependencies) {
			return fmt.Errorf("dependency lock edges do not match the %s manifest", parent.ID)
		}
		for _, name := range sortedKeys(parent.Manifest.Dependencies) {
			edge, exists := edges[name]
			if !exists {
				return fmt.Errorf("missing dependency lock edge %q of %s", name, parent.ID)
			}
			if edge.Path != parent.Manifest.Dependencies[name] {
				return fmt.Errorf("dependency lock path mismatch for edge %q of %s", name, parent.ID)
			}
			child := parent.Dependencies[name]
			if child.ID != edge.Target {
				return fmt.Errorf("dependency lock target mismatch for edge %q of %s", name, parent.ID)
			}
			if visited[child.ID] {
				continue
			}
			visited[child.ID] = true
			entry, exists := g.Lock.Projects[child.ID]
			if !exists {
				return fmt.Errorf("missing dependency lock entry for %q", child.ID)
			}
			if entry.Lineage != child.Lineage {
				return fmt.Errorf("dependency lock lineage mismatch for %q", child.ID)
			}
			if entry.ManifestSHA256 != child.ManifestSHA256 || entry.SourceSHA256 != child.SourceSHA256 || entry.FixturesSHA256 != child.FixturesSHA256 {
				return fmt.Errorf("stale dependency digest for %q", child.ID)
			}
			if !reflect.DeepEqual(entry.ErrorRegistry, child.Registry) {
				return fmt.Errorf("dependency registry snapshot mismatch for %q", child.ID)
			}
			if err := visit(child, entry.Edges); err != nil {
				return err
			}
		}
		for _, name := range sortedKeys(edges) {
			if _, exists := parent.Manifest.Dependencies[name]; !exists {
				return fmt.Errorf("unused dependency lock edge %q of %s", name, parent.ID)
			}
		}
		return nil
	}
	if err := visit(g.Root, g.Lock.Edges); err != nil {
		return err
	}
	if len(visited) != len(g.Lock.Projects) {
		return fmt.Errorf("dependency lock has missing or unused entries")
	}
	for id := range g.Lock.Projects {
		if !visited[id] {
			return fmt.Errorf("unused dependency lock entry for %q", id)
		}
	}
	return nil
}

func (g *Graph) verifyRegistryGraph() error {
	owners := map[uint64]string{}
	for _, e := range catalogue.Builtin().Inventory().Errors {
		owners[uint64(e.ID)] = "catalogue"
	}
	for _, key := range sortedKeys(g.Projects) {
		project := g.Projects[key]
		claim := func(id uint64) error {
			if owner, exists := owners[id]; exists {
				return fmt.Errorf("error ID %d is claimed by %s and %s", id, owner, project.ID)
			}
			owners[id] = project.ID
			return nil
		}
		for _, e := range project.Registry.Active {
			if err := claim(e.ID); err != nil {
				return err
			}
		}
		for _, id := range project.Registry.Retired {
			if err := claim(id); err != nil {
				return err
			}
		}
	}
	return nil
}
