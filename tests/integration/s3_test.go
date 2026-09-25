package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const sqlS3Manifest = `{"source_root":"src","error_registry":"can.errors.json"}`

func writeS3Project(t *testing.T, root, manifest string) {
	t.Helper()
	sourceRoot, _ := filepath.Abs("../..")
	fixture, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/s3/objects.can"))
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", manifest)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", string(fixture))
}

func TestCurrentS3Objects(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged S3 execution")
	}
	endpoint := os.Getenv("CAN_TEST_S3_ENDPOINT")
	region := os.Getenv("CAN_TEST_S3_REGION")
	bucket := os.Getenv("CAN_TEST_S3_BUCKET")
	access := os.Getenv("CAN_TEST_S3_ACCESS_KEY")
	secret := os.Getenv("CAN_TEST_S3_SECRET_KEY")
	if endpoint == "" || region == "" || bucket == "" || access == "" || secret == "" {
		t.Skip("set CAN_TEST_S3_ENDPOINT/REGION/BUCKET/ACCESS_KEY/SECRET_KEY for staged S3 execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeS3Project(t, root, sqlS3Manifest)
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root)
	if status != 0 || diag != "" {
		t.Fatalf("s3 assert: %d %s %s", status, out, diag)
	}
	buildCmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "build", root)
	buildCmd.Dir = home
	buildCmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("live build: %v %s", err, string(buildOut))
	}
	var build struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal(buildOut, &build); err != nil || build.Directory == "" {
		t.Fatalf("invalid live build report %v %s", err, string(buildOut))
	}
	prefix := fmt.Sprintf("s3i/%d-%d/", time.Now().UnixNano(), os.Getpid())
	seed := filepath.Join(sourceRoot, "tests/integration/testdata/s3/payload.bin")
	driver := filepath.Join(sourceRoot, "tests/integration/testdata/s3/s3-driver.ts")
	setupCmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), driver, "setup", seed, prefix)
	setupCmd.Dir = home
	setupCmd.Env = []string{"PATH=/nonexistent", "HOME=" + home, "CAN_TEST_S3_ENDPOINT=" + endpoint, "CAN_TEST_S3_REGION=" + region, "CAN_TEST_S3_BUCKET=" + bucket, "CAN_TEST_S3_ACCESS_KEY=" + access, "CAN_TEST_S3_SECRET_KEY=" + secret}
	setupOut, err := setupCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("live setup: %v %s", err, string(setupOut))
	}
	var setup struct {
		Host   string `json:"host"`
		Bucket string `json:"bucket"`
		Staged string `json:"staged"`
		Size   int    `json:"size"`
	}
	if err := json.Unmarshal(setupOut, &setup); err != nil || setup.Bucket == "" || setup.Size != 8192 {
		t.Fatalf("invalid live setup report %v %s", err, string(setupOut))
	}
	// The compiled CLI reads its credential from the fd-3 environment
	// snapshot, exactly like production entries; secrets never appear
	// in argv or the process environment.
	snapshot := filepath.Join(home, "snapshot.json")
	snap, err := json.Marshal(map[string]string{
		"CAN_TEST_S3_ENDPOINT":   endpoint,
		"CAN_TEST_S3_REGION":     region,
		"CAN_TEST_S3_BUCKET":     bucket,
		"CAN_TEST_S3_ACCESS_KEY": access,
		"CAN_TEST_S3_SECRET_KEY": secret,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshot, snap, 0600); err != nil {
		t.Fatal(err)
	}
	deniedSnap, err := json.Marshal(map[string]string{
		"CAN_TEST_S3_ENDPOINT":   endpoint,
		"CAN_TEST_S3_REGION":     region,
		"CAN_TEST_S3_BUCKET":     bucket,
		"CAN_TEST_S3_ACCESS_KEY": "nope",
		"CAN_TEST_S3_SECRET_KEY": "nope-nope-nope",
	})
	if err != nil {
		t.Fatal(err)
	}
	denied := filepath.Join(home, "denied.json")
	if err := os.WriteFile(denied, deniedSnap, 0600); err != nil {
		t.Fatal(err)
	}
	runWith := func(snap string, args ...string) (int, string, string) {
		t.Helper()
		file, err := os.Open(snap)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{filepath.Join(build.Directory, "entry.ts")}, args...)...)
		cmd.Dir = home
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
		cmd.ExtraFiles = []*os.File{file}
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), stdout.String(), stderr.String()
		}
		return 0, stdout.String(), stderr.String()
	}
	run := func(args ...string) (int, string, string) {
		t.Helper()
		return runWith(snapshot, args...)
	}
	// Best-effort cleanup never fails the test: it runs deferred so
	// even a mid-matrix failure removes only this run's keys.
	created := []string{prefix + "payload.bin"}
	defer func() {
		for _, key := range created {
			func() {
				defer func() { _ = recover() }()
				file, err := os.Open(snapshot)
				if err != nil {
					return
				}
				defer file.Close()
				cmd := exec.CommandContext(context.WithoutCancel(ctx), filepath.Join(bundle, "runtime/bun"), filepath.Join(build.Directory, "entry.ts"), "rm", key)
				cmd.Dir = home
				cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
				cmd.ExtraFiles = []*os.File{file}
				_ = cmd.Run()
			}()
		}
	}()
	track := func(key string) {
		created = append(created, key)
	}
	checks := 0
	ok := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(args...)
		if status != 0 || out != want || diag != "" {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, out, diag)
		}
	}
	fault := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(args...)
		if status != 1 || out != "" || !strings.Contains(diag, `"error":"`+want+`"`) {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, out, diag)
		}
	}
	stat := func(key, size string) {
		t.Helper()
		checks++
		status, out, diag := run("stat", key)
		rest, isStat := strings.CutPrefix(out, size+"@")
		if status != 0 || !isStat || diag != "" {
			t.Fatalf("live stat %s: %d stdout=%q stderr=%s", key, status, out, diag)
		}
		if _, err := strconv.ParseInt(strings.TrimSpace(rest), 10, 64); err != nil {
			t.Fatalf("live stat %s: bad instant %q", key, rest)
		}
	}
	// Signed URLs stay out of failure output: the check asserts shape
	// (key, signature parameter, expiry) and reports lengths only.
	sign := func(method, key string) string {
		t.Helper()
		checks++
		status, out, diag := run("sign", method, key, "300")
		if status != 0 || diag != "" {
			t.Fatalf("live sign %s: %d diag=%s", method, status, diag)
		}
		url := strings.TrimSpace(out)
		if !strings.Contains(url, key) || !strings.Contains(url, "X-Amz-Signature=") || !strings.Contains(url, "X-Amz-Expires=300") {
			t.Fatalf("live sign %s: malformed url len=%d", method, len(url))
		}
		return url
	}
	usage := "usage: put <key> <body> | get <key> | stat <key> | exists <key> | rm <key> | ls <dir> <limit> | lsall <dir> <limit> | sign <method> <key> <expires> | uput <key> <body> | range <key> <offset> <length> | stream <key> | copy <source> <target> | abort <key>"
	ok(nil, usage)
	ok([]string{"bogus"}, usage)
	ok([]string{"put"}, "usage: put <key> <body>")
	// Every command opens its own client: bytes written by one process
	// read back in the next, which proves service persistence across
	// separately opened handles rather than shared state.
	doc := prefix + "doc.txt"
	track(doc)
	ok([]string{"put", doc, "hello"}, "5")
	ok([]string{"get", doc}, "hello")
	stat(doc, "5")
	ok([]string{"exists", doc}, "true")
	ok([]string{"range", doc, "1", "3"}, "ell")
	ok([]string{"stream", doc}, "hello")
	big := strings.Repeat("0123456789abcdef", 16384)
	wide := prefix + "wide.txt"
	track(wide)
	ok([]string{"put", wide, big}, "262144")
	stat(wide, "262144")
	// The small-object get caps at 65536 bytes: the 8KB binary
	// payload downloads and fails UTF-8 decode instead, while the
	// wide object fails the bound without downloading.
	fault([]string{"get", wide}, "s3::over_limit")
	for _, name := range []string{"a.txt", "b.txt", "dir/c.txt"} {
		key := prefix + "ls/" + name
		track(key)
		ok([]string{"put", key, "x"}, "1")
	}
	ok([]string{"ls", prefix + "ls/", "10"}, "3")
	ok([]string{"lsall", prefix + "ls/", "1"}, "3")
	fault([]string{"ls", prefix + "ls/", "abc"}, "text::invalid_number")
	fault([]string{"ls", prefix + "ls/", "0"}, "s3::invalid_config")
	getURL := sign("GET", doc)
	putKey := prefix + "via-url.bin"
	track(putKey)
	putURL := sign("PUT", putKey)
	client := &http.Client{Timeout: 30 * time.Second}
	fetch := func(url string) (int, string) {
		t.Helper()
		checks++
		res, err := client.Get(url)
		if err != nil {
			t.Fatalf("live fetch: %v", err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		if err != nil {
			t.Fatalf("live fetch: %v", err)
		}
		return res.StatusCode, string(body)
	}
	if status, body := fetch(getURL); status != 200 || body != "hello" {
		t.Fatalf("live signed get: %d body len=%d", status, len(body))
	}
	putReq, err := http.NewRequestWithContext(ctx, "PUT", putURL, strings.NewReader("via-url"))
	if err != nil {
		t.Fatal(err)
	}
	putRes, err := client.Do(putReq)
	if err != nil {
		t.Fatalf("live signed put: %v", err)
	}
	putRes.Body.Close()
	checks++
	if putRes.StatusCode != 200 {
		t.Fatalf("live signed put: %d", putRes.StatusCode)
	}
	ok([]string{"get", putKey}, "via-url")
	swap, err := http.NewRequestWithContext(ctx, "PUT", getURL, strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	swapRes, err := client.Do(swap)
	if err != nil {
		t.Fatalf("live method swap: %v", err)
	}
	swapRes.Body.Close()
	checks++
	if swapRes.StatusCode != 403 {
		t.Fatalf("live method swap: %d", swapRes.StatusCode)
	}
	multi := prefix + "multi.bin"
	track(multi)
	ok([]string{"uput", multi, "multipart-body"}, "14")
	ok([]string{"get", multi}, "multipart-body")
	copied := prefix + "copied.bin"
	track(copied)
	ok([]string{"copy", doc, copied}, "5")
	ok([]string{"get", copied}, "hello")
	dropped := prefix + "dropped.bin"
	track(dropped)
	ok([]string{"abort", dropped}, "cancelled")
	ok([]string{"exists", dropped}, "false")
	ok([]string{"rm", doc}, "deleted")
	ok([]string{"exists", doc}, "false")
	fault([]string{"get", prefix + "absent.bin"}, "s3::missing_key")
	fault([]string{"stat", prefix + "absent.bin"}, "s3::missing_key")
	fault([]string{"range", doc, "x", "1"}, "text::invalid_number")
	fault([]string{"put", "", "x"}, "s3::invalid_config")
	fault([]string{"sign", "GET", doc, "0"}, "s3::invalid_config")
	stat(prefix+"payload.bin", "8192")
	fault([]string{"get", prefix + "payload.bin"}, "codec::invalid_data")
	checks++
	status, out, diag = runWith(denied, "stat", multi)
	if status != 1 || out != "" || !strings.Contains(diag, `"error":"s3::access_denied"`) {
		t.Fatalf("live denied: %d stdout=%q stderr=%s", status, out, diag)
	}
	for _, key := range []string{wide, multi, copied, putKey, dropped, prefix + "payload.bin", prefix + "ls/a.txt", prefix + "ls/b.txt", prefix + "ls/dir/c.txt"} {
		ok([]string{"rm", key}, "deleted")
	}
	ok([]string{"ls", prefix, "10"}, "0")
	t.Logf("live s3: %d checks against %s bucket %s", checks, setup.Host, setup.Bucket)
}
