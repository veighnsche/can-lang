package source

import "testing"

func TestEOFInsertionAndMultilineRanges(t *testing.T) {
	text := "first 😀\nsecond e\u0301\n"
	file, err := New("eof.can", text)
	if err != nil {
		t.Fatal(err)
	}
	// Zero-width insertion at EOF converts and round-trips.
	eof, err := file.UTF16Position(len(text))
	if err != nil {
		t.Fatal(err)
	}
	if eof != (UTF16Position{Line: 2, Character: 0}) {
		t.Fatalf("eof = %+v", eof)
	}
	if back, err := file.Offset(eof); err != nil || back != len(text) {
		t.Fatalf("eof round trip = %d, %v", back, err)
	}
	if err := file.Validate(Span{Start: len(text), End: len(text)}); err != nil {
		t.Fatalf("eof insertion rejected: %v", err)
	}
	// Zero-width insertion mid-line converts without widening.
	mid, err := file.UTF16Position(4)
	if err != nil {
		t.Fatal(err)
	}
	if mid != (UTF16Position{Line: 0, Character: 4}) {
		t.Fatalf("mid = %+v", mid)
	}
	if err := file.Validate(Span{Start: 4, End: 4}); err != nil {
		t.Fatalf("mid insertion rejected: %v", err)
	}
	if back, err := file.Offset(mid); err != nil || back != 4 {
		t.Fatalf("mid round trip = %d, %v", back, err)
	}
	// Multiline span preserves both end lines: "😀\nsecond" crosses lines 0-1.
	span := Span{Start: 6, End: 17}
	start, err := file.UTF16Position(span.Start)
	if err != nil {
		t.Fatal(err)
	}
	end, err := file.UTF16Position(span.End)
	if err != nil {
		t.Fatal(err)
	}
	if start.Line == end.Line {
		t.Fatalf("multiline collapsed: %+v -> %+v", start, end)
	}
	if start != (UTF16Position{Line: 0, Character: 6}) {
		t.Fatalf("multiline start = %+v", start)
	}
	if end != (UTF16Position{Line: 1, Character: 6}) {
		t.Fatalf("multiline end = %+v", end)
	}
	// Empty file: EOF insertion at 0,0.
	empty, err := New("empty.can", "")
	if err != nil {
		t.Fatal(err)
	}
	zero, err := empty.UTF16Position(0)
	if err != nil || zero != (UTF16Position{Line: 0, Character: 0}) {
		t.Fatalf("empty eof = %+v, %v", zero, err)
	}
}

func TestMalformedBareCRUsesEditorLineCoordinates(t *testing.T) {
	file, err := New("invalid.can", "a\r𐐀b\r\nc\n")
	if err != nil {
		t.Fatal(err)
	}
	if file.LineCount() != 4 {
		t.Fatalf("line count %d", file.LineCount())
	}
	for _, offset := range []int{0, 1, 2, 6, 7, 9, 10, 11} {
		position, err := file.UTF16Position(offset)
		if err != nil {
			t.Fatal(err)
		}
		roundtrip, err := file.Offset(position)
		if err != nil || roundtrip != offset {
			t.Fatalf("%d => %+v => %d: %v", offset, position, roundtrip, err)
		}
	}
	if position, err := file.UTF16Position(6); err != nil || position != (UTF16Position{Line: 1, Character: 2}) {
		t.Fatalf("position %+v %v", position, err)
	}
	if _, err := file.UTF16Position(8); err == nil {
		t.Fatal("CRLF interior accepted")
	}
}
