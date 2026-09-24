// Package baseline freezes the T01 dated implementation baseline and the
// agent-comparison fixture registry, and verifies both stay repeatable.
package baseline

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func loadJSON(t *testing.T, path string, dst any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}

func run(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out.String())
	}
	return strings.TrimSpace(out.String())
}

type baselineFile struct {
	Task     string `json:"task"`
	Date     string `json:"date"`
	Compiler struct {
		Revision string `json:"revision"`
	} `json:"compiler"`
	Toolchain struct {
		Bun struct {
			Version  string `json:"version"`
			Revision string `json:"revision"`
		} `json:"bun"`
		Go struct {
			Minimum string `json:"minimum"`
		} `json:"go"`
	} `json:"toolchain"`
}

// The frozen toolchain must match the live environment exactly for Bun
// (pinned) and satisfy the go.mod floor for Go.
func TestBaselineToolchainMatchesFreeze(t *testing.T) {
	root := repoRoot(t)
	var frozen baselineFile
	loadJSON(t, filepath.Join(root, "tests", "baseline", "baseline.json"), &frozen)

	if frozen.Task != "T01" || frozen.Date == "" {
		t.Fatalf("baseline task/date not frozen: %+v", frozen)
	}
	if got := run(t, root, "bun", "--version"); got != frozen.Toolchain.Bun.Version {
		t.Fatalf("bun --version = %q, frozen %q", got, frozen.Toolchain.Bun.Version)
	}
	if got := run(t, root, "bun", "--revision"); got != frozen.Toolchain.Bun.Revision {
		t.Fatalf("bun --revision = %q, frozen %q", got, frozen.Toolchain.Bun.Revision)
	}
	goVer := run(t, root, "go", "version")
	m := regexp.MustCompile(`go1\.(\d+)`).FindStringSubmatch(goVer)
	if m == nil {
		t.Fatalf("unparseable go version: %q", goVer)
	}
	var minor int
	for _, c := range m[1] {
		minor = minor*10 + int(c-'0')
	}
	if minor < 25 {
		t.Fatalf("go version %q below frozen minimum %s", goVer, frozen.Toolchain.Go.Minimum)
	}
	if ok, _ := regexp.MatchString(`^[0-9a-f]{40}$`, frozen.Compiler.Revision); !ok {
		t.Fatalf("compiler revision %q is not a full sha", frozen.Compiler.Revision)
	}
	cmd := exec.Command("git", "merge-base", "--is-ancestor", frozen.Compiler.Revision, "HEAD")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		t.Fatalf("frozen revision %s is not an ancestor of HEAD: %v", frozen.Compiler.Revision, err)
	}
}

type registryFile struct {
	Task  string `json:"task"`
	Cases []struct {
		ID       string `json:"id"`
		Family   string `json:"family"`
		Workload string `json:"workload"`
		Prompts  struct {
			Creation string `json:"creation"`
			Refactor string `json:"refactor"`
			Repair   string `json:"repair"`
		} `json:"prompts"`
		CurrentIdiom struct {
			Fixtures []string `json:"fixtures"`
			Run      string   `json:"run"`
		} `json:"currentIdiom"`
		Candidate struct {
			Status    string `json:"status"`
			OwnerTask string `json:"ownerTask"`
		} `json:"candidate"`
		HeldOut struct {
			ID      string   `json:"id"`
			Rule    string   `json:"rule"`
			Renamed []string `json:"renamed"`
		} `json:"heldOut"`
		HiddenChecks []struct {
			ID     string `json:"id"`
			Kind   string `json:"kind"`
			Expect string `json:"expect"`
		} `json:"hiddenChecks"`
		Trial string `json:"trial"`
	} `json:"cases"`
	Trials map[string]struct {
		Attempts int `json:"attempts"`
		Primary  struct {
			Model  string `json:"model"`
			Effort string `json:"effort"`
		} `json:"primary"`
		Efficient struct {
			Model  string `json:"model"`
			Effort string `json:"effort"`
		} `json:"efficient"`
		Escalation struct {
			Model  string `json:"model"`
			Effort string `json:"effort"`
		} `json:"escalation"`
	} `json:"trials"`
}

