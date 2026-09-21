package check

import (
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// The state object is compiler-owned protocol data, not a new constructible Can
// nominal type. Each field's graph is nevertheless the ordinary sealed codec
// graph, including recursive records and exact integer leaves.
func judgeState(native *NativeDeclaration, inputs []ir.Local) (types.CodecSchema, []ir.Local, error) {
	stateInputs := append([]ir.Local(nil), inputs[len(inputs)-len(native.State):]...)
	root := types.CodecNode{Identity: "can:judge-state/" + native.Symbol.ID, Kind: types.Record, Name: "state"}
	nodes := map[string]types.CodecNode{}
	for i, input := range stateInputs {
		schema, err := types.Schema(input.Type)
		if err != nil {
			return types.CodecSchema{}, nil, err
		}
		root.Fields = append(root.Fields, types.CodecField{Name: native.State[i].Name.Text, Type: schema.Root})
		for _, node := range schema.Nodes {
			nodes[node.Identity] = node
		}
	}
	nodes[root.Identity] = root
	result := types.CodecSchema{Root: root.Identity}
	for _, node := range nodes {
		result.Nodes = append(result.Nodes, node)
	}
	sort.Slice(result.Nodes, func(i, j int) bool { return result.Nodes[i].Identity < result.Nodes[j].Identity })
	return result, stateInputs, nil
}
