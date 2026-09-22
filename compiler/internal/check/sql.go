package check

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/sql"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
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

// sqlRecord resolves a manifest type name to an ordinary record in the
// owning project. The name must be a bare package-qualified nominal; type
// arguments and unknown packages are rejected without consulting SQL.
func sqlRecord(world *resolve.World, projects map[string]*project.Project, owner, name string) (*resolve.Symbol, error) {
	parts := strings.Split(name, "::")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("type %q is not a bare package-qualified nominal", name)
	}
	owning, ok := projects[owner]
	if !ok {
		return nil, fmt.Errorf("unknown owning project %q", owner)
	}
	if !project.Identifier(parts[0]) || !project.Identifier(parts[1]) {
		return nil, fmt.Errorf("type %q is not a bare package-qualified nominal", name)
	}
	var symbol *resolve.Symbol
	for _, pkg := range world.Packages {
		if pkg.Source == nil || pkg.Source.Owner != owning || pkg.Name != parts[0] {
			continue
		}
		if candidate, ok := pkg.Scope.Symbols[parts[1]]; ok {
			symbol = candidate
		}
	}
	if symbol == nil {
		return nil, fmt.Errorf("type %q is not declared in project %q", name, owner)
	}
	if symbol.Kind != resolve.Record {
		return nil, fmt.Errorf("type %q is not an ordinary record", name)
	}
	return symbol, nil
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

func checkSQLDescriptor(byDeclaration map[string]*types.Type, world *resolve.World, projects map[string]*project.Project, owner, name string, manifest project.SQLDescriptor) (ir.SQLDescriptor, error) {
	fail := func(format string, args ...any) (ir.SQLDescriptor, error) {
		return ir.SQLDescriptor{}, fmt.Errorf("sql %q/%s: %s", owner, name, fmt.Sprintf(format, args...))
	}
	paramSymbol, err := sqlRecord(world, projects, owner, manifest.ParameterType)
	if err != nil {
		return fail("%v", err)
	}
	param, ok := byDeclaration[paramSymbol.ID]
	if !ok || param.Kind() != types.Record || len(param.Arguments()) != 0 {
		return fail("parameter type %q is not a concrete ordinary record", manifest.ParameterType)
	}
	rowSymbol, err := sqlRecord(world, projects, owner, manifest.RowType)
	if err != nil {
		return fail("%v", err)
	}
	row, ok := byDeclaration[rowSymbol.ID]
	if !ok || row.Kind() != types.Record || len(row.Arguments()) != 0 {
		return fail("row type %q is not a concrete ordinary record", manifest.RowType)
	}
	fields := param.Fields()
	if len(fields) != len(manifest.Parameters) {
		return fail("parameter record %q has %d fields, manifest lists %d", manifest.ParameterType, len(fields), len(manifest.Parameters))
	}
	for i, field := range fields {
		if field.Name != manifest.Parameters[i] {
			return fail("parameter record field %d is %q, manifest lists %q", i+1, field.Name, manifest.Parameters[i])
		}
		if _, err := types.SQLFieldOf(field.Type); err != nil {
			return fail("parameter %q has non-SQL type %s", field.Name, field.Type.Declaration())
		}
	}
	for _, field := range row.Fields() {
		if _, err := types.SQLFieldOf(field.Type); err != nil {
			return fail("row field %q has non-SQL type %s", field.Name, field.Type.Declaration())
		}
	}
	checked, err := sql.CheckDescriptor(name, manifest.Statement, manifest.Parameters, manifest.Cardinality, manifest.RowLimitParameter)
	if err != nil {
		return ir.SQLDescriptor{}, err
	}
	return ir.SQLDescriptor{Owner: owner, ParamType: param.Identity(), RowType: row.Identity(), Parameters: append([]string(nil), manifest.Parameters...), Checked: checked}, nil
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

// SQLSpecialization binds one query operation to concrete P/R records plus
// their shared scalar projections. The descriptor is resolved per call
// site, never per specialization, so one P/R pair can serve many names.
type SQLSpecialization struct {
	Operation  string
	P, R       *types.Type
	Params     types.SQLSchema
	Rows       types.SQLSchema
	ResultSome string
	ResultNone string
	Contract   *types.Type
}

// SQLSiteRecord ties one checked call site to its static descriptor name.
// Correspondence with the checked descriptor is validated once the
// descriptor table is finished; see CheckSQLCallSites.
type SQLSiteRecord struct {
	Key   string
	Owner string
	Name  string
}

func (c *programChecker) instantiateSQLQuery(op *catalogue.Operation, args []*types.Type) (ValueBinding, error) {
	if len(args) != len(op.Parameters) {
		return ValueBinding{}, fmt.Errorf("sql query %s requires %d explicit type arguments", op.Name, len(op.Parameters))
	}
	parameters := map[string]*types.Type{}
	for i, p := range op.Parameters {
		parameters[p.Name] = args[i]
	}
	params, err := types.SQLSchemaOf(parameters["P"])
	if err != nil {
		return ValueBinding{}, fmt.Errorf("sql query %s parameters: %v", op.Name, err)
	}
	special := &SQLSpecialization{Operation: op.Identity, P: parameters["P"], Params: params}
	if r, ok := parameters["R"]; ok {
		rows, err := types.SQLSchemaOf(r)
		if err != nil {
			return ValueBinding{}, fmt.Errorf("sql query %s rows: %v", op.Name, err)
		}
		special.R, special.Rows = r, rows
	}
	key, err := types.SpecializationKey(op.Identity, args)
	if err != nil {
		return ValueBinding{}, err
	}
	if old := c.program.SQLs[key]; old != nil {
		return ValueBinding{Identity: key, Type: old.Contract}, nil
	}
	result, err := c.catalogueType(op.Result, parameters)
	if err != nil {
		return ValueBinding{}, err
	}
	if op.Identity == sqlQueryOptional {
		for _, leaf := range result.Leaves() {
			switch leaf.Declaration() {
			case "can.std.option@1::some":
				special.ResultSome = leaf.Identity()
			case "can.std.option@1::none":
				special.ResultNone = leaf.Identity()
			}
		}
		if special.ResultSome == "" || special.ResultNone == "" {
			return ValueBinding{}, fmt.Errorf("sql query %s result is not option::value<R>", op.Name)
		}
	}
	var inputs, failures []*types.Type
	for _, input := range op.Inputs {
		typ, err := c.catalogueType(input.Type, parameters)
		if err != nil {
			return ValueBinding{}, err
		}
		inputs = append(inputs, typ)
	}
	for _, failure := range op.Emits {
		typ, err := c.catalogueType(failure, parameters)
		if err != nil {
			return ValueBinding{}, err
		}
		failures = append(failures, typ)
	}
	contract, err := types.CallableOfChecked(result, inputs, failures)
	if err != nil {
		return ValueBinding{}, err
	}
	special.Contract = contract
	if c.program.SQLs == nil {
		c.program.SQLs = map[string]*SQLSpecialization{}
	}
	c.program.SQLs[key] = special
	c.program.Intrinsics[key] = contract
	c.bindings[key] = contract
	descriptor := CallableDeclaration{Kind: resolve.Function, Contract: contract}
	for _, input := range op.Inputs {
		descriptor.Names = append(descriptor.Names, input.Name)
		descriptor.Near = append(descriptor.Near, false)
	}
	c.callables[key] = descriptor
	return ValueBinding{Identity: key, Type: contract}, nil
}

func (c *programChecker) specializeSQL(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, err
	}
	if symbol.ID == sqlWithTransaction {
		return c.specializeTransaction(file, scope, name, args)
	}
	if !sqlPoolQueryOperation(symbol.ID) && !sqlTransactionQueryOperation(symbol.ID) {
		return ValueBinding{}, fmt.Errorf("generic invocation requires specialization; SQL expects a pool or transaction query operation")
	}
	op := sqlOperation(symbol.ID)
	if op == nil {
		return ValueBinding{}, fmt.Errorf("unknown SQL operation %s", symbol.ID)
	}
	if len(args) != len(op.Parameters) {
		return ValueBinding{}, fmt.Errorf("sql query %s requires %d explicit type arguments", op.Name, len(op.Parameters))
	}
	arguments := make([]*types.Type, len(args))
	for i, arg := range args {
		arguments[i], err = c.annotation(file, arg, false)
		if err != nil {
			return ValueBinding{}, err
		}
	}
	return c.instantiateSQLQuery(op, arguments)
}

