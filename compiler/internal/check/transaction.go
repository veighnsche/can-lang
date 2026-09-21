package check

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const (
	sqlWithTransaction        = "can.std.sql@1::with_transaction"
	sqlTransactionQueryOne    = "can.std.sql@1::transaction_query_one"
	sqlTransactionQueryOption = "can.std.sql@1::transaction_query_optional"
	sqlTransactionQueryRows   = "can.std.sql@1::transaction_query_rows"
	sqlTransactionExecute     = "can.std.sql@1::transaction_execute"

	sqlTransactionHandle = "can.std.sql@1::transaction"
	sqlDecision          = "can.std.sql@1::decision"
	sqlCommit            = "can.std.sql@1::commit"
	sqlRollback          = "can.std.sql@1::rollback"
)

// sqlTransactionQueryOperation admits the four I38 in-transaction query
// operations. They share I35 descriptors, P/R checking, and cardinality
// rules; only the first input (a transaction handle, never a pool) and
// the emitted receiver differ.
func sqlTransactionQueryOperation(identity string) bool {
	switch identity {
	case sqlTransactionQueryOne, sqlTransactionQueryOption, sqlTransactionQueryRows, sqlTransactionExecute:
		return true
	}
	return false
}

// isTransactionScopeRequest reports whether the type is the ingress-only
// transaction handle. Handles flow only into the transaction callback and
// its query operations; the scope machinery rejects every escape.
func isTransactionScopeRequest(typ *types.Type) bool {
	return typ != nil && typ.Kind() == types.Opaque && typ.Declaration() == sqlTransactionHandle
}

// TransactionSpecialization binds with_transaction to one concrete result
// type plus the commit/rollback leaf identities the runtime uses to
// classify the callback decision. Leaves are nominal records, so identity
// comparison is exact without consulting bindings.
type TransactionSpecialization struct {
	Operation string
	T         *types.Type
	Commit    string
	Rollback  string
	Contract  *types.Type
}

// specializeTransaction admits with_transaction<T>: T must be data, the
// callback contract is exactly (sql::transaction) -> sql::decision<T>
// with an empty error bound, and the outer contract returns T with the
// catalogue connection/transaction/commit errors. The whole nested
// signature resolves through one specializer call so the callback input
// shares its identity scheme with ordinary declared functions.
func (c *programChecker) specializeTransaction(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, err
	}
	if symbol.ID != sqlWithTransaction {
		return ValueBinding{}, fmt.Errorf("transaction specialization expects with_transaction, saw %s", symbol.ID)
	}
	op := sqlOperation(symbol.ID)
	if op == nil {
		return ValueBinding{}, fmt.Errorf("unknown SQL operation %s", symbol.ID)
	}
	if len(args) != 1 || len(op.Parameters) != 1 {
		return ValueBinding{}, fmt.Errorf("sql with_transaction requires one explicit type argument")
	}
	argument, err := c.annotation(file, args[0], false)
	if err != nil {
		return ValueBinding{}, err
	}
	if argument.Kind() == types.Void {
		return ValueBinding{}, fmt.Errorf("type %s violates catalogue constraint data", argument.Identity())
	}
	key, err := types.SpecializationKey(op.Identity, []*types.Type{argument})
	if err != nil {
		return ValueBinding{}, err
	}
	if old := c.program.Transactions[key]; old != nil {
		return ValueBinding{Identity: key, Type: old.Contract}, nil
	}
	parameters := map[string]*types.Type{"T": argument}
	decision, err := c.catalogueType("sql::decision<T>", parameters)
	if err != nil {
		return ValueBinding{}, err
	}
	special := &TransactionSpecialization{Operation: op.Identity, T: argument}
	for _, leaf := range decision.Leaves() {
		switch leaf.Declaration() {
		case sqlCommit:
			special.Commit = leaf.Identity()
		case sqlRollback:
			special.Rollback = leaf.Identity()
		}
	}
	if special.Commit == "" || special.Rollback == "" {
		return ValueBinding{}, fmt.Errorf("sql with_transaction result leaves are not commit<T> and rollback<T>")
	}
	bound := map[string]*types.Type{"catalogue_t": argument}
	node := func(text string) (syntax.TypeNode, error) {
		return catalogueTypeNode(text, []string{"T"})
	}
	decisionNode, err := node("sql::decision<T>")
	if err != nil {
		return ValueBinding{}, err
	}
	handleNode, err := node("sql::transaction")
	if err != nil {
		return ValueBinding{}, err
	}
	poolNode, err := node("sql::pool")
	if err != nil {
		return ValueBinding{}, err
	}
	resultNode, err := node("T")
	if err != nil {
		return ValueBinding{}, err
	}
	callback := &syntax.CallableType{Result: decisionNode, Inputs: []syntax.TypeNode{handleNode}}
	var failures []syntax.TypeNode
	for _, failure := range op.Emits {
		parsed, err := parseCatalogueType(failure)
		if err != nil {
			return ValueBinding{}, err
		}
		failures = append(failures, parsed)
	}
	signature := &syntax.CallableType{Result: resultNode, Inputs: []syntax.TypeNode{poolNode, callback}}
	signature.Errors.Types = failures
	builtin := &resolve.File{Scope: c.world.Prelude, Imports: c.world.Packages}
	contract, err := c.specializer.Resolve(builtin, signature, bound, false)
	if err != nil {
		return ValueBinding{}, err
	}
	special.Contract = contract
	if c.program.Transactions == nil {
		c.program.Transactions = map[string]*TransactionSpecialization{}
	}
	c.program.Transactions[key] = special
	c.program.Intrinsics[key] = contract
	c.bindings[key] = contract
	c.callables[key] = CallableDeclaration{
		Kind: resolve.Function, Contract: contract,
		Names: []string{"pool", "callback"}, Near: []bool{false, false},
	}
	return ValueBinding{Identity: key, Type: contract}, nil
}
