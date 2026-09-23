package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/sql"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// sqlTable renders checked descriptors as the manifest-owned table consumed
// by $canCreateSQLDescriptors. Entries key by owning project and manifest
// name; static call-site names resolve owner-locally against this table in
// I35. Only segments, identities, and counts cross into TypeScript: no SQL
// text is ever assembled with values.
func sqlTable(program *check.Program) (string, error) {
	type entry struct {
		Dialect     string        `json:"dialect"`
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
			Dialect:     string(checked.Dialect),
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

// sqlPlan renders one query specialization as the JSON plan consumed by
// the $canSQLPools methods: shared parameter/row scalar projections plus
// the concrete result option identities for query_optional. The
// descriptor itself is spliced per call site, never carried in the plan.
func sqlPlan(special *check.SQLSpecialization) (string, error) {
	type schema = types.SQLSchema
	plan := struct {
		Params schema  `json:"params"`
		Rows   *schema `json:"rows,omitempty"`
		Some   string  `json:"some,omitempty"`
		None   string  `json:"none,omitempty"`
	}{Params: special.Params}
	if special.Operation != "can.std.sql@1::execute" && special.Operation != "can.std.sql@1::transaction_execute" {
		plan.Rows = &special.Rows
	}
	if special.Operation == "can.std.sql@1::query_optional" || special.Operation == "can.std.sql@1::transaction_query_optional" {
		plan.Some, plan.None = special.ResultSome, special.ResultNone
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// sqlMethod maps a pool or transaction query operation to its runtime
// method. Transaction operations share method names with their pool
// twins; only the receiver differs.
func sqlMethod(operation string) (string, error) {
	switch operation {
	case "can.std.sql@1::query_one", "can.std.sql@1::transaction_query_one":
		return "queryOne", nil
	case "can.std.sql@1::query_optional", "can.std.sql@1::transaction_query_optional":
		return "queryOptional", nil
	case "can.std.sql@1::query_rows", "can.std.sql@1::transaction_query_rows":
		return "queryRows", nil
	case "can.std.sql@1::execute", "can.std.sql@1::transaction_execute":
		return "execute", nil
	}
	return "", fmt.Errorf("unknown SQL specialization %s", operation)
}

// sqlReceiver selects the runtime query receiver: pools own their
// clients, while transaction operations run against the scoped handle.
func sqlReceiver(operation string) string {
	switch operation {
	case "can.std.sql@1::transaction_query_one", "can.std.sql@1::transaction_query_optional",
		"can.std.sql@1::transaction_query_rows", "can.std.sql@1::transaction_execute":
		return "$canTransactions"
	}
	return "$canSQLPools"
}