// Every registered case must carry creation/refactor/repair prompts, existing
// current-idiom fixtures, a pending candidate slot with an owner, a held-out
// variant, hidden checks and frozen model/effort settings before any trial.
func TestRegistryComplete(t *testing.T) {
	root := repoRoot(t)
	var reg registryFile
	loadJSON(t, filepath.Join(root, "tests", "baseline", "registry.json"), &reg)

	if reg.Task != "T01" {
		t.Fatalf("registry task = %q", reg.Task)
	}
	if len(reg.Cases) == 0 {
		t.Fatal("registry has no cases")
	}
	seen := map[string]bool{}
	for _, c := range reg.Cases {
		if c.ID == "" || seen[c.ID] {
			t.Fatalf("case id missing or duplicated: %q", c.ID)
		}
		seen[c.ID] = true
		if c.Family == "" || c.Workload == "" {
			t.Fatalf("case %s missing family/workload", c.ID)
		}
		if c.Prompts.Creation == "" || c.Prompts.Refactor == "" || c.Prompts.Repair == "" {
			t.Fatalf("case %s missing creation/refactor/repair prompts", c.ID)
		}
		if len(c.CurrentIdiom.Fixtures) == 0 || c.CurrentIdiom.Run == "" {
			t.Fatalf("case %s missing current-idiom fixtures/run", c.ID)
		}
		for _, f := range c.CurrentIdiom.Fixtures {
			if _, err := os.Stat(filepath.Join(root, f)); err != nil {
				t.Fatalf("case %s current fixture %s: %v", c.ID, f, err)
			}
		}
		if c.Candidate.Status != "pending" || c.Candidate.OwnerTask == "" {
			t.Fatalf("case %s candidate slot must be pending with an owner task", c.ID)
		}
		if c.HeldOut.ID == "" || c.HeldOut.Rule == "" || len(c.HeldOut.Renamed) == 0 {
			t.Fatalf("case %s missing held-out variant (id/rule/renames)", c.ID)
		}
		if len(c.HiddenChecks) == 0 {
			t.Fatalf("case %s has no hidden checks", c.ID)
		}
		for _, h := range c.HiddenChecks {
			if h.ID == "" || h.Kind == "" || h.Expect == "" {
				t.Fatalf("case %s has an incomplete hidden check", c.ID)
			}
		}
		trial, ok := reg.Trials[c.Trial]
		if !ok {
			t.Fatalf("case %s references unknown trial %q", c.ID, c.Trial)
		}
		if trial.Attempts != 5 {
			t.Fatalf("case %s trial attempts = %d, want 5", c.ID, trial.Attempts)
		}
		if trial.Primary.Model != "gpt-6-sol" || trial.Primary.Effort != "medium" {
			t.Fatalf("case %s primary = %+v, want gpt-6-sol/medium", c.ID, trial.Primary)
		}
		if trial.Efficient.Model != "gpt-6-luna" || trial.Efficient.Effort != "medium" {
			t.Fatalf("case %s efficient = %+v, want gpt-6-luna/medium", c.ID, trial.Efficient)
		}
		if trial.Escalation.Model != "gpt-6-astra" || trial.Escalation.Effort != "high" {
			t.Fatalf("case %s escalation = %+v, want gpt-6-astra/high", c.ID, trial.Escalation)
		}
	}
}

type smokeReport struct {
	Mode     string `json:"mode"`
	Failures int    `json:"failures"`
	Dir      string `json:"dir"`
	Steps    []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"steps"`
}

// The smoke harness must run in a disposable directory outside the source
// tree and produce a passing report.
func TestHarnessSmokeRepeatable(t *testing.T) {
	root := repoRoot(t)
	script := filepath.Join(root, "tests", "baseline", "run.sh")
	if st, err := os.Stat(script); err != nil || st.Mode()&0o111 == 0 {
		t.Fatalf("harness %s missing or not executable", script)
	}
	cmd := exec.Command("bash", script, "--smoke")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "KEEP_DIR=1")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("run.sh --smoke: %v\n%s", err, out.String())
	}
	m := regexp.MustCompile(`report: (\S+) \(failures=(\d+)\)`).FindStringSubmatch(out.String())
	if m == nil {
		t.Fatalf("no report line in harness output:\n%s", out.String())
	}
	reportPath := m[1]
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(reportPath)) })
	var rep smokeReport
	loadJSON(t, reportPath, &rep)
	if rep.Mode != "smoke" || rep.Failures != 0 {
		t.Fatalf("smoke report not clean: %+v", rep)
	}
	if !strings.HasPrefix(rep.Dir, os.TempDir()) {
		t.Fatalf("harness dir %q is not disposable (want under %s)", rep.Dir, os.TempDir())
	}
	if strings.HasPrefix(rep.Dir, root) {
		t.Fatalf("harness dir %q is inside the source tree", rep.Dir)
	}
	for _, s := range rep.Steps {
		if s.Status != "pass" {
			t.Fatalf("harness step %s status = %s", s.Name, s.Status)
		}
	}
}
