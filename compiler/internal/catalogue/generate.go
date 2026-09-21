package catalogue

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

//go:embed runtime-header.txt
var runtimeHeader string

//go:embed runtime-footer.txt
var runtimeFooter string

// Generate derives all checked-in views from catalogue.json. Check mode performs
// no writes and fails if a mirror or reference document is missing/stale.
func Generate(root string, check bool) error {
	c, err := load(source)
	if err != nil {
		return err
	}
	outputs, err := c.generatedFiles()
	if err != nil {
		return err
	}
	for _, name := range sortedKeys(outputs) {
		path := filepath.Join(root, filepath.FromSlash(name))
		data := outputs[name]
		if check {
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(actual, data) {
				return fmt.Errorf("stale catalogue mirror %s; run go run ./compiler/internal/catalogue/cmd/cataloguegen", name)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
func goName(name string) string {
	var out strings.Builder
	for _, part := range strings.FieldsFunc(name, func(r rune) bool { return r == ':' || r == '.' || r == '_' }) {
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		out.WriteString(string(runes))
	}
	return out.String()
}
func (c *Catalogue) generatedFiles() (map[string][]byte, error) {
	var goCode strings.Builder
	fmt.Fprintf(&goCode, "// Code generated from catalogue.json; DO NOT EDIT.\npackage catalogue\n\nconst GeneratedSourceSHA256 = %q\nconst GeneratedRevision = %d\nconst GeneratedTargetID = %q\n", SourceHash(), c.inventory.Revision, c.inventory.TargetID)
	for _, t := range c.inventory.Types {
		fmt.Fprintf(&goCode, "const Type%s = %q\n", goName(t.Name), t.Name)
	}
	for _, e := range c.inventory.Errors {
		fmt.Fprintf(&goCode, "const Error%sID = %d\nconst Error%sName = %q\nconst Error%sIdentity = %q\n", goName(e.Name), e.ID, goName(e.Name), e.Name, goName(e.Name), e.Identity)
	}
	for _, op := range c.inventory.Operations {
		fmt.Fprintf(&goCode, "const Op%s = %q\n", goName(op.Name), op.Name)
	}
	goBytes, err := format.Source([]byte(goCode.String()))
	if err != nil {
		return nil, err
	}
	var ts strings.Builder
	ts.WriteString(runtimeHeader)
	fmt.Fprintf(&ts, "export const catalogueSHA256 = %q;\nexport const catalogue = freeze(%s as const);\n", SourceHash(), strings.TrimSpace(string(source)))
	shapes, err := c.runtimeShapes()
	if err != nil {
		return nil, err
	}
	shapeJSON, err := json.MarshalIndent(shapes, "", "  ")
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&ts, "export const catalogueTypeShapes = freeze(%s as const);\n", shapeJSON)
	ts.WriteString(runtimeFooter)
	var doc strings.Builder
	fmt.Fprintf(&doc, "# Closed distribution catalogue\n\nGenerated from compiler/internal/catalogue/catalogue.json; do not edit this mirror.\nRevision: **%d**. Target: %s. Source SHA-256: %s.\n\nThis is the complete approved descriptor inventory, not a claim that every\nruntime adapter is implemented. Each native recipe names its implementation\ntask. No user host protocol, kernel, opaque representation or catalogue package\nregistration is available. Standard failures stay outside domain emits.\n\nRegenerate with go run ./compiler/internal/catalogue/cmd/cataloguegen; verify\nwith the same command plus --check. Go tests also reject stale mirrors.\n\n", c.inventory.Revision, c.inventory.TargetID, SourceHash())
	doc.WriteString("## Packages\n\n")
	for _, p := range c.inventory.Packages {
		fmt.Fprintf(&doc, "- %s → %s\n", p.Name, p.Identity)
	}
	doc.WriteString("\n## Types\n\n| Name | Kind | Parameters | Fields or leaves | Constructor |\n| --- | --- | --- | --- | --- |\n")
	for _, t := range c.inventory.Types {
		shape := fieldText(t.Fields)
		if len(t.Leaves) > 0 {
			shape = strings.Join(t.Leaves, ", ")
		}
		if len(t.Projections) > 0 {
			shape = "read-only: " + fieldText(t.Projections)
		}
		fmt.Fprintf(&doc, "| %s | %s | %s | %s | %t |\n", t.Name, t.Kind, parameterText(t.Parameters), shape, t.Constructible)
	}
	doc.WriteString("\n## Domain errors\n\n| ID | Kind | Parameters | Ordered payload |\n| --- | --- | --- | --- |\n")
	for _, e := range c.inventory.Errors {
		fmt.Fprintf(&doc, "| %d | %s | %s | %s |\n", e.ID, e.Name, parameterText(e.Parameters), fieldText(e.Fields))
	}
	doc.WriteString("\n## Operations\n\nCallbacks have exact ordered input/result types. Derived means the bound is\nthe actual callback's finite domain error set; it is not authored effect syntax.\nReal assertions run computation; supplied requires a boundary fixture;\nscoped combines real adapter computation with fixture-owned opaque handles or\ncallbacks. Later assertion work must enforce those rules before side effects.\n\n| Operation | Receiver; inputs → result | Domain bound | Callback contracts | Native mapping | Adapter contract | Assertion | Task / references |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, op := range c.inventory.Operations {
		signature := fieldText(op.Inputs) + " → " + op.Result
		if op.Receiver != "" {
			signature = "receiver " + op.Receiver + "; " + signature
		}
		if len(op.Parameters) > 0 {
			signature = parameterText(op.Parameters) + "; " + signature
		}
		if len(op.StaticInputs) > 0 {
			signature += "; static " + strings.Join(op.StaticInputs, ", ")
		}
		bound := "[" + strings.Join(op.Emits, ", ") + "]"
		for _, cb := range op.CallbackErrors {
			bound += " + " + cb + " errors"
		}
		callbacks := []string{}
		for _, cb := range op.Callbacks {
			errors := "[" + strings.Join(cb.Emits, ", ") + "]"
			if cb.DeriveErrors {
				errors = "derived"
			}
			callbacks = append(callbacks, cb.Name+"("+strings.Join(cb.Inputs, ",")+") → "+cb.Result+" emits "+errors)
		}
		fmt.Fprintf(&doc, "| %s | %s | %s | %s | %s | %s | %s | %s / %s |\n", op.Name, signature, bound, strings.Join(callbacks, "; "), strings.Join(op.Lowering.Native, ", "), op.Lowering.Adapter, op.Assertion, op.Lowering.Task, strings.Join(op.Refs, ","))
	}
	doc.WriteString("\n## Native declaration profiles\n\nThese are the required intrinsic bounds, before checking the complete authored\nemits declaration. Conditional modes are never narrowed by constant-success\nspeculation. Bound unions use the declared question/handler/body contracts.\n\n| Mode | Fixed errors | Conditions | Additional bound sources | Task |\n| --- | --- | --- | --- | --- |\n")
	for _, n := range c.inventory.NativeDeclarations {
		conditions := []string{}
		for _, b := range n.ConditionalEmits {
			conditions = append(conditions, b.Condition+": "+strings.Join(b.Emits, ", "))
		}
		fmt.Fprintf(&doc, "| %s | %s | %s | %s | %s |\n", n.Name, strings.Join(n.Emits, ", "), strings.Join(conditions, "; "), strings.Join(n.UnionBounds, ", "), n.Task)
	}
	doc.WriteString("\nStandard failure categories: " + strings.Join(c.inventory.StandardFailures, ", ") + ".\n\nProperty descriptors (array.length, str.length, bytes.length) require no\ncall marker. Method descriptor names such as array.map are internal lookup\nkeys, not new reserved source packages. Package names cli and json remain\nreserved even though this slice declares no callable members in them.\n")
	type entry struct {
		ID   int    `json:"id"`
		Kind string `json:"kind"`
	}
	registry := struct {
		Active  []entry `json:"active"`
		Retired []int   `json:"retired"`
	}{Active: []entry{}, Retired: []int{}}
	for _, e := range c.inventory.Errors {
		registry.Active = append(registry.Active, entry{ID: e.ID, Kind: e.Name})
	}
	registryBytes, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{"compiler/internal/catalogue/generated.go": goBytes, "runtime/catalogue.ts": []byte(ts.String()), "std/catalogue/README.md": []byte(strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(doc.String())), "std/catalogue/errors.json": append(registryBytes, '\n')}, nil
}
func fieldText(fs []Field) string {
	xs := []string{}
	for _, f := range fs {
		xs = append(xs, f.Type+" "+f.Name)
	}
	return strings.Join(xs, ", ")
}
func parameterText(ps []Parameter) string {
	xs := []string{}
	for _, p := range ps {
		xs = append(xs, p.Name+":"+p.Constraint)
	}
	return strings.Join(xs, ", ")
}
