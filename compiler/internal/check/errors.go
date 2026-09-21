package check

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type ErrorDeclaration struct {
	Identity   string `json:"identity"`
	Name       string `json:"name"`
	ID         uint64 `json:"id"`
	Parameters int    `json:"parameters"`
}
type ErrorRegistry struct{ declarations map[string]ErrorDeclaration }

// ErrorDeclarations consumes the manifest/lock/source agreement established by
// project.Load, and binds each allocation to the resolver's declaration identity.
// Rechecking the allocation/source association here prevents a registry entry
// from being used as permission to emit a different declaration with that ID.
func ErrorDeclarations(world *resolve.World) (*ErrorRegistry, error) {
	registry := &ErrorRegistry{declarations: map[string]ErrorDeclaration{}}
	ids := map[uint64]string{}
	add := func(d ErrorDeclaration) error {
		if prior := ids[d.ID]; prior != "" {
			return fmt.Errorf("duplicate error allocation %d", d.ID)
		}
		if _, exists := registry.declarations[d.Identity]; exists {
			return fmt.Errorf("duplicate error declaration %s", d.Identity)
		}
		ids[d.ID] = d.Identity
		registry.declarations[d.Identity] = d
		return nil
	}
	for _, d := range catalogue.Builtin().Inventory().Errors {
		if err := add(ErrorDeclaration{d.Identity, d.Name, uint64(d.ID), len(d.Parameters)}); err != nil {
			return nil, err
		}
	}
	var owners []string
	for key := range world.Graph.Projects {
		owners = append(owners, key)
	}
	sort.Strings(owners)
	for _, key := range owners {
		owner := world.Graph.Projects[key]
		for _, retired := range owner.Registry.Retired {
			if retired < 1000000 || retired > 2147483647 || ids[retired] != "" {
				return nil, fmt.Errorf("invalid or reused retired error ID %d", retired)
			}
			ids[retired] = "retired"
		}
		for _, allocation := range owner.Registry.Active {
			if allocation.ID < 1000000 || allocation.ID > 2147483647 {
				return nil, fmt.Errorf("project error allocation outside application range")
			}
			parts := strings.Split(allocation.Kind, "::")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid allocated error name")
			}
			pkg := world.Packages[parts[0]]
			if pkg == nil || pkg.Source == nil || pkg.Source.Owner != owner {
				return nil, fmt.Errorf("error allocation has wrong nominal owner")
			}
			symbol := pkg.Scope.Symbols[parts[1]]
			if symbol == nil || symbol.Kind != resolve.Error {
				return nil, fmt.Errorf("error allocation lacks source declaration")
			}
			declaration, ok := symbol.Declaration.(*syntax.ErrorDecl)
			if !ok {
				return nil, fmt.Errorf("error allocation lacks error AST")
			}
			id, err := strconv.ParseUint(declaration.ID.Text, 10, 64)
			if err != nil || id != allocation.ID {
				return nil, fmt.Errorf("error source/registry ID mismatch")
			}
			if err = add(ErrorDeclaration{symbol.ID, allocation.Kind, allocation.ID, len(symbol.Parameters)}); err != nil {
				return nil, err
			}
		}
	}
	for _, pkg := range world.Packages {
		if pkg.Source == nil {
			continue
		}
		for _, symbol := range pkg.Scope.Symbols {
			if symbol.Kind == resolve.Error {
				if _, ok := registry.declarations[symbol.ID]; !ok {
					return nil, fmt.Errorf("source error is unallocated")
				}
			}
		}
	}
	return registry, nil
}
func (r *ErrorRegistry) Declarations() []ErrorDeclaration {
	out := make([]ErrorDeclaration, 0, len(r.declarations))
	for _, d := range r.declarations {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type ConcreteError struct {
	Declaration  ErrorDeclaration `json:"declaration"`
	TypeIdentity string           `json:"typeIdentity"`
	Arguments    []string         `json:"arguments"`
	Type         *types.Type      `json:"-"`
}

func (r *ErrorRegistry) Concrete(typ *types.Type) (ConcreteError, error) {
	if typ == nil || !types.Equal(typ, typ) || typ.Kind() != types.Error {
		return ConcreteError{}, fmt.Errorf("domain bound requires a checked nominal error, not a standard failure")
	}
	declaration, ok := r.declarations[typ.Declaration()]
	if !ok {
		return ConcreteError{}, fmt.Errorf("unallocated nominal error identity")
	}
	args := typ.Arguments()
	if len(args) != declaration.Parameters {
		return ConcreteError{}, fmt.Errorf("error specialization arity mismatch")
	}
	out := ConcreteError{Declaration: declaration, TypeIdentity: typ.Identity(), Arguments: []string{}, Type: typ}
	for _, arg := range args {
		out.Arguments = append(out.Arguments, arg.Identity())
	}
	return out, nil
}

type ErrorBound struct{ entries []ConcreteError }

func (r *ErrorRegistry) Bound(errors []*types.Type) (ErrorBound, error) {
	out := ErrorBound{}
	seen := map[string]bool{}
	for _, typ := range errors {
		entry, err := r.Concrete(typ)
		if err != nil {
			return ErrorBound{}, err
		}
		if seen[entry.TypeIdentity] {
			return ErrorBound{}, fmt.Errorf("duplicate concrete error in authored emits")
		}
		seen[entry.TypeIdentity] = true
		out.entries = append(out.entries, entry)
	}
	sort.Slice(out.entries, func(i, j int) bool { return out.entries[i].TypeIdentity < out.entries[j].TypeIdentity })
	return out, nil
}
func (b ErrorBound) Entries() []ConcreteError {
	out := append([]ConcreteError(nil), b.entries...)
	for i := range out {
		out[i].Arguments = append([]string(nil), out[i].Arguments...)
	}
	return out
}

// CheckEscaping compares exact specializations; it does not widen payloads or
// infer an error set. Region checking supplies only the errors that escape its
// explicit handlers, leaving standard failures in their separate channel.
func (b ErrorBound) CheckEscaping(actual ErrorBound) error {
	allowed := map[string]ErrorDeclaration{}
	for _, entry := range b.entries {
		allowed[entry.TypeIdentity] = entry.Declaration
	}
	for _, entry := range actual.entries {
		if allowed[entry.TypeIdentity] != entry.Declaration {
			return fmt.Errorf("undeclared escaping domain error %s (%s)", entry.Declaration.Name, entry.TypeIdentity)
		}
	}
	return nil
}
func (b ErrorBound) ResolveBareArm(declarationIdentity string) (ConcreteError, error) {
	var found *ConcreteError
	for _, entry := range b.entries {
		if entry.Declaration.Identity == declarationIdentity {
			if found != nil {
				return ConcreteError{}, fmt.Errorf("ambiguous generic error arm; normalize concrete payloads through a named wrapper")
			}
			copy := entry
			found = &copy
		}
	}
	if found == nil {
		return ConcreteError{}, fmt.Errorf("error arm is outside matched bound")
	}
	found.Arguments = append([]string(nil), found.Arguments...)
	return *found, nil
}
