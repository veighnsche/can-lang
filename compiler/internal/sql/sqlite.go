package sql

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sqlc-dev/meyer/ast"
	"github.com/sqlc-dev/meyer/parser"
)

// SQLiteVersion is the pinned SQLite backend stamp recorded on checked
// descriptors: Meyer release v0.1.2. It names the parser release, not a
// SQLite engine version or a Tree-sitter ABI number; compiler content
// hashes remain the complete compiled-content identity. Parser or
// adapter admission changes must update this stamp and the module pin
// together.
const SQLiteVersion = 102

// sqliteMaxStatementBytes caps descriptor statement input before parsing:
// the upstream parser reads without a byte ceiling, so the adapter
// enforces one. sqliteMaxTraversalNodes caps the adapter's own iterative
// node walk, retaining the previous one-million-node resource guard.
// Neither promises a hard memory/time bound.
const sqliteMaxStatementBytes = 1 << 20

const sqliteMaxTraversalNodes = 1000000

// sqliteParserOptions selects the grammar Bun's SQLite build accepts:
// ORDER BY and LIMIT on UPDATE and DELETE are admitted there, so the
// pinned build's default rejection must not apply.
var sqliteParserOptions = parser.Options{UpdateDeleteLimit: true}

// sqliteRef renders one site for diagnostics: explicit and named
// spellings print verbatim, bare sites print with their number.
func sqliteRef(text string, number int) string {
	if text == "?" {
		return "?" + strconv.Itoa(number)
	}
	return text
}

// sqliteCursor converts a byte offset into the character cursor the
// Failure contract requires. Negative or out-of-range offsets (the
// backend reports -1 when the position is unknown) map to 0.
func sqliteCursor(input string, offset int) int {
	if offset < 0 || offset > len(input) {
		return 0
	}
	return utf8.RuneCountInString(input[:offset])
}

// sqliteKind maps one upstream statement node to the compiler-owned kind
// tag. Tags keep the established snake_case spellings so cardinality and
// emitter behavior is unchanged. The switch is exhaustive over the pinned
// release; an unmapped node is a backend-shape error, never a silent tag.
func sqliteKind(stmt ast.Stmt) (string, error) {
	switch stmt.(type) {
	case *ast.SelectStmt:
		return "select_statement", nil
	case *ast.InsertStmt:
		return "insert_statement", nil
	case *ast.UpdateStmt:
		return "update_statement", nil
	case *ast.DeleteStmt:
		return "delete_statement", nil
	case *ast.AlterTableStmt:
		return "alter_table_statement", nil
	case *ast.AnalyzeStmt:
		return "analyze_statement", nil
	case *ast.AttachStmt:
		return "attach_statement", nil
	case *ast.BeginStmt:
		return "begin_statement", nil
	case *ast.CommitStmt:
		return "commit_statement", nil
	case *ast.CreateIndexStmt:
		return "create_index_statement", nil
	case *ast.CreateTableStmt:
		return "create_table_statement", nil
	case *ast.CreateTriggerStmt:
		return "create_trigger_statement", nil
	case *ast.CreateViewStmt:
		return "create_view_statement", nil
	case *ast.CreateVirtualTableStmt:
		return "create_virtual_table_statement", nil
	case *ast.DetachStmt:
		return "detach_statement", nil
	case *ast.DropIndexStmt:
		return "drop_index_statement", nil
	case *ast.DropTableStmt:
		return "drop_table_statement", nil
	case *ast.DropTriggerStmt:
		return "drop_trigger_statement", nil
	case *ast.DropViewStmt:
		return "drop_view_statement", nil
	case *ast.ExplainStmt:
		return "explain_statement", nil
	case *ast.PragmaStmt:
		return "pragma_statement", nil
	case *ast.ReindexStmt:
		return "reindex_statement", nil
	case *ast.ReleaseStmt:
		return "release_statement", nil
	case *ast.RollbackStmt:
		return "rollback_statement", nil
	case *ast.SavepointStmt:
		return "savepoint_statement", nil
	case *ast.VacuumStmt:
		return "vacuum_statement", nil
	default:
		return "", fmt.Errorf("unmapped sqlite statement node %T", stmt)
	}
}

