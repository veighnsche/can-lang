package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
