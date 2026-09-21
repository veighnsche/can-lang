package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/sql"
)

// sqlTable renders checked descriptors as the manifest-owned table consumed
// by $canCreateSQLDescriptors. Entries key by owning project and manifest
// name; static call-site names resolve owner-locally against this table in
// I35. Only segments, identities, and counts cross into TypeScript: no SQL
// text is ever assembled with values.
func sqlTable(program *check.Program) (string, error) {
	type entry struct {
		Cardinality string        `json:"cardinality"`
		Kind        string        `json:"kind"`
		Segments    []sql.Segment `json:"segments"`
		Params      []string      `json:"params"`
		ParamType   string        `json:"paramType"`
		RowType     string        `json:"rowType"`
		Limit       int           `json:"limit"`
		Total       int           `json:"total"`
		Version     int           `json:"version"`
	}
	table := map[string]map[string]entry{}
	for _, descriptor := range program.SQL {
		checked := descriptor.Checked
		if checked.Name == "" || checked.Kind == "" || len(checked.Segments) == 0 {
			return "", fmt.Errorf("sql descriptor is not checked")
		}
		params := descriptor.Parameters
		if params == nil {
			params = []string{}
		}
		owner := table[descriptor.Owner]
		if owner == nil {
			owner = map[string]entry{}
			table[descriptor.Owner] = owner
		}
		owner[checked.Name] = entry{
			Cardinality: checked.Cardinality,
			Kind:        checked.Kind,
			Segments:    checked.Segments,
			Params:      params,
			ParamType:   descriptor.ParamType,
			RowType:     descriptor.RowType,
			Limit:       checked.Limit,
			Total:       checked.Total,
			Version:     checked.Version,
		}
	}
	raw, err := json.Marshal(table)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
