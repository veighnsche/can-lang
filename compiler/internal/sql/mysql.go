package sql

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	tidbparser "github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"
	testdriver "github.com/pingcap/tidb/pkg/parser/test_driver"
)

// MySQLVersion is the grammar compatibility version the tidb backend
// targets (MySQL 8.0.11), recorded on checked descriptors and pinned
// against the runtime's parser table. The module carries no numeric
// version of its own; the pin itself lives in go.mod.
const MySQLVersion = 80011

// Assumed server sql_mode is the MySQL 8.4 default. The parser runs its
// own default mode, which matches except for NO_AUTO_CREATE_USER (a
// server-removed flag with no lexer effect): backslash escapes stay on
// and ANSI_QUOTES stays off, exactly what my-03/my-04 pin.
type mysqlSiteCollector struct {
	offsets []int
}

func (c *mysqlSiteCollector) Enter(n ast.Node) bool {
	if marker, ok := n.(*testdriver.ParamMarkerExpr); ok {
		c.offsets = append(c.offsets, marker.Offset)
	}
	return false
}

func (c *mysqlSiteCollector) Leave(ast.Node) bool { return true }

// mysqlLineColumn converts the backend's "line L column C" failure
// position into a character offset. The shape is diagnostic only: any
// deviation falls back to 0 rather than mispositioning, and rejection
// never depends on this parse.
func mysqlLineColumn(input, message string) int {
	lineIdx := strings.Index(message, "line ")
	if lineIdx < 0 {
		return 0
	}
	rest := message[lineIdx+len("line "):]
	colIdx := strings.Index(rest, " column ")
	if colIdx < 0 {
		return 0
	}
	line, err := strconv.Atoi(rest[:colIdx])
	if err != nil || line < 1 {
		return 0
	}
	rest = rest[colIdx+len(" column "):]
	digits := 0
	for digits < len(rest) && rest[digits] >= '0' && rest[digits] <= '9' {
		digits++
	}
	column, err := strconv.Atoi(rest[:digits])
	if err != nil || digits == 0 || column < 1 {
		return 0
	}
	offset := 0
	current := 1
	for offset < len(input) && current < line {
		if input[offset] == '\n' {
			current++
		}
		offset++
	}
	if current != line {
		return 0
	}
	runes := []rune(input[offset:])
	if column-1 > len(runes) {
		return 0
	}
	return utf8.RuneCountInString(input[:offset]) + column - 1
}

// mysqlFailure wraps the backend message so grammar failures keep the
// corpus "syntax error" shape with a best-effort character cursor.
func mysqlFailure(input string, cause error) *Failure {
	message := strings.TrimSpace(cause.Error())
	if !strings.Contains(message, "syntax error") {
		message = "syntax error: " + message
	}
	return &Failure{Message: message, Cursor: mysqlLineColumn(input, cause.Error())}
}

// mysqlKindName reports the backend's own statement tag: SelectStmt,
// InsertStmt, UpdateStmt, DeleteStmt admit through the shared checks,
// and anything else (SetOprStmt for UNION, ExplainStmt, ...) rejects
// as a cardinality mismatch.
func mysqlKindName(name string, node ast.StmtNode) (string, error) {
	raw := fmt.Sprintf("%T", node)
	tag, ok := strings.CutPrefix(raw, "*ast.")
	if !ok || tag == "" {
		return "", fmt.Errorf("sql descriptor %q: mysql node shape %q", name, raw)
	}
	return tag, nil
}

// mysqlLimitParam reads only the top-level LIMIT count: its parameter
// number when the count is exactly one parameter site and the offset
// binds no site, else 0. A LIMIT clause holding two sites (LIMIT ?, ?
// or LIMIT ? OFFSET ?) cannot satisfy the single trailing number, so
// it reports 0 and the cardinality check rejects it. Subquery limits
// never surface because only the top node reads.
func mysqlLimitParam(limit *ast.Limit, numbers map[int]int) int {
	if limit == nil {
		return 0
	}
	if _, ok := limit.Offset.(*testdriver.ParamMarkerExpr); ok {
		return 0
	}
	marker, ok := limit.Count.(*testdriver.ParamMarkerExpr)
	if !ok {
		return 0
	}
	return numbers[marker.Offset]
}

func mysqlReturning(node ast.StmtNode) bool {
	switch typed := node.(type) {
	case *ast.InsertStmt:
		return len(typed.Returning) > 0
	case *ast.UpdateStmt:
		return len(typed.Returning) > 0
	case *ast.DeleteStmt:
		return len(typed.Returning) > 0
	}
	return false
}

// analyzeMySQL runs the tidb backend: statement spans located from the
// nodes' own text (position recording stays off upstream), positional
// parameter sites in source order, top-level LIMIT shape, and
// RETURNING presence. Every reported offset must point at a literal
// "?" byte; anything else is an adapter defect, never a site.
func analyzeMySQL(name, statement string) (Analysis, error) {
	var out Analysis
	parser := tidbparser.New()
	nodes, _, err := parser.Parse(statement, "", "")
	if err != nil {
		out.Failure = mysqlFailure(statement, err)
		return out, nil
	}
	out.Version = MySQLVersion
	cursor := 0
	for _, node := range nodes {
		text := node.Text()
		if text == "" {
			return Analysis{}, fmt.Errorf("sql descriptor %q: mysql statement without text", name)
		}
		at := strings.Index(statement[cursor:], text)
		if at < 0 {
			return Analysis{}, fmt.Errorf("sql descriptor %q: mysql statement span outside input", name)
		}
		kind, err := mysqlKindName(name, node)
		if err != nil {
			return Analysis{}, err
		}
		out.Statements = append(out.Statements, Statement{
			Location: cursor + at,
			Length:   len(text),
			Kind:     kind,
		})
		cursor += at + len(text)
	}
	if len(nodes) != 1 {
		return out, nil
	}
	collector := &mysqlSiteCollector{}
	ast.Walk(nodes[0], collector)
	sort.Ints(collector.offsets)
	numbers := make(map[int]int, len(collector.offsets))
	for i, offset := range collector.offsets {
		if offset < 0 || offset >= len(statement) || statement[offset] != '?' {
			return Analysis{}, fmt.Errorf("sql descriptor %q: mysql site %d outside parameter text", name, offset)
		}
		numbers[offset] = i + 1
		out.Sites = append(out.Sites, ParamSite{
			Number: i + 1,
			Start:  offset,
			End:    offset + 1,
			Ref:    "?",
		})
	}
	shaped := &out.Statements[0]
	shaped.Returning = mysqlReturning(nodes[0])
	if selectStmt, ok := nodes[0].(*ast.SelectStmt); ok {
		shaped.HasLimit = selectStmt.Limit != nil
		shaped.LimitParam = mysqlLimitParam(selectStmt.Limit, numbers)
	}
	return out, nil
}