// sqlQueryKey recovers the pool or transaction query operation from a
// specialization key, or "" when the key is not a SQL query
// specialization. with_transaction carries no descriptor site.
func sqlQueryKey(key string) string {
	operation, _, ok := strings.Cut(key, "/instance/")
	if !ok || (!sqlPoolQueryOperation(operation) && !sqlTransactionQueryOperation(operation)) {
		return ""
	}
	return operation
}

// resolveSQLSite enforces the catalogue staticInputs contract before
// ordinary argument checking: the descriptor is a static string literal,
// and the emitter splices the checked descriptor value per call site. The
// literal stays in the lowered arguments so supplied fixtures keep matching.
func (c *regionChecker) resolveSQLSite(operation, key string, args []syntax.Argument) (ir.SQLCallSite, error) {
	want := 3
	if operation == sqlQueryRows || operation == sqlTransactionQueryRows {
		want = 4
	}
	if len(args) != want {
		suffix := ""
		if want == 4 {
			suffix = " and max rows"
		}
		handle := "a pool"
		if sqlTransactionQueryOperation(operation) {
			handle = "a transaction handle"
		}
		return ir.SQLCallSite{}, fmt.Errorf("sql query requires %s, a static descriptor, parameters%s", handle, suffix)
	}
	for _, arg := range args {
		if arg.Spread || arg.Group != nil {
			return ir.SQLCallSite{}, fmt.Errorf("sql query requires fixed ordinary arguments")
		}
	}
	literal, ok := fetchUngroup(args[1].Value).(*syntax.LiteralExpr)
	if !ok || literal.Token.Kind != syntax.String {
		return ir.SQLCallSite{}, fmt.Errorf("sql descriptor must be a static string literal")
	}
	name := literal.Token.Value
	if name == "" || !utf8.ValidString(name) || strings.ContainsRune(name, 0) {
		return ir.SQLCallSite{}, fmt.Errorf("sql descriptor must be a static string literal")
	}
	if c.context.SQLSite == nil {
		return ir.SQLCallSite{}, fmt.Errorf("sql query requires its calling project")
	}
	return c.context.SQLSite(key, name), nil
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