// isNilNode reports interface nils and typed nil pointers: Children
// slices are assembled from optional fields, so typed nils must not
// leak into the traversal.
func isNilNode(n ast.Node) bool {
	if n == nil {
		return true
	}
	v := reflect.ValueOf(n)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map:
		return v.IsNil()
	}
	return false
}

// collectBinds gathers every BindParam under root in source order with an
// iterative budgeted walk over Node.Children. The walk never recurses:
// flat operator chains nest deeply without counting against the upstream
// parser's own recursion ceiling, so a recursive visitor is unsafe here.
// The same walk validates every reachable SELECT's own direct compound
// cores, failing fast and deterministically (the root is visited first,
// so a root violation reports exactly as before). Children order is not
// always source order (comma LIMIT visits Count before Offset), so the
// result is sorted by byte position and each span is validated against
// the input before use.
func collectBinds(root ast.Node, input string, budget int) ([]*ast.BindParam, error) {
	var out []*ast.BindParam
	visited := 0
	stack := []ast.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if isNilNode(n) {
			continue
		}
		visited++
		if visited > budget {
			return nil, errors.New("sqlite statement exceeds the node limit")
		}
		if sel, ok := n.(*ast.SelectStmt); ok {
			if policy := misplacedCoreClause(sel); policy != nil {
				return nil, policy
			}
		}
		if bind, ok := n.(*ast.BindParam); ok {
			out = append(out, bind)
		}
		stack = append(stack, n.Children()...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pos() != out[j].Pos() {
			return out[i].Pos() < out[j].Pos()
		}
		return out[i].End() < out[j].End()
	})
	prevEnd := 0
	for _, bind := range out {
		start, end := bind.Pos(), bind.End()
		if start < 0 || end < start || end > len(input) {
			return nil, fmt.Errorf("sqlite bind span [%d,%d) out of bounds", start, end)
		}
		if start < prevEnd {
			return nil, fmt.Errorf("sqlite bind span [%d,%d) overlaps", start, end)
		}
		prevEnd = end
		if bind.Raw != input[start:end] {
			return nil, fmt.Errorf("sqlite bind span [%d,%d) does not match %q", start, end, bind.Raw)
		}
		if strings.HasPrefix(bind.Raw, "#") {
			return nil, &sqlitePolicyError{message: fmt.Sprintf("unsupported parameter spelling %q", bind.Raw), offset: start}
		}
	}
	return out, nil
}

// sqlitePolicyError is an input rejection the adapter synthesizes where
// the upstream grammar admits more than Can accepts: `#` spellings,
// which are outside the declared parameter family, and misplaced
// compound clauses, which the engine rejects. It carries a byte offset
// for cursor conversion.
type sqlitePolicyError struct {
	message string
	offset  int
}

func (e *sqlitePolicyError) Error() string { return e.message }

// misplacedCoreClause rejects ORDER BY or LIMIT on the non-final direct
// cores of one compound SELECT, using the engine's own wording. The
// upstream grammar attaches those clauses per core; the engine accepts
// them only after the final UNION. The collection walk applies this to
// every reachable SELECT, so malformed CTE, subquery and mutation-source
// compounds fail the same way as top-level ones. Only each SELECT's own
// direct cores are inspected.
func misplacedCoreClause(sel *ast.SelectStmt) *sqlitePolicyError {
	if len(sel.Cores) < 2 {
		return nil
	}
	for _, core := range sel.Cores[:len(sel.Cores)-1] {
		query, ok := core.(*ast.SelectQuery)
		if !ok {
			continue
		}
		if len(query.OrderBy) > 0 {
			return &sqlitePolicyError{message: "ORDER BY clause should come after UNION not before", offset: query.OrderBy[0].Pos()}
		}
		if query.Limit != nil {
			return &sqlitePolicyError{message: "LIMIT clause should come after UNION not before", offset: query.Limit.Pos()}
		}
	}
	return nil
}

