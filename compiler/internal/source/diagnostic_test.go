package source

import (
	"errors"
	"fmt"
	"testing"
)

func TestLocateKeepsInnermost(t *testing.T) {
	inner := Locate("a.can", Span{Start: 10, End: 20}, errors.New("boom"))
	outer := Locate("a.can", Span{Start: 0, End: 100}, fmt.Errorf("wrap: %w", inner))
	located, ok := AsLocated(outer)
	if !ok {
		t.Fatal("located failure lost through wrapping")
	}
	if located.File != "a.can" || located.Span.Start != 10 || located.Span.End != 20 {
		t.Fatalf("outer span shadowed inner: %+v", located)
	}
	if outer.Error() != "wrap: boom" {
		t.Fatalf("message changed: %q", outer.Error())
	}
}

func TestLocateSkipsUnlocated(t *testing.T) {
	if err := Locate("a.can", Span{Start: 1, End: 2}, nil); err != nil {
		t.Fatalf("nil error gained a location: %v", err)
	}
	plain := errors.New("boom")
	if Locate("", Span{Start: 1, End: 2}, plain) != plain {
		t.Fatal("empty file gained a location")
	}
	if _, ok := AsLocated(plain); ok {
		t.Fatal("plain error reports a location")
	}
}

func TestRelateAttachesSecondary(t *testing.T) {
	err := Relate("b.can", Span{Start: 3, End: 4}, "origin", Locate("a.can", Span{Start: 1, End: 2}, errors.New("boom")))
	located, ok := AsLocated(err)
	if !ok || len(located.Related) != 1 {
		t.Fatalf("related span missing: %+v", located)
	}
	related := located.Related[0]
	if related.File != "b.can" || related.Span.Start != 3 || related.Note != "origin" {
		t.Fatalf("related span wrong: %+v", related)
	}
	if plain := errors.New("boom"); Relate("b.can", Span{}, "x", plain) != plain {
		t.Fatal("unlocated error gained a related span")
	}
}
