package source

import (
	"errors"
	"testing"
	"unicode/utf8"
)

func TestOriginalBytesAndCoordinateConventions(t *testing.T) {
	text := "\uFEFFa😀e\u0301\r\nz\tq\n"
	file, err := New("unicode.can", text)
	if err != nil {
		t.Fatal(err)
	}
	if file.Text() != text || file.BOMLength() != 3 || file.LineCount() != 3 {
		t.Fatal("source changed")
	}
	for _, row := range []struct{ offset, line, scalar, utf16 int }{{3, 1, 1, 0}, {4, 1, 2, 1}, {8, 1, 3, 3}, {9, 1, 4, 4}, {11, 1, 5, 5}, {13, 2, 1, 0}, {15, 2, 3, 2}, {17, 3, 1, 0}} {
		human, err := file.Position(row.offset)
		if err != nil || human != (Position{Line: row.line, Column: row.scalar}) {
			t.Fatalf("human %d: %+v %v", row.offset, human, err)
		}
		editor, err := file.UTF16Position(row.offset)
		if err != nil || editor != (UTF16Position{Line: row.line - 1, Character: row.utf16}) {
			t.Fatalf("editor %d: %+v %v", row.offset, editor, err)
		}
		mapping, err := file.MapPosition(row.offset)
		if err != nil || mapping != (MapPosition{Line: row.line, Column: row.utf16}) {
			t.Fatalf("mapping %d: %+v %v", row.offset, mapping, err)
		}
		offset, err := file.Offset(editor)
		if err != nil || offset != row.offset {
			t.Fatalf("round trip %d: %d %v", row.offset, offset, err)
		}
	}
	got, err := file.Slice(Span{Start: 3, End: 11})
	if err != nil || got != "a😀e\u0301" {
		t.Fatalf("slice %q %v", got, err)
	}
	if line, err := file.LineSpan(0); err != nil || line != (Span{Start: 3, End: 11}) {
		t.Fatalf("line span %+v %v", line, err)
	}
}
func TestRejectSplitAndOutOfRangeCoordinates(t *testing.T) {
	file, _ := New("positions.can", "\uFEFFa😀\r\nz")
	for _, offset := range []int{-1, 0, 1, 2, 5, 6, 7, 9, 99} {
		if _, err := file.UTF16Position(offset); err == nil {
			t.Errorf("accepted offset %d", offset)
		}
	}
	for _, position := range []UTF16Position{{Line: -1}, {Line: 2}, {Character: -1}, {Character: 2}, {Character: 4}, {Line: 1, Character: 2}} {
		if _, err := file.Offset(position); err == nil {
			t.Errorf("accepted position %+v", position)
		}
	}
	if _, err := file.Slice(Span{Start: 8, End: 4}); err == nil {
		t.Fatal("accepted reversed span")
	}
	if _, err := file.LineSpan(-1); err == nil {
		t.Fatal("accepted negative line")
	}
}
func TestInvalidUTF8HasOriginalOffset(t *testing.T) {
	for _, input := range []string{"a\xff", "a\xf0\x9f", "a\xed\xa0\x80"} {
		_, err := New("bad.can", input)
		var encoding *EncodingError
		if !errors.As(err, &encoding) || encoding.Offset != 1 {
			t.Fatalf("unexpected encoding error: %v", err)
		}
	}
	if _, err := New("replacement.can", "�"); err != nil {
		t.Fatal("valid replacement scalar rejected")
	}
}
func FuzzCoordinates(f *testing.F) {
	for _, s := range []string{"", "a😀e\u0301\r\nx\n", "\uFEFFabc", "x\rinside", "\xff"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 8192 {
			t.Skip()
		}
		file, err := New("fuzz.can", text)
		if !utf8.ValidString(text) {
			if err == nil {
				t.Fatal("invalid UTF-8 accepted")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		for offset := file.BOMLength(); offset <= len(text); offset++ {
			position, err := file.UTF16Position(offset)
			if err != nil {
				continue
			}
			roundTrip, err := file.Offset(position)
			if err != nil || roundTrip != offset {
				t.Fatalf("round trip %d -> %+v -> %d: %v", offset, position, roundTrip, err)
			}
		}
	})
}
