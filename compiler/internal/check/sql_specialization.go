package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

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
