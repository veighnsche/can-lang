package sql

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/veighnsche/can-lang/compiler/internal/sql/sqlite"
)

// SQLite parameter spellings follow the engine: ?NNN binds number NNN
// explicitly, a bare ? takes one past the current maximum, and each
// distinct :name/@name/$name takes the next number with repeats sharing
// it. Sigils do not distinguish names: :a, @a, and $a are one parameter.
// $1-style digit names never reach this mapping because the grammar
// rejects them as syntax errors.
func mapSQLiteNumbers(texts []string) []int {
	out := make([]int, len(texts))
	max := 0
	named := map[string]int{}
	for i, text := range texts {
		switch {
		case text == "?":
			max++
			out[i] = max
		case strings.HasPrefix(text, "?"):
			number, err := strconv.Atoi(text[1:])
			if err != nil || number < 0 {
				out[i] = -1
			} else {
				out[i] = number
			}
			if out[i] > max {
				max = out[i]
			}
		default:
			name := text[1:]
			number, seen := named[name]
			if !seen {
				max++
				number = max
				named[name] = number
			}
			out[i] = number
		}
	}
	return out
}

// sqliteRef renders one mapped site for diagnostics: explicit and named
// spellings print verbatim, bare sites print with their mapped number.
func sqliteRef(text string, number int) string {
	if text == "?" {
		return "?" + strconv.Itoa(number)
	}
	return text
}

// sqliteFailure builds the grammar failure from the first ERROR or
// MISSING node in pre-order: a structural message plus the character
// offset. The grammar supplies no message text, so the cursor and the
// offending span are the whole diagnostic.
func sqliteFailure(input string, root *sqlite.Node) *Failure {
	var first *sqlite.Node
	var walk func(n *sqlite.Node)
	walk = func(n *sqlite.Node) {
		if first != nil {
			return
		}
		if n.Type == "ERROR" || n.Type == "MISSING" {
			first = n
			return
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	if first == nil {
		return &Failure{Message: "syntax error", Cursor: 0}
	}
	cursor := utf8.RuneCountInString(input[:first.Start])
	text := first.Text(input)
	if text == "" {
		return &Failure{Message: "syntax error", Cursor: cursor}
	}
	runes := []rune(text)
	if len(runes) > 48 {
		text = string(runes[:48])
	}
	return &Failure{Message: fmt.Sprintf("syntax error near %q", text), Cursor: cursor}
}

// sqliteLimitShape reads only the top-level limit_clause of a SELECT: the
// clause must hold exactly one bare parameter site for LimitParam, else 0.
// Comments inside the clause are trivia; any other named node (an offset
// form, an operator, a function call) means the limit is not a lone site.
// Subquery limits never surface here because only direct children read.
func sqliteLimitShape(stmt *sqlite.Node, numbers map[*sqlite.Node]int) (hasLimit bool, limitParam int) {
	for _, child := range stmt.Children {
		if child.Type != "limit_clause" {
			continue
		}
		var binds []*sqlite.Node
		child.Collect("bind_parameter", &binds)
		bare := len(binds) == 1
		if bare {
			var walk func(n *sqlite.Node)
			walk = func(n *sqlite.Node) {
				if !bare {
					return
				}
				for _, kid := range n.Children {
					if !kid.Named || kid.Type == "bind_parameter" ||
						kid.Type == "line_comment" || kid.Type == "block_comment" {
						walk(kid)
						continue
					}
					bare = false
					return
				}
			}
			walk(child)
		}
		if bare {
			return true, numbers[binds[0]]
		}
		return true, 0
	}
	return false, 0
}

// analyzeSQLite runs the tree-sitter backend: one top-level statement,
// mapped parameter sites in source order, top-level LIMIT shape, and
// RETURNING presence. Top-level comments are trivia like the PostgreSQL
// lexer treats them; anything else named counts as a statement.
func analyzeSQLite(name, statement string) (Analysis, error) {
	var out Analysis
	root, hasError, err := sqlite.Parse(statement)
	if err != nil {
		return Analysis{}, err
	}
	if root.Type != "source_file" {
		return Analysis{}, fmt.Errorf("sql descriptor %q: sqlite root shape %q", name, root.Type)
	}
	if hasError {
		out.Failure = sqliteFailure(statement, root)
		return out, nil
	}
	out.Version = sqlite.LanguageVersion
	var stmts []*sqlite.Node
	for _, child := range root.Statements() {
		if child.Type == "line_comment" || child.Type == "block_comment" {
			continue
		}
		stmts = append(stmts, child)
	}
	for _, stmt := range stmts {
		out.Statements = append(out.Statements, Statement{
			Location: stmt.Start,
			Length:   stmt.End - stmt.Start,
			Kind:     stmt.Type,
		})
	}
	if len(stmts) != 1 {
		return out, nil
	}
	stmt := stmts[0]
	var binds []*sqlite.Node
	stmt.Collect("bind_parameter", &binds)
	texts := make([]string, len(binds))
	for i, bind := range binds {
		texts[i] = bind.Text(statement)
	}
	numbers := mapSQLiteNumbers(texts)
	byNode := map[*sqlite.Node]int{}
	for i, bind := range binds {
		byNode[bind] = numbers[i]
		out.Sites = append(out.Sites, ParamSite{
			Number: numbers[i],
			Start:  bind.Start,
			End:    bind.End,
			Ref:    sqliteRef(texts[i], numbers[i]),
		})
	}
	shaped := &out.Statements[0]
	for _, child := range stmt.Children {
		if child.Type == "returning_clause" {
			shaped.Returning = true
		}
	}
	if stmt.Type == "select_statement" {
		shaped.HasLimit, shaped.LimitParam = sqliteLimitShape(stmt, byNode)
	}
	return out, nil
}
