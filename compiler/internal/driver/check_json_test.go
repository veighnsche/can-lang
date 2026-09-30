package driver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf16"
)

func checkJSON(t *testing.T, directory string) (CheckReport, int, []byte) {
	t.Helper()
	raw, exit := CheckProjectJSON(context.Background(), directory)
	var report CheckReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, raw)
	}
	if report.SchemaVersion != CheckSchemaVersion || report.Kind != CheckKind {
		t.Fatalf("bad envelope: %+v", report)
	}
	return report, exit, raw
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckJSONHealthy(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	report, exit, raw := checkJSON(t, root)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0\n%s", exit, raw)
	}
	if report.Status != CheckStatusCompleted || report.Failure != nil {
		t.Fatalf("status = %+v", report)
	}
	result := report.Result
	if result == nil || !result.Accepted {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	for _, stage := range result.Completeness.Stages {
		if stage.State != CheckStageFinished {
			t.Fatalf("stage %+v not finished", stage)
		}
	}
	if len(result.Completeness.Unvisited) != 0 {
		t.Fatalf("unvisited = %+v", result.Completeness.Unvisited)
	}
	// Inputs: every read is identified; the overall digest recomputes.
	byPath := map[string]CheckInputFile{}
	for _, file := range result.Inputs.Files {
		byPath[file.Path] = file
	}
	mainBytes, err := os.ReadFile(filepath.Join(root, "src", "main.can"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(mainBytes)
	entry, ok := byPath["src/main.can"]
	if !ok || entry.Absent || entry.SHA256 == nil || *entry.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("src/main.can entry = %+v", entry)
	}
	var digest strings.Builder
	for _, file := range result.Inputs.Files {
		if file.Absent {
			fmt.Fprintf(&digest, "%s:absent\n", file.Path)
		} else {
			fmt.Fprintf(&digest, "%s:%d:%s\n", file.Path, *file.Length, *file.SHA256)
		}
	}
	recomputed := sha256.Sum256([]byte(digest.String()))
	if got := hex.EncodeToString(recomputed[:]); got != result.Inputs.OverallDigest {
		t.Fatalf("overall digest = %s, recomputed %s", result.Inputs.OverallDigest, got)
	}
	if len(result.Inputs.DeclaredRoots) != 1 || result.Inputs.DeclaredRoots[0].Path != "." {
		t.Fatalf("declared roots = %+v", result.Inputs.DeclaredRoots)
	}
	if result.Inputs.DeclaredRoots[0].Manifest == nil {
		t.Fatal("root manifest identity missing")
	}
	if strings.Contains(string(raw), root) {
		t.Fatal("report leaks the absolute root")
	}
}

func checkRejected(t *testing.T, root string, phase, code string) CheckReport {
	t.Helper()
	report, exit, raw := checkJSON(t, root)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\n%s", exit, raw)
	}
	if report.Status != CheckStatusCompleted || report.Result == nil || report.Result.Accepted {
		t.Fatalf("status = %+v", report)
	}
	if len(report.Result.Diagnostics) == 0 {
		t.Fatal("rejection without diagnostics")
	}
	first := report.Result.Diagnostics[0]
	if first.Phase != phase || first.Code != code {
		t.Fatalf("first diagnostic = %+v, want phase %s code %s", first, phase, code)
	}
	if first.Location == nil || first.File == nil {
		t.Fatalf("first diagnostic lacks source location: %+v", first)
	}
	return report
}

func TestCheckJSONParseError(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	broken := strings.Replace(bridgeMain, "    ok seed\n", "    ok seed seed\n", 1)
	writeFile(t, filepath.Join(root, "src", "main.can"), broken)
	report := checkRejected(t, root, CheckPhaseParse, "syntax")
	if report.Result.Diagnostics[0].Location.StartColumn != 12 {
		t.Fatalf("column = %+v", report.Result.Diagnostics[0].Location)
	}
}

func TestCheckJSONLexError(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	tabbed := strings.Replace(bridgeMain, "    ok seed\n", "\tok seed\n", 1)
	writeFile(t, filepath.Join(root, "src", "main.can"), tabbed)
	checkRejected(t, root, CheckPhaseLex, "CAN-LEX-TAB")
}

