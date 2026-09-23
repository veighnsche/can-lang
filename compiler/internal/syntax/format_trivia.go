package syntax

import (
	"fmt"
	"sort"
	"strings"
)

// FormatTrivia renders the canonical layout with source trivia preserved.
// Own-line comments keep their content and attachment at canonical indent,
// trailing comments stay on their code lines, and blank runs between
// rendered lines survive verbatim. String contents render byte-for-byte
// from their tokens; only comment line endings normalize to LF. Output
// ends with exactly one newline. Every comment must attach; anything
// unplaceable fails instead of silently dropping.
func FormatTrivia(file *File) (string, error) {
	if file == nil || file.Source == nil {
		return "", fmt.Errorf("format requires a parsed source file")
	}
	text := file.Source.Text()
	starts := lineStarts(text)
	lineOf := func(offset int) (int, error) {
		if offset < 0 || offset > len(text) {
			return 0, fmt.Errorf("format anchor %d is outside the source", offset)
		}
		line := 1
		for line < len(starts) && starts[line] <= offset {
			line++
		}
		return line, nil
	}
	f := &formatter{trivia: true}
	f.render(file)
	if len(f.staged) == 0 {
		return "", fmt.Errorf("format found no lines to render")
	}
	anchorLines := make([]int, len(f.staged))
	for i, staged := range f.staged {
		line, err := lineOf(staged.offset)
		if err != nil {
			return "", err
		}
		if i > 0 && line < anchorLines[i-1] {
			return "", fmt.Errorf("format anchor order broke at source line %d", line)
		}
		anchorLines[i] = line
	}
	leading, trailing, err := splitComments(text, starts, lineOf, file.Comments)
	if err != nil {
		return "", err
	}
	resolveSynthetic(text, starts, f.staged, anchorLines, leading, trailing)
	var out []string
	lineIndex := make([]int, len(f.staged))
	li := 0
	prev := 0
	for i, staged := range f.staged {
		line := anchorLines[i]
		for source := prev + 1; source <= line; source++ {
			for li < len(leading) && leading[li].line == source {
				out = append(out, indentText(staged.level, cleanComment(leading[li].comment.Text)))
				li++
			}
			if source < line && prev > 0 && isBlank(text, starts, source) {
				out = append(out, "")
			}
		}
		lineIndex[i] = len(out)
		out = append(out, indentText(staged.level, staged.text))
		prev = line
	}
	for ; li < len(leading); li++ {
		start := starts[leading[li].line-1]
		out = append(out, text[start:leading[li].comment.Span.Start]+cleanComment(leading[li].comment.Text))
	}
	trailed := make([]bool, len(f.staged))
	for _, placed := range trailing {
		comment := placed.comment
		line, err := lineOf(comment.Span.Start)
		if err != nil {
			return "", err
		}
		target := -1
		for i := len(anchorLines) - 1; i >= 0; i-- {
			if anchorLines[i] <= line {
				target = i
				break
			}
		}
		if target < 0 {
			return "", fmt.Errorf("format cannot attach the comment at source line %d", line)
		}
		at := lineIndex[target]
		gap := "  "
		if trailed[target] {
			gap = " "
		}
		trailed[target] = true
		out[at] += gap + cleanComment(comment.Text)
	}
	return strings.Join(out, "\n") + "\n", nil
}

type placedComment struct {
	comment Comment
	line    int
}

// resolveSynthetic assigns each synthetic layout line its true keyword
// line: the first code line between the surrounding real anchors. The
// keyword always sits there in valid sources; without it the line keeps
// the previous anchor so attachment stays adjacent rather than failing.
func resolveSynthetic(text string, starts []int, staged []stagedLine, anchorLines []int, leading, trailing []placedComment) {
	startsOn := func(list []placedComment, line int) bool {
		for _, placed := range list {
			if placed.line == line {
				return true
			}
		}
		return false
	}
	isCode := func(line int) bool {
		if line < 1 || line > len(starts) || isBlank(text, starts, line) {
			return false
		}
		if startsOn(trailing, line) {
			return true
		}
		return !startsOn(leading, line)
	}
	for i, line := range staged {
		if !line.synthetic {
			continue
		}
		prev := anchorLines[i-1]
		next := len(starts) + 1
		for j := i + 1; j < len(staged); j++ {
			if !staged[j].synthetic {
				next = anchorLines[j]
				break
			}
		}
		for line := prev + 1; line < next; line++ {
			if isCode(line) {
				anchorLines[i] = line
				break
			}
		}
	}
}

// splitComments divides trivia into own-line comments, which render ahead
// of their code, and trailing comments, which stay on their code lines.
// A comment is own-line when only spaces or tabs precede it on its line.
func splitComments(text string, starts []int, lineOf func(int) (int, error), comments []Comment) (leading, trailing []placedComment, err error) {
	ordered := append([]Comment(nil), comments...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Span.Start < ordered[j].Span.Start })
	for _, comment := range ordered {
		line, err := lineOf(comment.Span.Start)
		if err != nil {
			return nil, nil, err
		}
		prefix := text[starts[line-1]:comment.Span.Start]
		own := true
		for _, r := range prefix {
			if r != ' ' && r != '\t' {
				own = false
				break
			}
		}
		if own {
			leading = append(leading, placedComment{comment: comment, line: line})
		} else {
			trailing = append(trailing, placedComment{comment: comment, line: line})
		}
	}
	return leading, trailing, nil
}

func lineStarts(text string) []int {
	starts := []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func isBlank(text string, starts []int, line int) bool {
	start := starts[line-1]
	end := len(text)
	if line < len(starts) {
		end = starts[line] - 1
	}
	return strings.TrimSpace(text[start:end]) == ""
}

func indentText(level int, text string) string {
	return strings.Repeat("    ", level) + text
}

// cleanComment normalizes comment line endings to LF. Comment content is
// otherwise verbatim; string tokens never pass through here.
func cleanComment(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "")
}
