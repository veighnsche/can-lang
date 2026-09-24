package distribution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// linuxTargetFile mirrors the pinned manifest shape for shape checks. The
// compiled pin stays the authority; this file only guards the pin bytes.
type linuxTargetFile struct {
	SchemaVersion int    `json:"schemaVersion"`
	TargetID      string `json:"targetId"`
	Runtime       struct {
		Name             string `json:"name"`
		Version          string `json:"version"`
		Revision         string `json:"revision"`
		Platform         string `json:"platform"`
		Architecture     string `json:"architecture"`
		MinimumOSVersion string `json:"minimumOSVersion"`
		Executable       string `json:"executable"`
		SHA256           string `json:"sha256"`
	} `json:"runtime"`
	Upstream struct {
		Repository string `json:"repository"`
		Release    string `json:"release"`
		Tag        string `json:"tag"`
		Archive    string `json:"archive"`
		URL        string `json:"url"`
		Size       int64  `json:"size"`
		SHA256     string `json:"sha256"`
		Member     string `json:"member"`
	} `json:"upstream"`
	RequiredNativeAPIs   []string `json:"requiredNativeAPIs"`
	RequiredBehaviorRefs []string `json:"requiredBehaviorProbes"`
}

func loadTargetFile(t *testing.T, name string) linuxTargetFile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(".", name))
	if err != nil {
		t.Fatal(err)
	}
	var parsed linuxTargetFile
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestLinuxTargetPin(t *testing.T) {
	hex64 := regexp.MustCompile(`^[0-9a-f]{64}$`)
	target := loadTargetFile(t, "target-linux-amd64.json")
	if target.SchemaVersion != 1 || target.TargetID != "bun-1.4.2-linux-amd64-v1" {
		t.Fatalf("wrong linux target identity: %+v", target)
	}
	rt := target.Runtime
	if rt.Name != "bun" || rt.Version != "1.4.2" || rt.Revision != "744846f844374847c902b5e7fd59b4342a51ef99" {
		t.Fatalf("wrong linux runtime identity: %+v", rt)
	}
	if rt.Platform != "linux" || rt.Architecture != "amd64" || rt.MinimumOSVersion != "13" {
		t.Fatalf("wrong linux platform contract: %+v", rt)
	}
	if rt.Executable != "runtime/bun" || !hex64.MatchString(rt.SHA256) {
		t.Fatalf("wrong linux executable pin: %+v", rt)
	}
	up := target.Upstream
	if up.Repository != "https://github.com/oven-sh/bun" || up.Tag != "bun-v1.4.2" {
		t.Fatalf("wrong linux upstream identity: %+v", up)
	}
	if up.Archive != "bun-linux-x64.zip" || up.Member != "bun-linux-x64/bun" {
		t.Fatalf("wrong linux archive member: %+v", up)
	}
	if up.URL != "https://github.com/oven-sh/bun/releases/download/bun-v1.4.2/bun-linux-x64.zip" {
		t.Fatalf("linux archive must come from the official release URL: %q", up.URL)
	}
	if up.Size != 36646985 || !hex64.MatchString(up.SHA256) {
		t.Fatalf("wrong linux archive pin: %+v", up)
	}
	// The Linux pin must differ from the macOS pin in every byte-binding
	// field: a copied digest would verify the wrong executable.
	darwin := loadTargetFile(t, "target.json")
	if rt.SHA256 == darwin.Runtime.SHA256 || up.SHA256 == darwin.Upstream.SHA256 {
		t.Fatal("linux pin reuses the darwin archive or executable digest")
	}
	if target.TargetID == darwin.TargetID {
		t.Fatal("linux target reuses the darwin target ID")
	}
	// Same native surface on both hosts: presence checks must not drift.
	if strings.Join(target.RequiredNativeAPIs, "\n") != strings.Join(darwin.RequiredNativeAPIs, "\n") {
		t.Fatal("linux native API surface differs from the darwin pin")
	}
	if strings.Join(target.RequiredBehaviorRefs, "\n") != strings.Join(darwin.RequiredBehaviorRefs, "\n") {
		t.Fatal("linux behavior probes differ from the darwin pin")
	}
}