func TestCheckJSONUnknownCallee(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	main := strings.Replace(bridgeMain, "ok seed", "ok call nosuchfn(seed)", 1)
	writeFile(t, filepath.Join(root, "src", "main.can"), main)
	report, exit, raw := checkJSON(t, root)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\n%s", exit, raw)
	}
	first := report.Result.Diagnostics[0]
	t.Logf("unknown callee diagnostic: %+v", first)
	if first.Phase != CheckPhaseResolve && first.Phase != CheckPhaseCheck {
		t.Fatalf("phase = %s, want resolve or check", first.Phase)
	}
}

func TestCheckJSONTypeMismatch(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	main := strings.Replace(bridgeMain, "ok seed", `ok "not an int"`, 1)
	writeFile(t, filepath.Join(root, "src", "main.can"), main)
	checkRejected(t, root, CheckPhaseCheck, "CAN-CHECK")
}

func TestCheckJSONWarningsAccepted(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	aliased := strings.Replace(bridgeMain, "    ok seed\n", "    int total = seed + 0\n    ok total\n", 1)
	writeFile(t, filepath.Join(root, "src", "main.can"), aliased)
	report, exit, raw := checkJSON(t, root)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0\n%s", exit, raw)
	}
	if !report.Result.Accepted || len(report.Result.Diagnostics) != 1 {
		t.Fatalf("result = %+v", report.Result)
	}
	warning := report.Result.Diagnostics[0]
	if warning.Phase != CheckPhaseCheck || warning.Code != "CAN-CHECK-UNNECESSARY-LOCAL" || warning.Severity != "warning" {
		t.Fatalf("warning = %+v", warning)
	}
	if warning.Location == nil || warning.File == nil {
		t.Fatalf("warning lacks source location: %+v", warning)
	}
}

func TestCheckJSONMissingRoot(t *testing.T) {
	report, exit, raw := checkJSON(t, filepath.Join(t.TempDir(), "absent"))
	if exit != 2 {
		t.Fatalf("exit = %d, want 2\n%s", exit, raw)
	}
	if report.Status != CheckStatusFailed || report.Result != nil || report.Failure == nil {
		t.Fatalf("status = %+v", report)
	}
	if report.Failure.Kind != CheckFailureIO {
		t.Fatalf("kind = %+v", report.Failure)
	}
}

func TestCheckJSONBadManifest(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	writeFile(t, filepath.Join(root, "can.project.json"), `{"source_root": 42}`)
	checkRejected(t, root, CheckPhaseProject, "CAN-PROJECT-CONFIG")
}

func TestCheckJSONMissingDependency(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	writeFile(t, filepath.Join(root, "can.project.json"), `{"source_root": "src", "error_registry": "can.errors.json", "dependencies": {"dep": "dep"}}`)
	report, exit, raw := checkJSON(t, root)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2\n%s", exit, raw)
	}
	if report.Status != CheckStatusFailed || report.Result != nil || report.Failure == nil {
		t.Fatalf("status = %+v", report)
	}
	if report.Failure.Kind != CheckFailureDependency {
		t.Fatalf("kind = %+v", report.Failure)
	}
}

func TestCheckJSONDeniedRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not deny reads on windows")
	}
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	target := filepath.Join(root, "src", "second.can")
	if err := os.Chmod(target, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(target, 0o644)
	report, exit, raw := checkJSON(t, root)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2\n%s", exit, raw)
	}
	if report.Status != CheckStatusFailed || report.Failure == nil {
		t.Fatalf("status = %+v", report)
	}
	if report.Failure.Kind != CheckFailureIO {
		t.Fatalf("kind = %+v", report.Failure)
	}
}

func TestCheckJSONUnicodeColumns(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	// An astral character precedes the error span on its line: the reported
	// column must count UTF-16 units (12), not runes (11) or bytes (14).
	line := "    ok \"\U0001D11E\" + 1\n"
	broken := strings.Replace(bridgeMain, "    ok seed\n", line, 1)
	writeFile(t, filepath.Join(root, "src", "main.can"), broken)
	report := checkRejected(t, root, CheckPhaseCheck, "CAN-CHECK")
	first := report.Result.Diagnostics[0]
	prefix := "    ok \"\U0001D11E\" "
	want := len(utf16.Encode([]rune(prefix)))
	if want != 12 || len([]rune(prefix)) != 11 || len(prefix) != 14 {
		t.Fatalf("test premise broken: utf16=%d runes=%d bytes=%d", want, len([]rune(prefix)), len(prefix))
	}
	if first.Location.StartColumn != want || first.Location.EndColumn != want+1 {
		t.Fatalf("columns = %+v, want %d-%d", first.Location, want, want+1)
	}
}

