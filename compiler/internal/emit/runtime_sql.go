package emit

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// sqlOperationBindings maps the concrete SQL pool operations to their
// state-module targets. Query operations resolve per specialization key
// in sqlSpecializationBindings; only concrete operations bind here.
func sqlOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.sql@1::pool_open":          "$canSQLPools.open",
		"can.std.sql@1::pool_close":         "$canSQLPools.close",
		"can.std.sql@1::sqlite_open_memory": "$canSQLPools.sqliteOpenMemory",
		"can.std.sql@1::sqlite_open_file":   "$canSQLPools.sqliteOpenFile",
		"can.std.sql@1::mysql_open":         "$canSQLPools.mysqlOpen",
	}
	return bindingContribution{domain: "sql", functions: functions}
}

// sqlSpecializationBindings assigns deterministic runtime names to the
// checked SQL query and transaction specializations of this program.
func (assembly *programAssembly) sqlSpecializationBindings() bindingContribution {
	program := assembly.program
	functions := map[string]string{}
	assembly.sqlIDs = make([]string, 0, len(program.SQLs))
	for id := range program.SQLs {
		assembly.sqlIDs = append(assembly.sqlIDs, id)
	}
	sort.Strings(assembly.sqlIDs)
	assembly.sqlNames = map[string]string{}
	for i, id := range assembly.sqlIDs {
		assembly.sqlNames[id] = fmt.Sprintf("$canSQLQuery%d", i)
		functions[id] = assembly.sqlNames[id] + ".run"
	}
	assembly.txIDs = make([]string, 0, len(program.Transactions))
	for id := range program.Transactions {
		assembly.txIDs = append(assembly.txIDs, id)
	}
	sort.Strings(assembly.txIDs)
	assembly.txNames = map[string]string{}
	for i, id := range assembly.txIDs {
		assembly.txNames[id] = fmt.Sprintf("$canSQLTransaction%d", i)
		functions[id] = assembly.txNames[id] + ".run"
	}
	return bindingContribution{domain: "sql-specializations", functions: functions}
}

// sqlStateImports lists the SQL descriptor, pool, and transaction factory
// modules the shared state module needs.
func (assembly *programAssembly) sqlStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/sql/descriptor.ts", Names: []ImportName{{"createSQLDescriptors", "$canCreateSQLDescriptors"}}},
		{Target: runtime + "/platform/sql/pool.ts", Names: []ImportName{{"createSQLPools", "$canCreateSQLPools"}, {"isSQLPoolValue", "$canIsSQLPool"}}},
		{Target: runtime + "/platform/sql/transaction.ts", Names: []ImportName{{"createSQLTransactions", "$canCreateSQLTransactions"}, {"isSQLTransactionValue", "$canIsSQLTransaction"}}},
	}
}

// sqlStateValueImportNames lists the SQL factory values authored and
// assertion modules import from the state module. Transactions are only
// reached through their frozen run constants, never imported directly.
func sqlStateValueImportNames() []ImportName {
	return []ImportName{{"$canSQL", "$canSQL"}, {"$canSQLPools", "$canSQLPools"}}
}

// declareSQLState emits the SQL descriptor, pool, and transaction factory
// bindings.
func (builder *stateBuilder) declareSQLState() {
	builder.out.WriteString("export let $canSQL:ReturnType<typeof $canCreateSQLDescriptors>;\nexport let $canSQLPools:ReturnType<typeof $canCreateSQLPools>;\nexport let $canTransactions:ReturnType<typeof $canCreateSQLTransactions>;\n")
}

