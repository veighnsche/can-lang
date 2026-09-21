package check

import (
	"fmt"
	"reflect"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

type siteKey struct {
	kind string
	span source.Span
}
type lexicalSites map[siteKey]string

// Source spans are lookup keys only. Emitted identities contain the resolved
// declaration and preorder ordinal, never offsets or checker-local counters.
// Walking the syntax structs in field/slice order covers nested native bodies,
// fixture expressions and handlers as well as ordinary function expressions.
// Type checking may revisit a node; that cannot allocate another lexical site.
func indexLexicalSites(owner string, root any) lexicalSites {
	sites := lexicalSites{}
	ordinal := 0
	add := func(kind string, span source.Span) {
		key := siteKey{kind, span}
		if sites[key] == "" {
			sites[key] = fmt.Sprintf("%s#%d", owner, ordinal)
			ordinal++
		}
	}
	var visit func(reflect.Value)
	visit = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		switch value.Kind() {
		case reflect.Interface, reflect.Pointer:
			if !value.IsNil() {
				visit(value.Elem())
			}
		case reflect.Slice:
			for i := 0; i < value.Len(); i++ {
				visit(value.Index(i))
			}
		case reflect.Struct:
			if value.Type().PkgPath() != reflect.TypeOf(syntax.Block{}).PkgPath() {
				return
			}
			switch node := value.Interface().(type) {
			case syntax.CallExpr:
				add("call", node.Invocation.Span)
			case syntax.MethodInvocation:
				add("method", node.Span)
			case syntax.ReferenceExpr:
				add("callable", node.Span)
			case syntax.Coordination:
				add("coordination", node.Span)
			}
			for i := 0; i < value.NumField(); i++ {
				visit(value.Field(i))
			}
		}
	}
	visit(reflect.ValueOf(root))
	return sites
}

func (c *regionChecker) lexicalSite(kind string, span source.Span) (string, error) {
	if site := c.context.Sites[siteKey{kind, span}]; site != "" {
		return site, nil
	}
	return "", fmt.Errorf("missing typed syntax site for %s at byte %d", kind, span.Start)
}
