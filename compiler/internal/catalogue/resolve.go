package catalogue

import (
	"fmt"
	"sort"
	"strings"
)

type ErrorIdentity struct {
	Name          string   `json:"name"`
	Identity      string   `json:"identity"`
	ID            int      `json:"id"`
	TypeArguments []string `json:"typeArguments"`
}
type CallbackContract struct {
	Inputs []string
	Result string
	Emits  []ErrorIdentity
}
type Specialization struct {
	Operation Operation
	Emits     []ErrorIdentity
}

func (c *Catalogue) ErrorIdentity(name string, arguments []string) (ErrorIdentity, error) {
	d, ok := c.errors[name]
	if !ok {
		return ErrorIdentity{}, fmt.Errorf("unallocated domain error %q", name)
	}
	if len(arguments) != len(d.Parameters) {
		return ErrorIdentity{}, fmt.Errorf("error specialization arity: %s", name)
	}
	canonical := make([]string, len(arguments))
	for i, a := range arguments {
		r, err := parseConcreteType(a)
		if err != nil || r.name == "void" {
			return ErrorIdentity{}, fmt.Errorf("invalid error type argument")
		}
		canonical[i] = r.String()
	}
	return ErrorIdentity{Name: d.Name, Identity: d.Identity, ID: d.ID, TypeArguments: canonical}, nil
}
func (c *Catalogue) validateError(e ErrorIdentity) error {
	if d, ok := c.errors[e.Name]; ok {
		if d.ID != e.ID || d.Identity != e.Identity || len(d.Parameters) != len(e.TypeArguments) {
			return fmt.Errorf("mismatched catalogue error identity %s", e.Name)
		}
	} else {
		if e.ID < 1000000 || int64(e.ID) > 2147483647 || !strings.Contains(e.Name, "::") || e.Identity == "" || strings.HasPrefix(e.Identity, "can.std.") || strings.HasPrefix(e.Identity, "can.prelude") {
			return fmt.Errorf("unallocated reserved error ID or invalid project identity: %d", e.ID)
		}
	}
	for _, arg := range e.TypeArguments {
		if _, err := parseConcreteType(arg); err != nil {
			return err
		}
	}
	return nil
}
func (c *Catalogue) bound(names []string, args map[string]string) ([]ErrorIdentity, error) {
	out := []ErrorIdentity{}
	for _, name := range names {
		r, err := parseType(name)
		if err != nil {
			return nil, err
		}
		r = r.substitute(args)
		xs := make([]string, len(r.arguments))
		for i, a := range r.arguments {
			xs[i] = a.String()
		}
		e, err := c.ErrorIdentity(r.name, xs)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}
func (c *Catalogue) union(groups ...[]ErrorIdentity) ([]ErrorIdentity, error) {
	seen := map[string]ErrorIdentity{}
	ids := map[int]string{}
	kinds := map[string]ErrorIdentity{}
	for _, group := range groups {
		for _, e := range group {
			if err := c.validateError(e); err != nil {
				return nil, err
			}
			if previous, ok := ids[e.ID]; ok && previous != e.Identity {
				return nil, fmt.Errorf("duplicate domain error ID %d", e.ID)
			}
			if previous, ok := kinds[e.Identity]; ok && (previous.Name != e.Name || previous.ID != e.ID) {
				return nil, fmt.Errorf("conflicting domain error kind")
			}
			ids[e.ID] = e.Identity
			kinds[e.Identity] = e
			key := e.Identity + "<" + strings.Join(e.TypeArguments, ",") + ">"
			if old, ok := seen[key]; ok && (old.ID != e.ID || old.Name != e.Name) {
				return nil, fmt.Errorf("conflicting error identity")
			}
			seen[key] = e
		}
	}
	keys := sortedKeys(seen)
	out := make([]ErrorIdentity, 0, len(keys))
	for _, key := range keys {
		out = append(out, seen[key])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Resolve specializes only an already admitted catalogue operation. Generic
// data arguments and actual callback contracts are supplied by the checker;
// finite callback errors are derived here without authored error-set variables.
func (c *Catalogue) Resolve(name, target string, revision int, args map[string]string, callbacks map[string]CallbackContract, project TypeAdmission) (Specialization, error) {
	op, err := c.Operation(name, target, revision)
	if err != nil {
		return Specialization{}, err
	}
	if len(args) != len(op.Parameters) {
		return Specialization{}, fmt.Errorf("generic argument arity for %s", name)
	}
	canonical := map[string]string{}
	for name, text := range args {
		typ, err := parseConcreteType(text)
		if err != nil {
			return Specialization{}, err
		}
		canonical[name] = typ.String()
	}
	args = canonical
	for _, p := range op.Parameters {
		if !c.admits(args[p.Name], p.Constraint, project) {
			return Specialization{}, fmt.Errorf("%s requires %s for %s", name, p.Constraint, p.Name)
		}
	}
	if len(callbacks) != len(op.Callbacks) {
		return Specialization{}, fmt.Errorf("callback arity for %s", name)
	}
	bound, err := c.bound(op.Emits, args)
	if err != nil {
		return Specialization{}, err
	}
	for i, cb := range op.Callbacks {
		actual, ok := callbacks[cb.Name]
		if !ok || len(actual.Inputs) != len(cb.Inputs) {
			return Specialization{}, fmt.Errorf("callback input arity for %s", cb.Name)
		}
		for j, typ := range cb.Inputs {
			expected := substitute(typ, args)
			input, parseErr := parseConcreteType(actual.Inputs[j])
			if parseErr != nil || expected != input.String() {
				return Specialization{}, fmt.Errorf("callback input mismatch")
			}
			op.Callbacks[i].Inputs[j] = expected
		}
		expected := substitute(cb.Result, args)
		result, parseErr := parseConcreteType(actual.Result)
		if parseErr != nil || result.String() != expected {
			return Specialization{}, fmt.Errorf("callback result mismatch")
		}
		op.Callbacks[i].Result = expected
		if cb.DeriveErrors {
			bound, err = c.union(bound, actual.Emits)
			if err != nil {
				return Specialization{}, err
			}
		} else {
			allowed, err := c.bound(cb.Emits, args)
			if err != nil {
				return Specialization{}, err
			}
			for _, e := range actual.Emits {
				found := false
				for _, a := range allowed {
					if e.Identity == a.Identity && e.ID == a.ID && strings.Join(e.TypeArguments, ",") == strings.Join(a.TypeArguments, ",") {
						found = true
					}
				}
				if !found {
					return Specialization{}, fmt.Errorf("callback exceeds fixed error bound")
				}
			}
		}
	}
	if op.Receiver != "" {
		op.Receiver = substitute(op.Receiver, args)
	}
	op.Result = substitute(op.Result, args)
	for i, f := range op.Inputs {
		if !strings.HasPrefix(f.Type, "$") {
			op.Inputs[i].Type = substitute(f.Type, args)
		}
	}
	bound, err = c.union(bound)
	if err != nil {
		return Specialization{}, err
	}
	return Specialization{Operation: op, Emits: bound}, nil
}

// RequiredNativeBound computes the intrinsic lower bound for authored native
// declarations. Their complete authored bound is checked by the native checker.
func (c *Catalogue) RequiredNativeBound(mode string, conditions map[string]bool, groups map[string][]ErrorIdentity) ([]ErrorIdentity, error) {
	n, ok := c.native[mode]
	if !ok {
		return nil, fmt.Errorf("unknown native mode; protocol registration is closed")
	}
	allowed := map[string]bool{}
	for _, cond := range n.ConditionalEmits {
		allowed[cond.Condition] = true
	}
	for key := range conditions {
		if !allowed[key] {
			return nil, fmt.Errorf("unknown native condition %s", key)
		}
	}
	for key := range groups {
		if !contains(n.UnionBounds, key) {
			return nil, fmt.Errorf("unknown native bound source %s", key)
		}
	}
	result, err := c.bound(n.Emits, nil)
	if err != nil {
		return nil, err
	}
	for _, cond := range n.ConditionalEmits {
		if conditions[cond.Condition] {
			extra, err := c.bound(cond.Emits, nil)
			if err != nil {
				return nil, err
			}
			result, err = c.union(result, extra)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, key := range n.UnionBounds {
		result, err = c.union(result, groups[key])
		if err != nil {
			return nil, err
		}
	}
	return c.union(result)
}