// initializeSQLState creates the SQL descriptor, pool and transaction
// factories inside the shared initializer, after the domain runtime
// exists.
func (builder *stateBuilder) initializeSQLState() error {
	sqlDescriptors, err := sqlTable(builder.assembly.program)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "$canSQL=$canCreateSQLDescriptors(%s);\n", sqlDescriptors)
	fmt.Fprintf(&builder.out, "$canSQLPools=$canCreateSQLPools($canDomain,{credentialsMissing:%s,connectionFailed:%s,queryFailed:%s,rowMissing:%s,rowCount:%s,schemaMismatch:%s,constraintFailed:%s,closeFailed:%s,rowLimit:%s,unsupportedValue:%s},$canOriginalEnvironment,$canSQL);\n", quote(builder.numberIDs["can.std.http@1::credentials_missing"]), quote(builder.numberIDs["can.std.sql@1::connection_failed"]), quote(builder.numberIDs["can.std.sql@1::query_failed"]), quote(builder.numberIDs["can.std.sql@1::row_missing"]), quote(builder.numberIDs["can.std.sql@1::row_count"]), quote(builder.numberIDs["can.std.sql@1::schema_mismatch"]), quote(builder.numberIDs["can.std.sql@1::constraint_failed"]), quote(builder.numberIDs["can.std.sql@1::close_failed"]), quote(builder.numberIDs["can.std.sql@1::row_limit"]), quote(builder.numberIDs["can.std.sql@1::unsupported_value"]))
	fmt.Fprintf(&builder.out, "$canTransactions=$canCreateSQLTransactions($canDomain,{connectionFailed:%s,queryFailed:%s,rowMissing:%s,rowCount:%s,schemaMismatch:%s,constraintFailed:%s,rowLimit:%s,unsupportedValue:%s,transactionFailed:%s,commitUnknown:%s},$canSQL);\n", quote(builder.numberIDs["can.std.sql@1::connection_failed"]), quote(builder.numberIDs["can.std.sql@1::query_failed"]), quote(builder.numberIDs["can.std.sql@1::row_missing"]), quote(builder.numberIDs["can.std.sql@1::row_count"]), quote(builder.numberIDs["can.std.sql@1::schema_mismatch"]), quote(builder.numberIDs["can.std.sql@1::constraint_failed"]), quote(builder.numberIDs["can.std.sql@1::row_limit"]), quote(builder.numberIDs["can.std.sql@1::unsupported_value"]), quote(builder.numberIDs["can.std.sql@1::transaction_failed"]), quote(builder.numberIDs["can.std.sql@1::commit_unknown"]))
	return nil
}

// emitSQLKinds emits the opaque-handle kind table the domain predicate
// uses to recognize pool and transaction values at boundaries.
func (builder *stateBuilder) emitSQLKinds() error {
	sqlKinds := map[string]string{}
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() != types.Opaque {
			continue
		}
		switch typ.Declaration() {
		case "can.std.sql@1::pool":
			sqlKinds[typ.Identity()] = "pool"
		case "can.std.sql@1::transaction":
			sqlKinds[typ.Identity()] = "transaction"
		}
	}
	sqlKindsJSON, err := json.Marshal(sqlKinds)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "const $canSQLKinds:Readonly<Record<string,string>>=%s;\n", sqlKindsJSON)
	return nil
}

// emitSQLSpecializationConstants freezes the per-program SQL query and
// transaction dispatch records after the initializer.
func (builder *stateBuilder) emitSQLSpecializationConstants() error {
	for _, id := range builder.assembly.sqlIDs {
		special := builder.assembly.program.SQLs[id]
		method, err := sqlMethod(special.Operation)
		if err != nil {
			return err
		}
		plan, err := sqlPlan(special)
		if err != nil {
			return err
		}
		// The static descriptor literal stays in the lowered arguments for
		// fixture matching; the bound descriptor value arrives spliced per
		// call site and is the only value the runtime method consumes.
		descriptor := "Parameters<typeof $canSQL.template>[0]"
		receiver := sqlReceiver(special.Operation)
		shape := "run:(pool:unknown,_name:unknown,params:unknown,descriptor:" + descriptor + ",$canContext?:$canAssertionContext):Promise<$canCompletion<unknown>>=>" + receiver + "." + method + "(descriptor," + plan + ",pool,params,$canContext)"
		if special.Operation == "can.std.sql@1::query_rows" || special.Operation == "can.std.sql@1::transaction_query_rows" {
			shape = "run:(pool:unknown,_name:unknown,params:unknown,maxRows:bigint,descriptor:" + descriptor + ",$canContext?:$canAssertionContext):Promise<$canCompletion<unknown>>=>" + receiver + "." + method + "(descriptor," + plan + ",pool,params,maxRows,$canContext)"
		}
		fmt.Fprintf(&builder.out, "export const %s = Object.freeze({%s});\n", builder.assembly.sqlNames[id], shape)
	}
	for _, id := range builder.assembly.txIDs {
		special := builder.assembly.program.Transactions[id]
		// The commit/rollback leaves are nominal identities, so the runtime
		// classifies the callback decision without consulting bindings.
		shape := "run:(pool:unknown,callback:unknown,$canContext?:$canAssertionContext):Promise<$canCompletion<unknown>>=>$canTransactions.withTransaction(pool,callback,{commit:" + quote(special.Commit) + ",rollback:" + quote(special.Rollback) + "},$canContext)"
		fmt.Fprintf(&builder.out, "export const %s = Object.freeze({%s});\n", builder.assembly.txNames[id], shape)
	}
	return nil
}
