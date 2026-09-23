package check

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// SQLSiteRecord ties one checked call site to its static descriptor name.
// Correspondence with the checked descriptor is validated once the
// descriptor table is finished; see CheckSQLCallSites.
type SQLSiteRecord struct {
	Key   string
	Owner string
	Name  string
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
