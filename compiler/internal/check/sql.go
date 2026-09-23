package check

import (
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// SQL value shapes are projected by types.SQLSchemaOf, shared with query
// specialization so manifest checking and call-site checking cannot drift.
const (
	sqlQueryOne      = "can.std.sql@1::query_one"
	sqlQueryOptional = "can.std.sql@1::query_optional"
	sqlQueryRows     = "can.std.sql@1::query_rows"
	sqlExecute       = "can.std.sql@1::execute"

	sqlPoolHandle = "can.std.sql@1::pool"
)

// isPoolScopeRequest reports whether the type is the opaque connection
// pool. No Can expression constructs a pool, so assertion rows omit
// pool inputs while the harness splices its scope token; any pool
// operation the row does not when-supply fails at the denied live
// boundary, exactly like transaction handles.
func isPoolScopeRequest(typ *types.Type) bool {
	return typ != nil && typ.Kind() == types.Opaque && typ.Declaration() == sqlPoolHandle
}

// CheckSQLDescriptors validates every project's manifest SQL descriptors
// against the finished type model and the upstream parser: parameter and
// row types resolve to ordinary records with admitted scalar fields,
// parameter names match record fields in declaration order, and statements
// check via the SQL package. Descriptors resolve in their owning
// project's manifest only; cross-project sharing flows through checked
// descriptor values, never through merged name tables.
func CheckSQLDescriptors(graph *project.Graph, world *resolve.World, model *types.Model) ([]ir.SQLDescriptor, error) {
	byDeclaration := map[string]*types.Type{}
	for _, typ := range model.Types() {
		if _, exists := byDeclaration[typ.Declaration()]; !exists {
			byDeclaration[typ.Declaration()] = typ
		}
	}
	projects := map[string]*project.Project{}
	for key, p := range graph.Projects {
		projects[key] = p
	}
	var out []ir.SQLDescriptor
	for _, key := range projectKeys(graph) {
		names := make([]string, 0, len(graph.Projects[key].Manifest.SQL))
		for name := range graph.Projects[key].Manifest.SQL {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			manifest := graph.Projects[key].Manifest.SQL[name]
			checked, err := checkSQLDescriptor(byDeclaration, world, projects, key, name, manifest)
			if err != nil {
				return nil, err
			}
			out = append(out, checked)
		}
	}
	return out, nil
}

// sqlPoolQueryOperation admits the four I35 pool query operations for
// per-use specialization. Transaction variants stay unadmitted until I38.
func sqlPoolQueryOperation(identity string) bool {
	switch identity {
	case sqlQueryOne, sqlQueryOptional, sqlQueryRows, sqlExecute:
		return true
	}
	return false
}

// sqlGenericOperation covers every generic SQL query operation so the
// fixed-signature admission loop skips them: P and R bind per call site,
// and with_transaction binds T with its callback contract.
func sqlGenericOperation(identity string) bool {
	if sqlPoolQueryOperation(identity) {
		return true
	}
	if identity == sqlWithTransaction {
		return true
	}
	switch identity {
	case "can.std.sql@1::transaction_query_one", "can.std.sql@1::transaction_query_optional",
		"can.std.sql@1::transaction_query_rows", "can.std.sql@1::transaction_execute":
		return true
	}
	return false
}

func sqlOperation(identity string) *catalogue.Operation {
	for _, op := range catalogue.Builtin().Inventory().Operations {
		if op.Identity == identity {
			return &op
		}
	}
	return nil
}

// CheckSQLCallSites validates every recorded query call site against the
// finished descriptor table: the name resolves in the calling project, the
// P/R records are the descriptor's own parameter/row records, and the
// operation matches the descriptor cardinality.
func CheckSQLCallSites(sqls map[string]*SQLSpecialization, sites []SQLSiteRecord, descriptors []ir.SQLDescriptor) error {
	byOwner := map[string]map[string]ir.SQLDescriptor{}
	for _, descriptor := range descriptors {
		owners := byOwner[descriptor.Owner]
		if owners == nil {
			owners = map[string]ir.SQLDescriptor{}
			byOwner[descriptor.Owner] = owners
		}
		owners[descriptor.Checked.Name] = descriptor
	}
	for _, site := range sites {
		special := sqls[site.Key]
		if special == nil {
			return fmt.Errorf("sql query site %s/%s lacks a checked specialization", site.Owner, site.Name)
		}
		descriptor, ok := byOwner[site.Owner][site.Name]
		if !ok {
			return fmt.Errorf("sql query site %s/%s names an undeclared descriptor", site.Owner, site.Name)
		}
		if special.P.Identity() != descriptor.ParamType {
			return fmt.Errorf("sql query site %s/%s binds parameters %s, descriptor wants %s", site.Owner, site.Name, special.P.Identity(), descriptor.ParamType)
		}
		want := ""
		switch special.Operation {
		case sqlQueryOne, sqlTransactionQueryOne:
			want = "one"
		case sqlQueryOptional, sqlTransactionQueryOption:
			want = "optional"
		case sqlQueryRows, sqlTransactionQueryRows:
			want = "many"
		case sqlExecute, sqlTransactionExecute:
			want = "execute"
		}
		if descriptor.Checked.Cardinality != want {
			return fmt.Errorf("sql query site %s/%s needs cardinality %s, descriptor has %s", site.Owner, site.Name, want, descriptor.Checked.Cardinality)
		}
		if special.Operation != sqlExecute && special.Operation != sqlTransactionExecute && special.R.Identity() != descriptor.RowType {
			return fmt.Errorf("sql query site %s/%s binds rows %s, descriptor wants %s", site.Owner, site.Name, special.R.Identity(), descriptor.RowType)
		}
	}
	return nil
}
