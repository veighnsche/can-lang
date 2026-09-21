package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// ArrayOfChecked derives a structural array from already validated data. It
// introduces no declaration or recursion edge, and does not reopen its element's
// sealed graph. This supports finite expression inference after annotation checks.
func ArrayOfChecked(element *Type) (*Type, error) {
	if element == nil || !Equal(element, element) || element.kind == Void {
		return nil, fmt.Errorf("array element requires checked data")
	}
	key := identity(Array, "", []*Type{element})
	sum := sha256.Sum256([]byte("can-concrete-type-v1\x00" + key))
	id := hex.EncodeToString(sum[:])
	if previous := element.graph.nodes[id]; previous != nil {
		if previous.key != key {
			return nil, fmt.Errorf("concrete type identity collision")
		}
		return previous, nil
	}
	t := &Type{kind: Array, key: key, id: id, element: element, defined: true, graph: newGraph()}
	t.graph.nodes[id] = t
	t.graph.sealed = true
	return t, nil
}

// CallableOfChecked derives a residual closure signature from sealed contracts.
// Like ArrayOfChecked, it adds no nominal definitions or recursion edges.
func CallableOfChecked(result *Type, inputs, errors []*Type) (*Type, error) {
	if !Equal(result, result) {
		return nil, fmt.Errorf("callable result requires checked type")
	}
	for _, input := range inputs {
		if !Equal(input, input) || input.Kind() == Void {
			return nil, fmt.Errorf("callable input requires checked data")
		}
	}
	bound := append([]*Type(nil), errors...)
	for _, e := range bound {
		if !Equal(e, e) || e.Kind() != Error {
			return nil, fmt.Errorf("callable requires checked nominal errors")
		}
	}
	sort.Slice(bound, func(i, j int) bool { return bound[i].Identity() < bound[j].Identity() })
	for i, e := range bound {
		if !Equal(e, e) || e.Kind() != Error || i > 0 && Equal(e, bound[i-1]) {
			return nil, fmt.Errorf("callable requires a distinct nominal error bound")
		}
	}
	data, _ := json.Marshal([]any{Callable, result.id, ids(inputs), ids(bound)})
	key := string(data)
	sum := sha256.Sum256([]byte("can-concrete-type-v1\x00" + key))
	id := hex.EncodeToString(sum[:])
	if previous := result.graph.nodes[id]; previous != nil {
		if previous.key != key {
			return nil, fmt.Errorf("concrete type identity collision")
		}
		return previous, nil
	}
	t := &Type{kind: Callable, key: key, id: id, result: result, inputs: append([]*Type(nil), inputs...), errors: bound, defined: true, graph: newGraph()}
	t.graph.nodes[id] = t
	t.graph.sealed = true
	return t, nil
}