func TestCheckJSONRelatedLocations(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	writeFile(t, filepath.Join(root, "can.project.json"), `{"source_root": "src", "assets": {}, "assets": {}}`)
	report := checkRejected(t, root, CheckPhaseProject, "CAN-PROJECT-CONFIG")
	first := report.Result.Diagnostics[0]
	if len(first.Related) == 0 {
		t.Fatalf("related locations missing: %+v", first)
	}
	seen := map[CheckRelated]bool{}
	for _, related := range first.Related {
		if related.File == "" {
			t.Fatalf("related without file: %+v", related)
		}
		if seen[related] {
			t.Fatalf("duplicate related location: %+v", related)
		}
		seen[related] = true
	}
}

func TestCheckJSONUndecodableSource(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	path := filepath.Join(root, "src", "broken.can")
	if err := os.WriteFile(path, []byte{0xff, 0xfe, 0xfd}, 0o644); err != nil {
		t.Fatal(err)
	}
	report, exit, raw := checkJSON(t, root)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\n%s", exit, raw)
	}
	first := report.Result.Diagnostics[0]
	if first.Phase != CheckPhaseLex || first.Location != nil || first.File == nil || first.SpanlessReason == nil {
		t.Fatalf("spanless diagnostic = %+v", first)
	}
}

func TestCheckJSONCancelled(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	raw, exit := CheckProjectJSON(ctx, root)
	var report CheckReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if exit != 2 || report.Status != CheckStatusFailed || report.Failure.Kind != CheckFailureCancelled {
		t.Fatalf("exit = %d report = %s", exit, raw)
	}
}

func TestCheckJSONReportOverflow(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	for i := 0; i < 300; i++ {
		writeFile(t, filepath.Join(root, "src", fmt.Sprintf("bad%03d.can", i)), "package bridge\nfn int broken(\n")
	}
	report, exit, raw := checkJSON(t, root)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2\n%s", exit, raw[:512])
	}
	if report.Status != CheckStatusFailed || report.Result != nil || report.Failure.Kind != CheckFailureIncomplete {
		t.Fatalf("status = %+v", report)
	}
}

func TestCheckJSONCycleRedactsPaths(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	manifest := `{"source_root": "src", "error_registry": "can.errors.json", "dependencies": {"self": "."}}`
	writeFile(t, filepath.Join(root, "can.project.json"), manifest)
	report, exit, raw := checkJSON(t, root)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\n%s", exit, raw)
	}
	if strings.Contains(string(raw), root) {
		t.Fatalf("report leaks the absolute root:\n%s", raw)
	}
	first := report.Result.Diagnostics[0]
	if first.Phase != CheckPhaseProject || first.Location != nil {
		t.Fatalf("cycle diagnostic = %+v", first)
	}
	if first.File == nil || *first.File != "can.project.json" {
		t.Fatalf("cycle file = %+v", first)
	}
}

func TestCheckJSONCompletenessGates(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	writeFile(t, filepath.Join(root, "can.project.json"), `{"source_root": 42}`)
	report, exit, _ := checkJSON(t, root)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1", exit)
	}
	states := map[string]string{}
	for _, stage := range report.Result.Completeness.Stages {
		states[stage.Stage] = stage.State
	}
	if states[CheckPhaseProject] != CheckStageFinished {
		t.Fatalf("stages = %+v", states)
	}
	if states[CheckPhaseResolve] != CheckStageNotEntered || states[CheckPhaseCheck] != CheckStageNotEntered {
		t.Fatalf("resolve/check must not run without a root: %+v", states)
	}
	if len(report.Result.Completeness.Unvisited) != 0 {
		t.Fatalf("unvisited = %+v", report.Result.Completeness.Unvisited)
	}
}
