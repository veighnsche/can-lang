package emit

import (
	"strings"
	"testing"
)

func TestCombineBindingContributionsMergesDisjoint(t *testing.T) {
	combined, err := combineBindingContributions(
		bindingContribution{domain: "core", functions: map[string]string{"a": "x"}},
		bindingContribution{domain: "ai", functions: map[string]string{"b": "y"}},
	)
	if err != nil {
		t.Fatalf("combineBindingContributions returned error: %v", err)
	}
	if len(combined) != 2 || combined["a"] != "x" || combined["b"] != "y" {
		t.Fatalf("unexpected combined bindings: %v", combined)
	}
}

func TestCombineBindingContributionsRejectsDuplicate(t *testing.T) {
	_, err := combineBindingContributions(
		bindingContribution{domain: "core", functions: map[string]string{"dup": "x"}},
		bindingContribution{domain: "ai", functions: map[string]string{"dup": "y"}},
	)
	if err == nil {
		t.Fatal("expected duplicate operation binding error")
	}
	if !strings.Contains(err.Error(), "dup") || !strings.Contains(err.Error(), "core") || !strings.Contains(err.Error(), "ai") {
		t.Fatalf("error names neither identity nor domains: %v", err)
	}
}
