// Package sqlite parses SQLite descriptor statements with the vendored
// tree-sitter parse.y mirror. It exposes the concrete syntax tree as Go
// nodes with exact byte spans; statement-kind, LIMIT-shape, RETURNING,
// and parameter-number rules live in the parent sql package. No grammar
// knowledge beyond node navigation belongs here.
package sqlite

// #cgo CFLAGS: -I${SRCDIR}
// #include <stdlib.h>
// char *sqlite_dump(const char *src, unsigned len, int *has_error, int *truncated);
// void sqlite_free(char *p);
import "C"

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

// LanguageVersion is the vendored grammar language version, recorded on
// checked SQLite descriptors alongside the vendored content pin.
const LanguageVersion = 15

// Node is one concrete syntax node: its grammar type, byte span into the
// parsed source, and pre-order children. Named reports grammar-named
// nodes; anonymous tokens (punctuation, keywords) carry Named=false.
type Node struct {
	Type       string
	Start, End int
	Named      bool
	Children   []*Node
}

// Parse dumps one SQLite statement into its concrete syntax tree. hasError
// reports ERROR or MISSING nodes anywhere in the tree. A truncated dump
// or an unloadable grammar is a Go error, never a silent tree.
func Parse(input string) (root *Node, hasError bool, err error) {
	csrc := C.CString(input)
	defer C.free(unsafe.Pointer(csrc))
	var cError, cTruncated C.int
	dump := C.sqlite_dump(csrc, C.uint(len(input)), &cError, &cTruncated)
	if dump == nil {
		return nil, false, fmt.Errorf("sqlite grammar failed to load")
	}
	defer C.sqlite_free(dump)
	if cTruncated != 0 {
		return nil, false, fmt.Errorf("sqlite statement exceeds the dump limit")
	}
	raw := C.GoString(dump)
	lines := strings.Split(raw, "\n")
	type framed struct {
		depth int
		node  *Node
	}
	var stack []framed
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\x1f")
		if len(fields) != 5 {
			return nil, false, fmt.Errorf("sqlite dump shape: %q", line)
		}
		depth, err := strconv.Atoi(fields[0])
		if err != nil || depth < 0 {
			return nil, false, fmt.Errorf("sqlite dump depth: %q", line)
		}
		start, err := strconv.Atoi(fields[3])
		if err != nil {
			return nil, false, fmt.Errorf("sqlite dump span: %q", line)
		}
		end, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, false, fmt.Errorf("sqlite dump span: %q", line)
		}
		if fields[1] != "0" && fields[1] != "1" {
			return nil, false, fmt.Errorf("sqlite dump shape: %q", line)
		}
		if fields[2] == "" || start < 0 || end < start || end > len(input) {
			return nil, false, fmt.Errorf("sqlite dump span: %q", line)
		}
		if len(stack) > 0 && depth > stack[len(stack)-1].depth+1 {
			return nil, false, fmt.Errorf("sqlite dump depth: %q", line)
		}
		node := &Node{Type: fields[2], Start: start, End: end, Named: fields[1] == "1"}
		for len(stack) > 0 && stack[len(stack)-1].depth >= depth {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			if root != nil || depth != 0 {
				return nil, false, fmt.Errorf("sqlite dump shape: %q", line)
			}
			root = node
		} else {
			parent := stack[len(stack)-1].node
			parent.Children = append(parent.Children, node)
		}
		stack = append(stack, framed{depth: depth, node: node})
	}
	if root == nil {
		return nil, false, fmt.Errorf("sqlite dump is empty")
	}
	return root, cError != 0, nil
}

// Text slices the node's exact source bytes.
func (n *Node) Text(input string) string { return input[n.Start:n.End] }

// Statements returns the top-level statement children: named nodes under
// the source_file root. Anonymous semicolons and trivia stay out; callers
// count these to enforce the single-statement rule.
func (n *Node) Statements() []*Node {
	var out []*Node
	for _, child := range n.Children {
		if child.Named {
			out = append(out, child)
		}
	}
	return out
}

// Find returns the first node with the given type in pre-order, or nil.
func (n *Node) Find(nodeType string) *Node {
	if n.Type == nodeType {
		return n
	}
	for _, child := range n.Children {
		if found := child.Find(nodeType); found != nil {
			return found
		}
	}
	return nil
}

// Collect appends every node with the given type in pre-order.
func (n *Node) Collect(nodeType string, out *[]*Node) {
	if n.Type == nodeType {
		*out = append(*out, n)
	}
	for _, child := range n.Children {
		child.Collect(nodeType, out)
	}
}
