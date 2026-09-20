// Package source retains original UTF-8 bytes and owns coordinate conversion for
// diagnostics, LSP and source maps. It never reads or executes a source path.
package source

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// Span is a half-open original-file byte interval. Synthetic tokens use empty
// spans at a real scalar/newline boundary.
type Span struct {
	Start int
	End   int
}
type Position struct {
	Line   int
	Column int
} // one-based line and scalar column
// UTF16Position follows LSP: both line and UTF-16 character column are zero-based.
type UTF16Position struct {
	Line      int
	Character int
}

// MapPosition follows source maps: one-based line, zero-based UTF-16 column.
type MapPosition struct {
	Line   int
	Column int
}
type EncodingError struct {
	Name   string
	Offset int
}

func (e *EncodingError) Error() string {
	return fmt.Sprintf("%s: invalid UTF-8 at byte %d", e.Name, e.Offset)
}

type line struct{ start, end int }
type File struct {
	name, text string
	bom        int
	lines      []line
}

func New(name, text string) (*File, error) {
	if !utf8.ValidString(text) {
		for i := 0; i < len(text); {
			r, size := utf8.DecodeRuneInString(text[i:])
			if r == utf8.RuneError && size == 1 {
				return nil, &EncodingError{Name: name, Offset: i}
			}
			i += size
		}
	}
	f := &File{name: name, text: text}
	if strings.HasPrefix(text, "\uFEFF") {
		f.bom = 3
	}
	start := f.bom
	for i := start; i < len(text); i++ {
		if text[i] == '\n' {
			end := i
			if end > start && text[end-1] == '\r' {
				end--
			}
			f.lines = append(f.lines, line{start, end})
			start = i + 1
		}
	}
	f.lines = append(f.lines, line{start, len(text)})
	return f, nil
}
func (f *File) Name() string   { return f.name }
func (f *File) Text() string   { return f.text }
func (f *File) Len() int       { return len(f.text) }
func (f *File) BOMLength() int { return f.bom }
func (f *File) LineCount() int { return len(f.lines) }
func (f *File) LineSpan(zeroBasedLine int) (Span, error) {
	if zeroBasedLine < 0 || zeroBasedLine >= len(f.lines) {
		return Span{}, fmt.Errorf("source line out of range")
	}
	l := f.lines[zeroBasedLine]
	return Span{l.start, l.end}, nil
}
func (f *File) lineAt(offset int) (int, error) {
	if offset < f.bom || offset > len(f.text) {
		return 0, fmt.Errorf("source offset out of range or inside ignored BOM")
	}
	if offset < len(f.text) && !utf8.RuneStart(f.text[offset]) {
		return 0, fmt.Errorf("source offset splits UTF-8 scalar")
	}
	i := sort.Search(len(f.lines), func(i int) bool { return f.lines[i].start > offset }) - 1
	if i < 0 || offset > f.lines[i].end {
		return 0, fmt.Errorf("source offset splits CRLF")
	}
	return i, nil
}
func (f *File) Position(offset int) (Position, error) {
	i, err := f.lineAt(offset)
	if err != nil {
		return Position{}, err
	}
	return Position{Line: i + 1, Column: utf8.RuneCountInString(f.text[f.lines[i].start:offset]) + 1}, nil
}
func (f *File) UTF16Position(offset int) (UTF16Position, error) {
	i, err := f.lineAt(offset)
	if err != nil {
		return UTF16Position{}, err
	}
	column := 0
	for _, r := range f.text[f.lines[i].start:offset] {
		column++
		if r > 0xffff {
			column++
		}
	}
	return UTF16Position{Line: i, Character: column}, nil
}
func (f *File) MapPosition(offset int) (MapPosition, error) {
	p, err := f.UTF16Position(offset)
	if err != nil {
		return MapPosition{}, err
	}
	return MapPosition{Line: p.Line + 1, Column: p.Character}, nil
}
func (f *File) Offset(position UTF16Position) (int, error) {
	if position.Line < 0 || position.Line >= len(f.lines) || position.Character < 0 {
		return 0, fmt.Errorf("source position out of range")
	}
	l := f.lines[position.Line]
	column := 0
	for offset := l.start; offset < l.end; {
		if column == position.Character {
			return offset, nil
		}
		r, size := utf8.DecodeRuneInString(f.text[offset:l.end])
		column++
		if r > 0xffff {
			column++
		}
		if column > position.Character {
			return 0, fmt.Errorf("source position splits UTF-16 surrogate pair")
		}
		offset += size
	}
	if column == position.Character {
		return l.end, nil
	}
	return 0, fmt.Errorf("source column out of range")
}
func (f *File) Validate(span Span) error {
	if span.End < span.Start {
		return fmt.Errorf("reversed source span")
	}
	if _, err := f.lineAt(span.Start); err != nil {
		return err
	}
	_, err := f.lineAt(span.End)
	return err
}
func (f *File) Slice(span Span) (string, error) {
	if err := f.Validate(span); err != nil {
		return "", err
	}
	return f.text[span.Start:span.End], nil
}
