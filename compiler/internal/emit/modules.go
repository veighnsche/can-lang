package emit

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

type ModuleImport struct {
	Target   string // canonical generation-relative module path, not authored spelling
	TypeOnly bool
	Names    []ImportName
}
type ImportName struct{ Exported, Local string }
type Module struct {
	Path    string
	Imports []ModuleImport
	// Body is produced by the checked IR emitters, never read from authored TS.
	Body string
}

var jsBinding = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

// Modules resolves structured import edges once and renders exact relative ESM
// specifiers. All modules refer to one generation-private runtime location;
// no helper state is duplicated per source module.
func Modules(modules []Module, dependencies ...ir.Artifact) ([]ir.Artifact, error) {
	paths := map[string]bool{}
	for _, artifact := range dependencies {
		if paths[artifact.Path] {
			return nil, fmt.Errorf("duplicate private artifact")
		}
		paths[artifact.Path] = true
	}
	for _, m := range modules {
		if paths[m.Path] || !strings.HasSuffix(m.Path, ".ts") {
			return nil, fmt.Errorf("duplicate or invalid module path")
		}
		paths[m.Path] = true
	}
	ordered := append([]Module(nil), modules...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	result := append([]ir.Artifact{}, dependencies...)
	for _, m := range ordered {
		var code strings.Builder
		edges := map[string]bool{}
		bindings := map[string]bool{}
		for _, edge := range m.Imports {
			if !paths[edge.Target] {
				return nil, fmt.Errorf("missing emitted module %s", edge.Target)
			}
			relative, err := filepath.Rel(filepath.Dir(m.Path), edge.Target)
			if err != nil {
				return nil, err
			}
			relative = filepath.ToSlash(relative)
			if !strings.HasPrefix(relative, ".") {
				relative = "./" + relative
			}
			names := make([]string, len(edge.Names))
			for i, name := range edge.Names {
				if !jsBinding.MatchString(name.Exported) || !jsBinding.MatchString(name.Local) || bindings[name.Local] {
					return nil, fmt.Errorf("invalid or conflicting imported binding")
				}
				bindings[name.Local] = true
				names[i] = name.Exported + " as " + name.Local
			}
			if len(names) == 0 {
				if edge.TypeOnly {
					return nil, fmt.Errorf("type-only import needs named types")
				}
				fmt.Fprintf(&code, "import %s;\n", quote(relative))
			} else {
				prefix := "import "
				if edge.TypeOnly {
					prefix += "type "
				}
				fmt.Fprintf(&code, "%s{ %s } from %s;\n", prefix, strings.Join(names, ", "), quote(relative))
			}
			edges[relative] = true
		}
		code.WriteString(m.Body)
		if !strings.HasSuffix(m.Body, "\n") {
			code.WriteByte('\n')
		}
		imports := make([]string, 0, len(edges))
		for name := range edges {
			imports = append(imports, name)
		}
		sort.Strings(imports)
		result = append(result, ir.Artifact{Path: m.Path, Bytes: []byte(code.String()), Imports: imports})
	}
	return result, nil
}
