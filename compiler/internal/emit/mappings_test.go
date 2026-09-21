package emit

import (
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"strings"
	"testing"
)

func TestMappingAssemblyPreservesUTF16AndCannotAdmitQuotedMarkers(t *testing.T) {
	span := source.Span{Start: 4, End: 12}
	token := mappingMark("can.project.root/p/main.can", span, "call")
	text := "const text = " + quote(token) + ";\n\"😀\";" + token + "invoke();\n"
	code, segments, err := extractMappings(text)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(code, 0) || len(segments) != 1 || segments[0].Line != 2 || segments[0].Column != 5 || segments[0].Start != 4 || segments[0].End != 12 {
		t.Fatalf("%q %+v", code, segments)
	}
	if !strings.Contains(code, `\u0000`) {
		t.Fatal("quoted authored token was consumed")
	}
	if _, _, err := extractMappings("\x00bad"); err == nil {
		t.Fatal("unfinished token admitted")
	}
}
