package host

import (
	"runtime"
	"testing"
)

// TestKernelNprocDemo proves the kernel itself refuses fork under a low
// NPROC limit: the trap is host-relative (half the live user count),
// so a pass means the backstop exists here, not merely in the manual.
func TestKernelNprocDemo(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only demonstration")
	}
	d := DemoNprocRefusal()
	t.Log(d.Detail)
	if d.Skip {
		t.Skip("host too quiet to trap fork")
	}
	if !d.Pass {
		t.Fatalf("NPROC refusal not demonstrated: %s", d.Detail)
	}
}

// TestKernelFsizeDemo proves the kernel itself refuses writes past
// FSIZE with zero overshoot: 64 KiB attempted under an 8 KiB cap dies
// on SIGXFSZ and the file never exceeds the cap.
func TestKernelFsizeDemo(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only demonstration")
	}
	d := DemoFsizeRefusal(t.TempDir())
	t.Log(d.Detail)
	if d.Skip {
		t.Skip("demo skipped")
	}
	if !d.Pass {
		t.Fatalf("FSIZE refusal not demonstrated: %s", d.Detail)
	}
}

// TestProbeHostFacts proves the spelled-out rlimit selectors work: a
// wrong selector fails getrlimit and leaves the fact zero, which
// fails closed here instead of admitting on ignorance.
func TestProbeHostFacts(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only probe")
	}
	f := Darwin().Probe(t.TempDir()).Facts
	t.Logf("facts: %+v", f)
	if f.NProcSoft == 0 || f.NProcHard == 0 {
		t.Fatal("NPROC fact unknown: selector wrong or getrlimit failed")
	}
	if f.MemBytes == 0 {
		t.Fatal("hw.memsize unknown")
	}
	if f.TmpFreeBytes == 0 {
		t.Fatal("tmp free disk unknown")
	}
	if f.AddrSpaceCapped || f.FileSizeCapped {
		t.Log("host carries finite AS/FSIZE caps (unusual for darwin)")
	}
}

// TestAddrSpaceProbeRuns records the memory-backstop fact. Either
// outcome is acceptable; the point is the record states what was
// probed, never what was assumed.
func TestAddrSpaceProbeRuns(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only probe")
	}
	settable, detail := ProbeAddrSpaceSettable()
	t.Logf("addr-space settable=%v: %s", settable, detail)
}
