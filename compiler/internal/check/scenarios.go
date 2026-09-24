package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// scenarioLinks resolves every link on an assertion root to canonical
// scenario identities. Links never shadow: an unqualified link must
// resolve to exactly one reachable scenario, so adding a same-named
// scenario elsewhere cannot silently retarget the root.
func scenarioLinks(file *resolve.File, row syntax.Assertion) ([]string, error) {
	if len(row.Links) == 0 {
		return nil, nil
	}
	defFile := file.Source.Syntax.Source.Name()
	seen := map[string]bool{}
	var links []string
	for _, link := range row.Links {
		symbol, err := resolveScenarioLink(file, link)
		if err != nil {
			return nil, source.Locate(defFile, link.Span, fmt.Errorf("assertion %s: %w", row.Name.Text, err))
		}
		if seen[symbol.ID] {
			return nil, source.Locate(defFile, link.Span, fmt.Errorf("assertion %s: duplicate scenario link %q", row.Name.Text, symbol.ID))
		}
		seen[symbol.ID] = true
		links = append(links, symbol.ID)
	}
	return links, nil
}

// resolveScenarioLink binds one link target to its scenario symbol.
// Qualified links follow ordinary import visibility: cross-package
// targets must be exported. Unqualified links search the owning
// package and every imported package without shadowing.
func resolveScenarioLink(file *resolve.File, link syntax.QualifiedName) (*resolve.Symbol, error) {
	spelling := link.Name
	if link.Package != "" {
		spelling = link.Package + "::" + link.Name
		symbol, err := file.Lookup(nil, link, resolve.ScenarioUse)
		if err == nil {
			return symbol, nil
		}
		if pkg := file.Imports[link.Package]; pkg != nil && pkg != file.Package {
			if candidate := pkg.Scope.Symbols[link.Name]; candidate != nil && candidate.Kind == resolve.Scenario && !candidate.Public {
				return nil, fmt.Errorf("scenario link %q names a scenario that %s does not export", spelling, pkg.ID)
			}
		}
		return nil, fmt.Errorf("stale scenario link %q: %w", spelling, err)
	}
	var candidates []*resolve.Symbol
	if symbol := file.Package.Scope.Symbols[link.Name]; symbol != nil && symbol.Kind == resolve.Scenario {
		candidates = append(candidates, symbol)
	}
	aliases := make([]string, 0, len(file.Imports))
	for alias := range file.Imports {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		pkg := file.Imports[alias]
		if pkg == file.Package {
			continue
		}
		symbol := pkg.Scope.Symbols[link.Name]
		if symbol != nil && symbol.Kind == resolve.Scenario && symbol.Public {
			candidates = append(candidates, symbol)
		}
	}
	switch len(candidates) {
	case 0:
		return nil, fmt.Errorf("stale scenario link %q: no reachable scenario is exported under that name", spelling)
	case 1:
		return candidates[0], nil
	default:
		ids := make([]string, len(candidates))
		for i, candidate := range candidates {
			ids[i] = candidate.ID
		}
		sort.Strings(ids)
		return nil, fmt.Errorf("ambiguous scenario link %q: candidates %s; qualify the link", spelling, strings.Join(ids, ", "))
	}
}
