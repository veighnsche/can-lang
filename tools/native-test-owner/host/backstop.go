package host

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// demoTimeout bounds one kernel-refusal demonstration.
const demoTimeout = 15 * time.Second

// DemoResult is the outcome of one live kernel-refusal demonstration.
// Skip reports a host too quiet to demonstrate on (never a failure);
// otherwise Pass decides.
type DemoResult struct {
	Name   string
	Pass   bool
	Skip   bool
	Detail string
}

// execCommand runs a bounded command and returns combined output. A
// nonzero exit is an error carrying the output: refusal demos assert on
// that failure.
func execCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("host: run %s: %w: %s", name, err, strings.TrimSpace(out.String()))
	}
	return out.Bytes(), nil
}

// userProcCount counts the caller's processes. The NPROC demo sizes its
// trap limit from it so the demo is host-relative, not magic.
func userProcCount(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "/bin/ps", "-u", strconv.Itoa(os.Getuid()))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("host: count user processes: %w", err)
	}
	lines := 0
	for _, ln := range strings.Split(out.String(), "\n") {
		if strings.TrimSpace(ln) != "" {
			lines++
		}
	}
	return max(lines-1, 0), nil // minus the header line
}

// DemoNprocRefusal demonstrates the kernel refusing fork under a low
// RLIMIT_NPROC. The limit is set inside the child only (contained: the
// test process and host are unaffected) at half the live user count,
// so the child's first fork must fail with "Resource temporarily
// unavailable" while plenty of margin covers count jitter.
func DemoNprocRefusal() DemoResult {
	const name = "kernel-nproc"
	if runtime.GOOS != "darwin" {
		return DemoResult{Name: name, Skip: true, Detail: "darwin-only demonstration"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), demoTimeout)
	defer cancel()
	count, err := userProcCount(ctx)
	if err != nil {
		return DemoResult{Name: name, Detail: "count user processes: " + err.Error()}
	}
	if count < 20 {
		return DemoResult{Name: name, Skip: true,
			Detail: fmt.Sprintf("host too quiet to trap fork: %d user processes", count)}
	}
	limit := count / 2
	// No exec: the shell must fork for dd, and that fork must fail.
	script := fmt.Sprintf("ulimit -u %d; /bin/dd if=/dev/zero of=/dev/null bs=1024 count=1", limit)
	out, err := execCommand(ctx, "/bin/sh", "-c", script)
	if err == nil {
		return DemoResult{Name: name,
			Detail: fmt.Sprintf("fork succeeded under ulimit -u %d (count %d): expected kernel refusal", limit, count)}
	}
	if !strings.Contains(string(out), "Resource temporarily unavailable") &&
		!strings.Contains(string(out), "fork") {
		return DemoResult{Name: name,
			Detail: fmt.Sprintf("unexpected failure under ulimit -u %d: %s", limit, strings.TrimSpace(string(out)))}
	}
	return DemoResult{Name: name, Pass: true,
		Detail: fmt.Sprintf("fork refused under ulimit -u %d (user count %d): %s",
			limit, count, strings.TrimSpace(string(out)))}
}

// DemoFsizeRefusal demonstrates the kernel refusing writes past
// RLIMIT_FSIZE: dd attempts 64 KiB under an 8 KiB file cap, dies on
// SIGXFSZ, and the file on disk never exceeds the cap (zero overshoot
// past the kernel boundary). dir holds the target file (removed after).
func DemoFsizeRefusal(dir string) DemoResult {
	const name = "kernel-fsize"
	if runtime.GOOS != "darwin" {
		return DemoResult{Name: name, Skip: true, Detail: "darwin-only demonstration"}
	}
	if dir == "" || !filepath.IsAbs(dir) {
		return DemoResult{Name: name, Detail: "demo needs an absolute directory"}
	}
	target := filepath.Join(dir, "fsize-demo.bin")
	defer os.Remove(target)
	ctx, cancel := context.WithTimeout(context.Background(), demoTimeout)
	defer cancel()
	// ulimit -f is in 1024-byte blocks here: 8 blocks = an 8192-byte cap.
	script := "ulimit -f 8; exec /bin/dd if=/dev/zero of=\"$1\" bs=1024 count=64"
	out, err := execCommand(ctx, "/bin/sh", "-c", script, "sh", target)
	if err == nil {
		os.Remove(target)
		return DemoResult{Name: name, Detail: "64 KiB write succeeded under an 8 KiB file cap"}
	}
	fi, serr := os.Stat(target)
	if serr != nil {
		return DemoResult{Name: name,
			Detail: "write refused but target missing: " + strings.TrimSpace(string(out))}
	}
	if fi.Size() <= 0 || fi.Size() > 8192 {
		return DemoResult{Name: name,
			Detail: fmt.Sprintf("target size %d bytes escapes the 8192-byte cap", fi.Size())}
	}
	return DemoResult{Name: name, Pass: true,
		Detail: fmt.Sprintf("write refused at %d/8192 bytes: %s", fi.Size(), strings.TrimSpace(string(out)))}
}

// ProbeAddrSpaceSettable probes whether RLIMIT_AS can even be lowered
// on this host. On darwin arm64 the shared-cache mapping makes any
// practical cap fail with EINVAL, so memory has no kernel backstop and
// the envelope rests on declared-peak ledger plus host-commit fit.
// Informational only: both outcomes are recorded, neither gates.
func ProbeAddrSpaceSettable() (settable bool, detail string) {
	if runtime.GOOS != "darwin" {
		return false, "darwin-only probe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), demoTimeout)
	defer cancel()
	// 1 GiB: if even this fails, no practical AS cap exists here.
	out, err := execCommand(ctx, "/bin/sh", "-c", "ulimit -v 1048576")
	if err != nil {
		return false, "RLIMIT_AS unsettable: " + strings.TrimSpace(string(out))
	}
	return true, "RLIMIT_AS lowerable to 1 GiB"
}
