// Package catalogue owns the closed distribution inventory. Applications cannot
// load another inventory or register host implementations through this package.
package catalogue

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
)

//go:embed catalogue.json
var source []byte

type Parameter struct {
	Name       string `json:"name"`
	Constraint string `json:"constraint"`
}
type Field struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type Package struct {
	Name     string `json:"name"`
	Identity string `json:"identity"`
}
type TypeDecl struct {
	Name          string      `json:"name"`
	Identity      string      `json:"identity"`
	Kind          string      `json:"kind"`
	Parameters    []Parameter `json:"parameters"`
	Fields        []Field     `json:"fields"`
	Leaves        []string    `json:"leaves"`
	Projections   []Field     `json:"projections"`
	Constructible bool        `json:"constructible"`
}
type ErrorDecl struct {
	Name       string      `json:"name"`
	Identity   string      `json:"identity"`
	Parameters []Parameter `json:"parameters"`
	Fields     []Field     `json:"fields"`
}
type Callback struct {
	Name         string   `json:"name"`
	Inputs       []string `json:"inputs"`
	Result       string   `json:"result"`
	DeriveErrors bool     `json:"deriveErrors"`
	Emits        []string `json:"emits"`
}
type Lowering struct {
	Native  []string `json:"native"`
	Adapter string   `json:"adapter"`
	Task    string   `json:"task"`
}
type Operation struct {
	Name           string      `json:"name"`
	Identity       string      `json:"identity"`
	Kind           string      `json:"kind"`
	Receiver       string      `json:"receiver"`
	Parameters     []Parameter `json:"parameters"`
	Inputs         []Field     `json:"inputs"`
	StaticInputs   []string    `json:"staticInputs"`
	Result         string      `json:"result"`
	Callbacks      []Callback  `json:"callbacks"`
	Emits          []string    `json:"emits"`
	CallbackErrors []string    `json:"callbackErrors"`
	Lowering       Lowering    `json:"lowering"`
	Assertion      string      `json:"assertion"`
	Refs           []string    `json:"refs"`
}
type ConditionalBound struct {
	Condition string   `json:"condition"`
	Emits     []string `json:"emits"`
}
type NativeDeclaration struct {
	Name             string             `json:"name"`
	Emits            []string           `json:"emits"`
	ConditionalEmits []ConditionalBound `json:"conditionalEmits"`
	UnionBounds      []string           `json:"unionBounds"`
	Task             string             `json:"task"`
	Refs             []string           `json:"refs"`
	Assertion        string             `json:"assertion"`
}
type Inventory struct {
	SchemaVersion      int                 `json:"schemaVersion"`
	Revision           int                 `json:"revision"`
	TargetID           string              `json:"targetId"`
	Packages           []Package           `json:"packages"`
	Prelude            []string            `json:"prelude"`
	StandardFailures   []string            `json:"standardFailures"`
	Types              []TypeDecl          `json:"types"`
	Errors             []ErrorDecl         `json:"errors"`
	Operations         []Operation         `json:"operations"`
	NativeDeclarations []NativeDeclaration `json:"nativeDeclarations"`
}
type Catalogue struct {
	inventory  Inventory
	types      map[string]TypeDecl
	errors     map[string]ErrorDecl
	operations map[string]Operation
	native     map[string]NativeDeclaration
	reserved   map[string]bool
}

var builtin *Catalogue
var once sync.Once

