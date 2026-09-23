package sql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/pganalyze/pg_query_go/v6/parser"
)

// Major is the pinned PostgreSQL parser major. Every successful result
// asserts it; a different major is an adapter error, never a silent parse.
const Major = 17

// Token is one PostgreSQL scanner token with byte offsets into the input.
// Kind holds the upstream token name (PARAM, SELECT, SCONST, ICONST,
// ASCII_59, ...) and Keyword the keyword kind (RESERVED_KEYWORD,
// NO_KEYWORD, ...).
type Token struct {
	Start   int
	End     int
	Kind    string
	Keyword string
}

var (
	versionOnce  sync.Once
	versionValue int
	versionErr   error
)

// Version returns the full upstream parser version number (170007) after
// asserting the pinned major. Checked descriptors record it.
func Version() (int, error) {
	versionOnce.Do(func() {
		tree, err := pg_query.Parse("SELECT 1")
		if err != nil {
			versionErr = fmt.Errorf("sql parser version probe: %w", err)
			return
		}
		versionValue = int(tree.Version)
		if err := checkMajor(versionValue); err != nil {
			versionErr = err
		}
	})
	return versionValue, versionErr
}

func checkMajor(version int) error {
	if version/10000 != Major {
		return fmt.Errorf("sql parser major %d, want %d (version %d)", version/10000, Major, version)
	}
	return nil
}

func failureOf(err error) (*Failure, error) {
	var upstream *parser.Error
	if !errors.As(err, &upstream) {
		return nil, fmt.Errorf("sql parser error shape: %T", err)
	}
	return &Failure{Message: upstream.Message, Cursor: upstream.Cursorpos}, nil
}

// limitShape reads only the top-level statement node: a LIMIT clause with
// its parameter number for SELECT, and RETURNING presence for mutations.
// Subquery limits and nested shapes are invisible here by construction.
func limitShape(node *pg_query.Node) Statement {
	var out Statement
	if selectStmt := node.GetSelectStmt(); selectStmt != nil {
		if limit := selectStmt.GetLimitCount(); limit != nil {
			out.HasLimit = true
			out.LimitParam = int(limit.GetParamRef().GetNumber())
		}
		return out
	}
	switch {
	case node.GetInsertStmt() != nil:
		out.Returning = len(node.GetInsertStmt().GetReturningList()) > 0
	case node.GetUpdateStmt() != nil:
		out.Returning = len(node.GetUpdateStmt().GetReturningList()) > 0
	case node.GetDeleteStmt() != nil:
		out.Returning = len(node.GetDeleteStmt().GetReturningList()) > 0
	}
	return out
}

func checkSpan(start, end, length int) error {
	if start < 0 || end < start || end > length {
		return fmt.Errorf("sql parser span [%d:%d] outside input of %d bytes", start, end, length)
	}
	return nil
}

// Parse returns the statement spans of input, or the upstream failure.
// A nil failure with empty statements means no statements, not an error.
//
// Upstream span quirks, pinned by parser_test.go and the research corpus:
//   - Statement Length is 0 unless the statement ends with a semicolon.
//   - Statement Location points just past the previous semicolon and can
//     include leading whitespace.
//   - Empty, comment-only, and lone-semicolon inputs yield zero statements
//     with no failure.
func Parse(input string) ([]Statement, *Failure, error) {
	tree, err := pg_query.Parse(input)
	if err != nil {
		failure, ferr := failureOf(err)
		if ferr != nil {
			return nil, nil, ferr
		}
		return nil, failure, nil
	}
	if err := checkMajor(int(tree.Version)); err != nil {
		return nil, nil, err
	}
	out := make([]Statement, 0, len(tree.Stmts))
	for _, stmt := range tree.Stmts {
		location, length := int(stmt.StmtLocation), int(stmt.StmtLen)
		if err := checkSpan(location, location+length, len(input)); err != nil {
			return nil, nil, err
		}
		kind := ""
		var shaped Statement
		if stmt.Stmt != nil {
			raw := fmt.Sprintf("%T", stmt.Stmt.Node)
			trimmed, ok := strings.CutPrefix(raw, "*pg_query.Node_")
			if !ok || trimmed == "" {
				return nil, nil, fmt.Errorf("sql parser node shape: %s", raw)
			}
			kind = trimmed
			shaped = limitShape(stmt.Stmt)
		}
		shaped.Location, shaped.Length, shaped.Kind = location, length, kind
		out = append(out, shaped)
	}
	return out, nil, nil
}

// Scan returns the scanner token stream of input, or the upstream failure.
// Token Start/End are byte offsets into the input, also before and after
// multibyte characters. Failure Cursor is a character offset.
func Scan(input string) ([]Token, *Failure, error) {
	scan, err := pg_query.Scan(input)
	if err != nil {
		failure, ferr := failureOf(err)
		if ferr != nil {
			return nil, nil, ferr
		}
		return nil, failure, nil
	}
	if err := checkMajor(int(scan.Version)); err != nil {
		return nil, nil, err
	}
	out := make([]Token, 0, len(scan.Tokens))
	for _, token := range scan.Tokens {
		start, end := int(token.Start), int(token.End)
		if err := checkSpan(start, end, len(input)); err != nil {
			return nil, nil, err
		}
		out = append(out, Token{Start: start, End: end, Kind: token.Token.String(), Keyword: token.KeywordKind.String()})
	}
	return out, nil, nil
}

// postgresSites converts the PostgreSQL scanner PARAM tokens into source-
// ordered parameter sites. Text resembling $N inside quotes or comments
// never surfaces as a PARAM token, so every site here is genuine.
func postgresSites(name, statement string, tokens []Token) ([]ParamSite, error) {
	var out []ParamSite
	last := 0
	for _, token := range tokens {
		if token.Start < last {
			return nil, fmt.Errorf("sql descriptor %q: scanner tokens out of order", name)
		}
		last = token.Start
		if token.Kind != "PARAM" {
			continue
		}
		text := statement[token.Start:token.End]
		number, err := strconv.Atoi(strings.TrimPrefix(text, "$"))
		if err != nil || "$"+strconv.Itoa(number) != text {
			return nil, fmt.Errorf("sql descriptor %q: parameter token %q", name, text)
		}
		out = append(out, ParamSite{Number: number, Start: token.Start, End: token.End})
	}
	return out, nil
}

// analyzePostgres runs the PostgreSQL backend: statement spans from the
// grammar, parameter sites from the scanner, and the pinned version.
func analyzePostgres(name, statement string) (Analysis, error) {
	var out Analysis
	stmts, failure, err := Parse(statement)
	if err != nil {
		return Analysis{}, err
	}
	if failure != nil {
		out.Failure = failure
		return out, nil
	}
	out.Statements = stmts
	version, err := Version()
	if err != nil {
		return Analysis{}, err
	}
	out.Version = version
	tokens, failure, err := Scan(statement)
	if err != nil {
		return Analysis{}, err
	}
	if failure != nil {
		// Statements stay so the composer keeps the original order:
		// parse failure, statement count, then scan failure.
		out.Failure = failure
		return out, nil
	}
	sites, err := postgresSites(name, statement, tokens)
	if err != nil {
		return Analysis{}, err
	}
	out.Sites = sites
	return out, nil
}