// sqliteLimitShape reads only the outer LIMIT of a top-level SELECT: the
// final query core must carry a LIMIT whose Count is exactly one
// parameter with no Offset or comma form. Subquery limits never surface
// here because only the final direct core is inspected.
func sqliteLimitShape(stmt ast.Stmt) (hasLimit bool, limitParam int) {
	sel, ok := stmt.(*ast.SelectStmt)
	if !ok || len(sel.Cores) == 0 {
		return false, 0
	}
	query, ok := sel.Cores[len(sel.Cores)-1].(*ast.SelectQuery)
	if !ok || query.Limit == nil {
		return false, 0
	}
	limit := query.Limit
	bind, ok := limit.Count.(*ast.BindParam)
	if !ok || limit.Offset != nil || limit.Comma {
		return true, 0
	}
	return true, bind.Number
}

// sqliteReturning reports a RETURNING list on INSERT/UPDATE/DELETE.
// Cardinality admits only the INSERT shape with cardinality one.
func sqliteReturning(stmt ast.Stmt) bool {
	switch n := stmt.(type) {
	case *ast.InsertStmt:
		return len(n.Returning) > 0
	case *ast.UpdateStmt:
		return len(n.Returning) > 0
	case *ast.DeleteStmt:
		return len(n.Returning) > 0
	}
	return false
}

// analyzeSQLite runs the Meyer backend over one descriptor statement:
// statement spans, parameter sites in source order with upstream
// numbers, top-level LIMIT shape, and RETURNING presence, or the
// grammar failure. Statement spans are the native half-open byte spans
// including any terminating semicolon.
func analyzeSQLite(name, statement string) (Analysis, error) {
	return analyzeSQLiteWithBudgets(name, statement, sqliteMaxStatementBytes, sqliteMaxTraversalNodes)
}

// analyzeSQLiteWithBudgets is analyzeSQLite with injectable budgets so
// small-budget tests can exercise the guards without constructing
// million-node inputs.
func analyzeSQLiteWithBudgets(name, statement string, maxBytes, maxNodes int) (Analysis, error) {
	var out Analysis
	if len(statement) > maxBytes {
		out.Failure = &Failure{Message: "statement exceeds the size limit", Cursor: 0}
		return out, nil
	}
	if i := strings.IndexByte(statement, 0); i >= 0 {
		out.Failure = &Failure{Message: "statement contains a NUL byte", Cursor: sqliteCursor(statement, i)}
		return out, nil
	}
	stmts, err := sqliteParserOptions.ParseString(statement)
	if err != nil {
		var backend *parser.Error
		if errors.As(err, &backend) {
			out.Failure = &Failure{Message: backend.Message, Cursor: sqliteCursor(statement, backend.Offset)}
			return out, nil
		}
		return Analysis{}, fmt.Errorf("sql descriptor %q: %v", name, err)
	}
	out.Version = SQLiteVersion
	for _, stmt := range stmts {
		kind, err := sqliteKind(stmt)
		if err != nil {
			return Analysis{}, fmt.Errorf("sql descriptor %q: %v", name, err)
		}
		out.Statements = append(out.Statements, Statement{
			Location: stmt.Pos(),
			Length:   stmt.End() - stmt.Pos(),
			Kind:     kind,
		})
	}
	if len(stmts) != 1 {
		return out, nil
	}
	binds, err := collectBinds(stmts[0], statement, maxNodes)
	if err != nil {
		var policy *sqlitePolicyError
		if errors.As(err, &policy) {
			out.Failure = &Failure{Message: policy.message, Cursor: sqliteCursor(statement, policy.offset)}
			out.Statements = nil
			out.Version = 0
			return out, nil
		}
		return Analysis{}, fmt.Errorf("sql descriptor %q: %v", name, err)
	}
	for _, bind := range binds {
		out.Sites = append(out.Sites, ParamSite{
			Number: bind.Number,
			Start:  bind.Pos(),
			End:    bind.End(),
			Ref:    sqliteRef(bind.Raw, bind.Number),
		})
	}
	shaped := &out.Statements[0]
	shaped.Returning = sqliteReturning(stmts[0])
	shaped.HasLimit, shaped.LimitParam = sqliteLimitShape(stmts[0])
	return out, nil
}
