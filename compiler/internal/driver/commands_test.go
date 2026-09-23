package driver

import (
	"strings"
	"testing"
)

func buildEntry(passed bool, pkg, decl, name, reason string, evidence ...string) map[string]any {
	labels := make([]any, 0, len(evidence))
	for _, label := range evidence {
		labels = append(labels, label)
	}
	entry := map[string]any{"passed": passed, "root": map[string]any{"package": pkg, "declaration": decl, "name": name}, "evidence": labels}
	if reason != "" {
		entry["reason"] = reason
	}
	return entry
}

func TestSummarizeBuildRootsRequiresEveryRoot(t *testing.T) {
	summary, err := summarizeBuildRoots([]map[string]any{
		buildEntry(true, "app", "app::first", "sample", "", "real-can"),
		buildEntry(true, "app", "app::second", "sample", "", "supplied-completion", "real-can"),
	})
	if err != nil || summary.Roots != 2 || summary.Passed != 2 || summary.Failed != 0 {
		t.Fatalf("passing suite rejected: %+v %v", summary, err)
	}
	if len(summary.Evidence) != 2 || summary.Evidence[0] != "real-can" || summary.Evidence[1] != "supplied-completion" {
		t.Fatalf("evidence not deduped/sorted: %v", summary.Evidence)
	}
	summary, err = summarizeBuildRoots(nil)
	if err != nil || summary.Roots != 0 || summary.Passed != 0 {
		t.Fatalf("empty suite rejected: %+v %v", summary, err)
	}
	_, err = summarizeBuildRoots([]map[string]any{
		buildEntry(true, "app", "app::first", "sample", "", "real-can"),
		buildEntry(false, "app", "app::answer", "wrong", "outcome mismatch", "real-can"),
	})
	if err == nil || !strings.Contains(err.Error(), "1 of 2 roots passed") || !strings.Contains(err.Error(), "app/app::answer/wrong (outcome mismatch)") {
		t.Fatalf("failing root misreported: %v", err)
	}
	many := []map[string]any{}
	for i := 0; i < 10; i++ {
		many = append(many, buildEntry(false, "app", "app::answer", "wrong", "outcome mismatch"))
	}
	_, err = summarizeBuildRoots(many)
	if err == nil || !strings.Contains(err.Error(), "and 2 more") {
		t.Fatalf("failure list unbounded: %v", err)
	}
	_, err = summarizeBuildRoots([]map[string]any{{"passed": false}})
	if err == nil || !strings.Contains(err.Error(), "0 of 1 roots passed") {
		t.Fatalf("malformed entry passed: %v", err)
	}
}
