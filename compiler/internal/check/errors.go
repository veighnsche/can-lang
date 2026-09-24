package check

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
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
			pkg := world.Packages[owner.ID+"/"+parts[0]]
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
			// Match project.Load's approved integer prefixes and C9's ID range.
			id, err := strconv.ParseUint(declaration.ID.Text, 0, 31)
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

type ErrorBound struct {
	entries []ConcreteError
	// open permits any escaping domain error. Wrapper policy arms are
	// checked against it while their calculated bound is still unknown;
	// the region still accumulates every observed escape for the bound.
	open bool
}

// OpenBound is the checking bound for wrapper policy arms. It must never
// escape into a published signature or an arm-resolution set.
func OpenBound() ErrorBound { return ErrorBound{open: true} }

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
	if b.open {
		return nil
	}
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

// errExactSpecialization marks ambiguity failures that list the exact
// specializations the author must choose between. Callers classify with
// errors.Is; the rendered message is unchanged.
var errExactSpecialization = errors.New("write one of the exact specializations")

// isExactSpecialization reports whether err asks the author to choose one
// exact specialization. It keeps errors.Is out of callers whose error-set
// parameters shadow the errors package.
func isExactSpecialization(err error) bool { return errors.Is(err, errExactSpecialization) }

func (b ErrorBound) ResolveBareArm(declarationIdentity string) (ConcreteError, error) {
	var found *ConcreteError
	for _, entry := range b.entries {
		if entry.Declaration.Identity == declarationIdentity {
			if found != nil {
				return ConcreteError{}, fmt.Errorf("ambiguous error head %q; %w: %s", found.Declaration.Name, errExactSpecialization, strings.Join(b.specializations(declarationIdentity), ", "))
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

// ResolveExactArm matches one complete nominal specialization. It is not a
// wildcard, a subtype test, or an implicit union over the bound.
func (b ErrorBound) ResolveExactArm(identity string) (ConcreteError, error) {
	for _, entry := range b.entries {
		if entry.TypeIdentity == identity {
			copy := entry
			copy.Arguments = append([]string(nil), copy.Arguments...)
			return copy, nil
		}
	}
	return ConcreteError{}, fmt.Errorf("error arm %s is outside matched bound", identity)
}

// stampCode attaches a stable diagnostic code to an error whose located span
// lacks one. Errors that already carry a code keep it, so choke points can
// classify failures without disturbing deeper, more precise obligations.
func stampCode(err error, code string) error {
	if err == nil {
		return nil
	}
	if located, ok := source.AsLocated(err); ok && located.Code == "" {
		located.Code = code
	}
	return err
}

// displayAlternatives renders one exact specialization per type: the short
// source-like spelling plus its exact identity. When short forms collide,
// the colliding entries keep their full declarations so every alternative
// stays distinct.
func displayAlternatives(alternatives []*types.Type) []string {
	out := make([]string, 0, len(alternatives))
	names := make([]string, 0, len(alternatives))
	for _, typ := range alternatives {
		names = append(names, displayType(typ, false))
	}
	seen := map[string]int{}
	for _, name := range names {
		seen[name]++
	}
	for i, typ := range alternatives {
		name := names[i]
		if seen[name] > 1 {
			name = displayType(typ, true)
		}
		out = append(out, name+" ("+typ.Identity()+")")
	}
	return out
}

func displayType(typ *types.Type, full bool) string {
	name := typ.Declaration()
	if name == "" {
		name = typ.Identity()
	} else if !full {
		if i := strings.LastIndex(name, "::"); i >= 0 {
			name = name[i+2:]
		}
	}
	args := typ.Arguments()
	if len(args) == 0 {
		return name
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, displayType(arg, full))
	}
	return name + "<" + strings.Join(parts, ", ") + ">"
}

func (b ErrorBound) specializations(declarationIdentity string) []string {
	var matches []*types.Type
	for _, entry := range b.entries {
		if entry.Declaration.Identity == declarationIdentity {
			matches = append(matches, entry.Type)
		}
	}
	out := displayAlternatives(matches)
	sort.Strings(out)
	return out
}
