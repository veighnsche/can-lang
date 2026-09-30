package source

import (
	"errors"
	"fmt"
	"testing"
)

func TestLocateJoinedIndependentProvenance(t *testing.T) {
	first := Locate("inner.can", Span{2, 3}, errors.New("first"))
	blocked := &BlockedError{Dependency: "missing"}
	joined := errors.Join(first, errors.New("second"), blocked)
	located := LocateCode("outer.can", Span{7, 9}, "OUTER", fmt.Errorf("context: %w", joined))
	wrapped := located.(interface{ Unwrap() error }).Unwrap()
	children := wrapped.(interface{ Unwrap() []error }).Unwrap()
	a, _ := AsLocated(children[0])
	b, _ := AsLocated(children[1])
	if a.File != "inner.can" || a.Span != (Span{2, 3}) || b.File != "outer.can" || b.Span != (Span{7, 9}) || b.Code != "OUTER" {
		t.Fatalf("lost per-leaf provenance: %+v %+v", a, b)
	}
	if !IsBlocked(children[2]) || IsBlocked(located) {
		t.Fatal("blocked sibling swallowed direct findings")
	}
	if located.Error() != fmt.Errorf("context: %w", joined).Error() {
		t.Fatal("CLI context lost")
	}
}