func Builtin() *Catalogue {
	once.Do(func() {
		if SourceHash() != GeneratedSourceSHA256 {
			panic("stale generated catalogue; regenerate mirrors")
		}
		var err error
		builtin, err = load(source)
		if err != nil {
			panic("invalid built-in catalogue: " + err.Error())
		}
	})
	return builtin
}
func SourceHash() string { sum := sha256.Sum256(source); return hex.EncodeToString(sum[:]) }
func clone[T any](value T) T {
	data, _ := json.Marshal(value)
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		panic(err)
	}
	return result
}
func (c *Catalogue) Inventory() Inventory              { return clone(c.inventory) }
func (c *Catalogue) Type(name string) (TypeDecl, bool) { v, ok := c.types[name]; return clone(v), ok }
func (c *Catalogue) Error(name string) (ErrorDecl, bool) {
	v, ok := c.errors[name]
	return clone(v), ok
}
func (c *Catalogue) IsReservedPackage(name string) bool { return c.reserved[name] }
func (c *Catalogue) CheckProjectPackage(name string) error {
	if c.reserved[name] {
		return fmt.Errorf("catalogue package %q is distribution-owned", name)
	}
	return nil
}
func (c *Catalogue) CheckConstructor(name string) error {
	if t, ok := c.types[name]; ok {
		if !t.Constructible {
			return fmt.Errorf("%s has no public constructor", name)
		}
		return nil
	}
	if _, ok := c.errors[name]; ok {
		return nil
	}
	return fmt.Errorf("unknown catalogue constructor %q", name)
}
func (c *Catalogue) Operation(name, target string, revision int) (Operation, error) {
	if target != c.inventory.TargetID || revision != c.inventory.Revision {
		return Operation{}, fmt.Errorf("unsupported catalogue target or revision")
	}
	op, ok := c.operations[name]
	if !ok {
		return Operation{}, fmt.Errorf("unknown catalogue operation %q; host registration is closed", name)
	}
	return clone(op), nil
}

// load is deliberately private. Only embedded distribution data is loadable by
// production consumers; malformed-inventory tests exercise this same validator.
func load(data []byte) (*Catalogue, error) {
	if err := uniqueJSONKeys(data); err != nil {
		return nil, err
	}
	var inventory Inventory
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&inventory); err != nil {
		return nil, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("trailing catalogue JSON")
	}
	c := &Catalogue{inventory: inventory, types: map[string]TypeDecl{}, errors: map[string]ErrorDecl{}, operations: map[string]Operation{}, native: map[string]NativeDeclaration{}, reserved: map[string]bool{}}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}
func uniqueJSONKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var value func() error
	value = func() error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		if delim == '{' {
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("invalid object key")
				}
				if seen[name] {
					return fmt.Errorf("duplicate JSON key %q", name)
				}
				seen[name] = true
				if err := value(); err != nil {
					return err
				}
			}
		} else if delim == '[' {
			for d.More() {
				if err := value(); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("invalid JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	return nil
}

var identifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
var taskID = regexp.MustCompile(`^(I[0-9]{2}|LF[0-9]{2}|B1-[0-9]{2})$`)

func (c *Catalogue) identity(name string) (string, error) {
	if p, m, ok := strings.Cut(name, "::"); ok {
		if !c.reserved[p] || !identifier.MatchString(m) {
			return "", fmt.Errorf("unknown catalogue owner: %s", name)
		}
		return fmt.Sprintf("can.std.%s@%d::%s", p, c.inventory.Revision, m), nil
	}
	if p, m, ok := strings.Cut(name, "."); ok {
		if (p != "array" && p != "str" && p != "bytes") || !identifier.MatchString(m) {
			return "", fmt.Errorf("unknown intrinsic receiver: %s", name)
		}
		return fmt.Sprintf("can.intrinsic.%s@%d::%s", p, c.inventory.Revision, m), nil
	}
	for _, p := range c.inventory.Prelude {
		if p == name {
			return fmt.Sprintf("can.prelude@%d::%s", c.inventory.Revision, name), nil
		}
	}
	return "", fmt.Errorf("unknown prelude identity %q", name)
}
func (c *Catalogue) validate() error {
	inv := c.inventory
	if inv.SchemaVersion != 1 || inv.Revision < 1 || inv.TargetID == "" {
		return fmt.Errorf("unsupported catalogue header")
	}
	identities := map[string]bool{}
	names := map[string]bool{}
	claim := func(name, id string) error {
		want, err := c.identity(name)
		if err != nil {
			return err
		}
		if want != id {
			return fmt.Errorf("identity mismatch for %s", name)
		}
		if names[name] || identities[id] {
			return fmt.Errorf("duplicate catalogue identity %s", name)
		}
		names[name] = true
		identities[id] = true
		return nil
	}
	for _, p := range inv.Packages {
		if !identifier.MatchString(p.Name) || c.reserved[p.Name] || p.Identity != fmt.Sprintf("can.std.%s@%d", p.Name, inv.Revision) {
			return fmt.Errorf("invalid/duplicate reserved package %s", p.Name)
		}
		c.reserved[p.Name] = true
	}
	if err := uniqueStrings(inv.Prelude, "prelude"); err != nil {
		return err
	}
	if err := uniqueStrings(inv.StandardFailures, "standard failure"); err != nil {
		return err
	}
	for _, t := range inv.Types {
		if err := claim(t.Name, t.Identity); err != nil {
			return err
		}
		if t.Kind != "record" && t.Kind != "variant" && t.Kind != "opaque" {
			return fmt.Errorf("invalid type kind %s", t.Kind)
		}
		if t.Constructible != (t.Kind == "record") {
			return fmt.Errorf("constructor policy for %s", t.Name)
		}
		if t.Kind != "record" && len(t.Fields) > 0 || t.Kind != "variant" && len(t.Leaves) > 0 || t.Kind != "opaque" && len(t.Projections) > 0 || t.Kind == "variant" && len(t.Leaves) == 0 {
			return fmt.Errorf("invalid type representation %s", t.Name)
		}
		c.types[t.Name] = t
	}
	for _, e := range inv.Errors {
		if err := claim(e.Name, e.Identity); err != nil {
			return err
		}
		c.errors[e.Name] = e
	}
	for _, t := range inv.Types {
		ps, err := parameters(t.Parameters)
		if err != nil {
			return err
		}
		if err := c.checkFields(t.Fields, ps); err != nil {
			return fmt.Errorf("%s: %w", t.Name, err)
		}
		if err := c.checkFields(t.Projections, ps); err != nil {
			return err
		}
		if err := uniqueStrings(t.Leaves, "variant leaf"); err != nil {
			return err
		}
		for _, leaf := range t.Leaves {
			if err := c.checkType(leaf, ps, false); err != nil {
				return err
			}
			ref, _ := parseType(leaf)
			decl, ok := c.types[ref.name]
			_, isError := c.errors[ref.name]
			if !isError && (!ok || decl.Kind != "record") {
				return fmt.Errorf("catalogue variant leaf must be a record/error: %s", leaf)
			}
		}
	}
	for _, e := range inv.Errors {
		ps, err := parameters(e.Parameters)
		if err != nil {
			return err
		}
		if err := c.checkFields(e.Fields, ps); err != nil {
			return err
		}
	}
	for _, op := range inv.Operations {
		if err := claim(op.Name, op.Identity); err != nil {
			return err
		}
		if op.Kind != "function" && op.Kind != "method" && op.Kind != "property" || op.Kind == "function" && op.Receiver != "" || op.Kind != "function" && op.Receiver == "" {
			return fmt.Errorf("invalid operation kind/receiver %s", op.Name)
		}
		ps, err := parameters(op.Parameters)
		if err != nil {
			return err
		}
		if op.Receiver != "" {
			if err := c.checkType(op.Receiver, ps, false); err != nil {
				return err
			}
		}
		if err := c.checkType(op.Result, ps, true); err != nil {
			return err
		}
		callbacks := map[string]Callback{}
		for _, cb := range op.Callbacks {
			if callbacks[cb.Name].Name != "" || !identifier.MatchString(cb.Name) {
				return fmt.Errorf("duplicate/invalid callback")
			}
			callbacks[cb.Name] = cb
			for _, typ := range cb.Inputs {
				if err := c.checkType(typ, ps, false); err != nil {
					return err
				}
			}
			if err := c.checkType(cb.Result, ps, true); err != nil {
				return err
			}
			if err := c.checkBound(cb.Emits, ps); err != nil {
				return err
			}
			if cb.DeriveErrors && len(cb.Emits) > 0 {
				return fmt.Errorf("derived callback has fixed bound")
			}
		}
		used := map[string]bool{}
		for _, f := range op.Inputs {
			if !identifier.MatchString(f.Name) || used[f.Name] {
				return fmt.Errorf("duplicate/invalid input %s", f.Name)
			}
			used[f.Name] = true
			if strings.HasPrefix(f.Type, "$") {
				if callbacks[strings.TrimPrefix(f.Type, "$")].Name == "" {
					return fmt.Errorf("unknown callback input")
				}
			} else if err := c.checkType(f.Type, ps, false); err != nil {
				return err
			}
		}
		for name := range callbacks {
			count := 0
			for _, f := range op.Inputs {
				if f.Type == "$"+name {
					count++
				}
			}
			if count != 1 {
				return fmt.Errorf("callback must have exactly one input")
			}
		}
		if err := uniqueStrings(op.StaticInputs, "static input"); err != nil {
			return err
		}
		for _, name := range op.StaticInputs {
			found := false
			for _, f := range op.Inputs {
				if f.Name == name && f.Type == "str" {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("invalid static input %s", name)
			}
		}
		if err := c.checkBound(op.Emits, ps); err != nil {
			return err
		}
		if err := uniqueStrings(op.CallbackErrors, "callback bound"); err != nil {
			return err
		}
		for _, name := range op.CallbackErrors {
			if !callbacks[name].DeriveErrors {
				return fmt.Errorf("unknown/non-derived callback bound %s", name)
			}
		}
		for name, cb := range callbacks {
			if cb.DeriveErrors && !contains(op.CallbackErrors, name) {
				return fmt.Errorf("missing callback bound %s", name)
			}
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" || !taskID.MatchString(op.Lowering.Task) || len(op.Refs) == 0 {
			return fmt.Errorf("missing native recipe/traceability: %s", op.Name)
		}
		if op.Assertion != "real" && op.Assertion != "supplied" && op.Assertion != "scoped" {
			return fmt.Errorf("invalid assertion recipe %s", op.Name)
		}
		c.operations[op.Name] = op
	}
	for _, n := range inv.NativeDeclarations {
		if c.native[n.Name].Name != "" {
			return fmt.Errorf("duplicate native declaration mode")
		}
		switch n.Name {
		case "noul", "choice", "record_choice", "score", "record_score", "choice_arm", "judge", "fetch_body", "fetch_envelope", "llm":
		default:
			return fmt.Errorf("user protocol/kernel registration is forbidden: %s", n.Name)
		}
		if err := c.checkBound(n.Emits, nil); err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, b := range n.ConditionalEmits {
			if b.Condition != "authenticated" && b.Condition != "uses_codec" || seen[b.Condition] {
				return fmt.Errorf("invalid conditional error bound")
			}
			seen[b.Condition] = true
			if err := c.checkBound(b.Emits, nil); err != nil {
				return err
			}
		}
		if err := uniqueStrings(n.UnionBounds, "union bound"); err != nil {
			return err
		}
		for _, u := range n.UnionBounds {
			if u != "questions" && u != "handlers" && u != "continuation" && u != "body" {
				return fmt.Errorf("unknown native bound source")
			}
		}
		if !taskID.MatchString(n.Task) || len(n.Refs) == 0 || (n.Assertion != "real" && n.Assertion != "raw-provider") {
			return fmt.Errorf("invalid native profile recipe")
		}
		c.native[n.Name] = n
	}
	for _, name := range inv.Prelude {
		if !names[name] {
			return fmt.Errorf("missing prelude declaration %s", name)
		}
	}
	return nil
}
func uniqueStrings(xs []string, kind string) error {
	seen := map[string]bool{}
	for _, x := range xs {
		if x == "" || seen[x] {
			return fmt.Errorf("empty/duplicate %s %q", kind, x)
		}
		seen[x] = true
	}
	return nil
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
func parameters(ps []Parameter) (map[string]bool, error) {
	out := map[string]bool{}
	for _, p := range ps {
		if !identifier.MatchString(p.Name) || out[p.Name] {
			return nil, fmt.Errorf("invalid generic parameter")
		}
		switch p.Constraint {
		case "data", "map_key", "sort_key", "wire", "form", "sql_parameters", "sql_row", "failure_variant":
		default:
			return nil, fmt.Errorf("unknown constraint %s", p.Constraint)
		}
		out[p.Name] = true
	}
	return out, nil
}
func (c *Catalogue) checkFields(fs []Field, ps map[string]bool) error {
	seen := map[string]bool{}
	for _, f := range fs {
		if !identifier.MatchString(f.Name) || seen[f.Name] {
			return fmt.Errorf("duplicate/invalid field %s", f.Name)
		}
		seen[f.Name] = true
		if err := c.checkType(f.Type, ps, false); err != nil {
			return err
		}
	}
	return nil
}
func (c *Catalogue) checkBound(bound []string, ps map[string]bool) error {
	if err := uniqueStrings(bound, "error bound"); err != nil {
		return err
	}
	for _, name := range bound {
		ref, err := parseType(name)
		if err != nil {
			return err
		}
		if _, ok := c.errors[ref.name]; !ok {
			return fmt.Errorf("unallocated domain error %s", name)
		}
		if err := c.checkType(name, ps, false); err != nil {
			return err
		}
	}
	return nil
}
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
