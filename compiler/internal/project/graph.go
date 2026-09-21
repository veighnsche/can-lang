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
	Root     *Project
	Projects map[string]*Project // empty key is the root; dependency keys are global
	Packages map[string]*Package // flat source names are globally unique
	Lock     Lock
}
type Project struct {
	Key, ID, Root                string
	Manifest                     Manifest
	ManifestSHA256, SourceSHA256 string
	Registry                     Registry
	Dependencies                 map[string]*Project
	Packages                     []*Package
	Sources                      []*Source
	CheckedAssets                []Asset
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

// Load reads exactly the named local project's manifest graph. It does not
// search parent directories, download dependencies, execute hooks, or write locks.
func Load(directory string) (*Graph, error) {
	root, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return nil, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	g := &Graph{Projects: map[string]*Project{}, Packages: map[string]*Package{}, Lock: Lock{Dependencies: map[string]LockEntry{}}}
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
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	loading := map[string]bool{}
	owners := map[string]string{}
	outputs := map[string]string{}
	var load func(string, string, int) (*Project, error)
	load = func(key, directory string, depth int) (*Project, error) {
		if depth > 256 {
			return nil, fmt.Errorf("dependency graph exceeds 256 levels")
		}
		if loading[directory] {
			return nil, fmt.Errorf("dependency manifest cycle at %s", directory)
		}
		if existing, ok := g.Projects[key]; ok {
			if existing.Root != directory {
				return nil, fmt.Errorf("dependency key %q resolves to different directories", key)
			}
			return existing, nil
		}
		if owner, exists := owners[directory]; exists && owner != key {
			return nil, fmt.Errorf("different dependency keys name one real directory: %q and %q", owner, key)
		}
		loading[directory] = true
		defer delete(loading, directory)
		owners[directory] = key
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
		identity := "can.project.root"
		if key != "" {
			identity = "can.project.dependency/" + key
		}
		project := &Project{Key: key, ID: identity, Root: directory, Manifest: manifest, ManifestSHA256: Digest(data), Registry: registry, Dependencies: map[string]*Project{}}
		g.Projects[key] = project
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
			dep, err := load(name, depDir, depth+1)
			if err != nil {
				return nil, err
			}
			project.Dependencies[name] = dep
		}
		if err := g.readSources(project, outputs); err != nil {
			return nil, err
		}
		if err := verifySourceRegistry(project); err != nil {
			return nil, err
		}
		return project, nil
	}
	g.Root, err = load("", root, 0)
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

func (g *Graph) readSources(project *Project, outputs map[string]string) error {
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
			file, err := source.New(real, string(data))
			if err != nil {
				return err
			}
			parsed := syntax.Parse(file)
			if !parsed.OK() {
				return fmt.Errorf("%s", parsed.Diagnostics[0].Format(file))
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
				if existing := g.Packages[packageName]; existing != nil {
					return fmt.Errorf("package name %q occurs in more than one directory", packageName)
				}
				id := project.ID + "/" + packageName
				output := "packages/p-" + Digest([]byte("can-package-path-v1\x00"+id))
				if err := claimOutput(outputs, output, id); err != nil {
					return err
				}
				pkg = &Package{Name: packageName, ID: id, Directory: packageDir, OutputDirectory: output, Owner: project}
				packages[packageDir] = pkg
				project.Packages = append(project.Packages, pkg)
				g.Packages[packageName] = pkg
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
	if len(g.Lock.Dependencies) != len(g.Projects)-1 {
		return fmt.Errorf("dependency lock has missing or unused entries")
	}
	for _, key := range sortedKeys(g.Projects) {
		if key == "" {
			continue
		}
		project := g.Projects[key]
		entry, exists := g.Lock.Dependencies[key]
		if !exists {
			return fmt.Errorf("missing dependency lock entry %q", key)
		}
		real, err := ConfinedPath(g.Root.Root, entry.Path, true)
		if err != nil {
			return err
		}
		if real != project.Root {
			return fmt.Errorf("dependency lock path mismatch for %q", key)
		}
		if entry.ManifestSHA256 != project.ManifestSHA256 || entry.SourceSHA256 != project.SourceSHA256 {
			return fmt.Errorf("stale dependency digest for %q", key)
		}
		if !reflect.DeepEqual(entry.ErrorRegistry, project.Registry) {
			return fmt.Errorf("dependency registry snapshot mismatch for %q", key)
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