func TestLinuxProvenanceMatchesPin(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(".", "linux", "provenance.json"))
	if err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		TargetID string `json:"targetId"`
		Upstream struct {
			URL              string `json:"url"`
			Size             int64  `json:"size"`
			SHA256           string `json:"sha256"`
			Member           string `json:"member"`
			ExecutableSHA256 string `json:"executableSHA256"`
		} `json:"upstream"`
		BaseImage struct {
			Ref                 string `json:"ref"`
			IndexDigest         string `json:"indexDigest"`
			Amd64ManifestDigest string `json:"amd64ManifestDigest"`
		} `json:"baseImage"`
		BuildToolchain struct {
			Go struct {
				URL    string `json:"url"`
				SHA256 string `json:"sha256"`
			} `json:"go"`
		} `json:"buildToolchain"`
	}
	if err := json.Unmarshal(raw, &provenance); err != nil {
		t.Fatal(err)
	}
	target := loadTargetFile(t, "target-linux-amd64.json")
	if provenance.TargetID != target.TargetID {
		t.Fatalf("provenance targets %q, pin is %q", provenance.TargetID, target.TargetID)
	}
	if provenance.Upstream.URL != target.Upstream.URL || provenance.Upstream.Size != target.Upstream.Size ||
		provenance.Upstream.SHA256 != target.Upstream.SHA256 || provenance.Upstream.Member != target.Upstream.Member ||
		provenance.Upstream.ExecutableSHA256 != target.Runtime.SHA256 {
		t.Fatalf("provenance archive pin differs from the target file: %+v", provenance.Upstream)
	}
	digestRef := regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	if provenance.BaseImage.Ref != "docker.io/library/debian:13" ||
		!digestRef.MatchString(provenance.BaseImage.IndexDigest) ||
		!digestRef.MatchString(provenance.BaseImage.Amd64ManifestDigest) {
		t.Fatalf("provenance base image is not digest-pinned: %+v", provenance.BaseImage)
	}
	dockerfile, err := os.ReadFile(filepath.Join(".", "linux", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	wantBase := "FROM " + provenance.BaseImage.Ref + "@" + provenance.BaseImage.Amd64ManifestDigest
	if !strings.Contains(string(dockerfile), wantBase) {
		t.Fatalf("Dockerfile does not pin the provenance base image %q", wantBase)
	}
	hex64 := regexp.MustCompile(`^[0-9a-f]{64}$`)
	if !strings.HasPrefix(provenance.BuildToolchain.Go.URL, "https://go.dev/dl/go1.") ||
		!hex64.MatchString(provenance.BuildToolchain.Go.SHA256) {
		t.Fatalf("provenance Go toolchain is not pinned: %+v", provenance.BuildToolchain.Go)
	}
	if !strings.Contains(string(dockerfile), provenance.BuildToolchain.Go.URL) ||
		!strings.Contains(string(dockerfile), provenance.BuildToolchain.Go.SHA256) {
		t.Fatal("Dockerfile does not pin the provenance Go toolchain")
	}
}

func TestHostTargetSelection(t *testing.T) {
	supported := SupportedTargets()
	if len(supported) != 2 || supported[0] == supported[1] {
		t.Fatalf("wrong supported target list: %v", supported)
	}
	target := PinnedTarget()
	found := false
	for _, id := range supported {
		if id == target.TargetID {
			found = true
		}
	}
	if !found {
		t.Fatalf("host target %q is not supported", target.TargetID)
	}
	var reparsed Target
	if err := json.Unmarshal(PinnedTargetJSON(), &reparsed); err != nil {
		t.Fatal(err)
	}
	if reparsed.TargetID != target.TargetID || reparsed.Runtime.SHA256 != target.Runtime.SHA256 {
		t.Fatal("host target bytes do not match the host target")
	}
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		if !HostSupported() || target.Runtime.Platform != "darwin" {
			t.Fatal("darwin host did not select the darwin target")
		}
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		if !HostSupported() || target.Runtime.Platform != "linux" {
			t.Fatal("linux host did not select the linux target")
		}
	default:
		if HostSupported() {
			t.Fatal("unsupported host reports support")
		}
	}
}
