package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	journal "github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

func testOwner(t *testing.T, j *journal.Journal) *Owner {
	t.Helper()
	if j == nil {
		j = journal.OpenMemory()
	}
	return New(j, os.Getpid(), "test-start-token")
}

func testBudgets() Budgets {
	return Budgets{MaxBytes: 1 << 20, MaxEntries: 1024, MaxDepth: 16}
}

func mustReserve(t *testing.T, o *Owner, opID string, b Budgets) Handle {
	t.Helper()
	h, err := o.Reserve(opID, t.TempDir(), b, "test", "", 0)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	return h
}

func TestHappyPathSealAndDispose(t *testing.T) {
	jdir := t.TempDir()
	j, err := journal.OpenDir(jdir)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	o := testOwner(t, j)
	h := mustReserve(t, o, "op.happy.reserve", testBudgets())

	// Intent is journaled before effects: the reservation is discoverable.
	if _, ok := j.Lookup("op.happy.reserve"); !ok {
		t.Fatal("reservation intent not journaled")
	}

	res, err := o.Materialize("op.happy.mat", h, []MaterialEntry{
		{Path: "inputs/main.can", Data: []byte("main")},
		{Path: "inputs/lib/util.can", Data: []byte("util")},
	}, ModeCreate)
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if !res.Complete || len(res.Outcomes) != 2 {
		t.Fatalf("unexpected materialize result: %+v", res)
	}

	rd, err := o.Read(h, "inputs/main.can", 0, 64, nil, "")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(rd.Data) != "main" || rd.Stability != StabilityScopedRevision {
		t.Fatalf("unexpected read: %q %s", rd.Data, rd.Stability)
	}
	if rd.Truncated {
		t.Fatal("small read must not truncate")
	}

	st, err := o.Stat(h, "inputs/lib")
	if err != nil || st.Kind != KindDir {
		t.Fatalf("Stat dir: %v %+v", err, st)
	}

	page, err := o.List(h, "inputs", ListCursor{Revision: rd.Revision}, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !page.Complete || page.TotalCount != 2 {
		t.Fatalf("unexpected list page: %+v", page)
	}

	wr, err := o.Write("op.happy.write", h, "scratch/out.txt", []byte("out"), ModeCreate, 0)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if wr.Revision <= rd.Revision {
		t.Fatal("write must advance the revision")
	}

	// Capability-mediated mutation invalidates pagination cursors.
	if _, err := o.List(h, "inputs", ListCursor{Revision: rd.Revision}, 100); !errors.Is(err, ErrStaleCursor) {
		t.Fatalf("stale cursor: want ErrStaleCursor, got %v", err)
	}

	seal, err := o.Seal("op.happy.seal", h, SealScope{Full: true})
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if seal.Stability != StabilityScopedRevision {
		t.Fatalf("seal stability: got %s", seal.Stability)
	}
	for i := 1; i < len(seal.Entries); i++ {
		if seal.Entries[i-1].Path >= seal.Entries[i].Path {
			t.Fatal("seal inventory not sorted")
		}
	}

	if _, err := o.Write("op.happy.sealed", h, "scratch/no.txt", []byte("x"), ModeCreate, 0); !errors.Is(err, ErrSealed) {
		t.Fatalf("write after full seal: want ErrSealed, got %v", err)
	}

	dr, err := o.Dispose("op.happy.dispose", h)
	if err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if !dr.Complete || len(dr.Remaining) != 0 {
		t.Fatalf("unexpected dispose receipt: %+v", dr)
	}

	// Repeats join the terminal receipt, under the same or a new ID.
	dr2, err := o.Dispose("op.happy.dispose", h)
	if err != nil || !dr2.Joined || !dr2.Complete {
		t.Fatalf("dispose rejoin: %+v %v", dr2, err)
	}
	dr3, err := o.Dispose("op.happy.dispose.2", h)
	if err != nil || !dr3.Joined || !dr3.Complete {
		t.Fatalf("dispose second ID join: %+v %v", dr3, err)
	}

	root, _ := o.RootForTesting(h)
	if _, serr := os.Lstat(root); !os.IsNotExist(serr) {
		t.Fatal("root must be gone after clean disposal")
	}
	if pend := j.Pending(); len(pend) != 0 {
		t.Fatalf("journal should drain on clean disposal, pending: %v", pend)
	}
}

func TestMaterializePartialKeepsLandedEntriesCharged(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.partial.reserve", testBudgets())

	// "x" lands as a file; "x/y" then fails because its parent is a file.
	res, err := o.Materialize("op.partial.mat", h, []MaterialEntry{
		{Path: "x", Data: []byte("first")},
		{Path: "x/y", Data: []byte("second")},
	}, ModeCreate)
	if err == nil || res == nil {
		t.Fatalf("want partial failure with outcomes, got %+v %v", res, err)
	}
	if res.Complete {
		t.Fatal("partial publication must not claim completeness")
	}
	if len(res.Outcomes) != 2 || res.Outcomes[0].Error != "" || res.Outcomes[1].Error == "" {
		t.Fatalf("unexpected outcomes: %+v", res.Outcomes)
	}
	// Landed entries remain owned, readable and charged.
	rd, rerr := o.Read(h, "x", 0, 64, nil, "")
	if rerr != nil || string(rd.Data) != "first" {
		t.Fatalf("landed entry not retained: %+v %v", rd, rerr)
	}
	nbytes, entries, _, cerr := o.Charges(h)
	if cerr != nil || nbytes < int64(len("first")) || entries < 3 {
		t.Fatalf("partial allocation must stay charged: %d bytes %d entries %v", nbytes, entries, cerr)
	}
	// Repeating the op joins the same partial fact instead of redispatching.
	res2, err := o.Materialize("op.partial.mat", h, []MaterialEntry{
		{Path: "x", Data: []byte("first")},
		{Path: "x/y", Data: []byte("second")},
	}, ModeCreate)
	if err == nil || res2 == nil || !res2.Joined {
		t.Fatalf("partial rejoin: %+v %v", res2, err)
	}
}

func TestSymlinkReplacementNeverFollowed(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.link.reserve", testBudgets())
	if _, err := o.Materialize("op.link.mat", h, []MaterialEntry{{Path: "f", Data: []byte("real")}}, ModeCreate); err != nil {
		t.Fatal(err)
	}
	root, _ := o.RootForTesting(h)

	// Out-of-band: replace the file with a link to an outside secret.
	secret := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secret, []byte("SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "f")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "f")); err != nil {
		t.Fatal(err)
	}
	rd, err := o.Read(h, "f", 0, 64, nil, "")
	if !errors.Is(err, ErrWrongKind) {
		t.Fatalf("read through replaced link: want ErrWrongKind, got %v", err)
	}
	if rd == nil || rd.Kind != KindLink || strings.Contains(string(rd.Data), "SECRET") {
		t.Fatalf("link must be inspected, never followed: %+v", rd)
	}
	st, err := o.Stat(h, "f")
	if err != nil || st.Kind != KindLink || st.Target != secret {
		t.Fatalf("stat link: %+v %v", st, err)
	}
	if st.Stability != StabilityChangedObserved && st.Stability != StabilityUnknown {
		t.Fatalf("replaced link stability: %s", st.Stability)
	}
	// The workspace stays owned and charged; disposal still works because
	// removal never follows links either.
	if _, err := o.Dispose("op.link.dispose", h); err != nil {
		t.Fatalf("Dispose after link replacement: %v", err)
	}
	if _, serr := os.Lstat(secret); serr != nil {
		t.Fatal("disposal must not touch the link target")
	}
}

func TestReplacedRootIsNeverAuthority(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.repl.reserve", testBudgets())
	if _, err := o.Materialize("op.repl.mat", h, []MaterialEntry{{Path: "f", Data: []byte("real")}}, ModeCreate); err != nil {
		t.Fatal(err)
	}
	root, _ := o.RootForTesting(h)
	evilParent := t.TempDir()
	evil := filepath.Join(evilParent, "evil")
	if err := os.Mkdir(evil, 0o700); err != nil {
		t.Fatal(err)
	}
	stash := root + ".stashed"
	if err := os.Rename(root, stash); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(stash)
	if err := os.Symlink(evil, root); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(root)

	if _, err := o.Read(h, "f", 0, 64, nil, ""); !errors.Is(err, ErrReplaced) {
		t.Fatalf("read via replaced root: want ErrReplaced, got %v", err)
	}
	dr, err := o.Dispose("op.repl.dispose", h)
	if !errors.Is(err, ErrReplaced) {
		t.Fatalf("dispose of replaced root: want ErrReplaced, got %v", err)
	}
	if dr == nil || len(dr.Remaining) == 0 {
		t.Fatalf("replaced root must stay unresolved: %+v", dr)
	}
	rep, err := o.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Outcomes) != 1 || rep.Outcomes[0].Action != "foreign" || len(rep.Outcomes[0].Remaining) == 0 {
		t.Fatalf("recover must report the replaced root: %+v", rep)
	}
	if _, serr := os.Lstat(evil); serr != nil {
		t.Fatal("recovery must not touch the replacement target")
	}
}

func TestActiveAndForeignLeasesBlockDisposal(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.lease.reserve", testBudgets())
	future := time.Now().Add(time.Hour).UnixMilli()
	past := time.Now().Add(-time.Hour).UnixMilli()

	if _, err := o.GrantLease("op.lease.grant", h, "reader-1", os.Getpid(), "test-start-token", future); err != nil {
		t.Fatal(err)
	}
	dr, err := o.Dispose("op.lease.dispose.1", h)
	if !errors.Is(err, ErrActiveLease) {
		t.Fatalf("dispose under active lease: want ErrActiveLease, got %v", err)
	}
	if dr == nil || len(dr.Remaining) == 0 {
		t.Fatalf("active lease must stay unresolved: %+v", dr)
	}
	// The workspace is still live and usable while the lease blocks.
	if _, err := o.Write("op.lease.write", h, "w.txt", []byte("w"), ModeCreate, 0); err != nil {
		t.Fatalf("workspace must stay live under active lease: %v", err)
	}
	rep, err := o.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Outcomes) != 1 || rep.Outcomes[0].Action != "active" {
		t.Fatalf("recover must report active work untouched: %+v", rep)
	}

	if err := o.ReleaseLease("op.lease.release", h, "reader-1", os.Getpid(), "test-start-token"); err != nil {
		t.Fatal(err)
	}
	// An expired dependent lease retires quietly at disposal.
	if _, err := o.GrantLease("op.lease.grant.2", h, "stale", os.Getpid(), "test-start-token", past); err != nil {
		t.Fatal(err)
	}
	dr, err = o.Dispose("op.lease.dispose.2", h)
	if err != nil || !dr.Complete {
		t.Fatalf("dispose after release: %+v %v", dr, err)
	}
}

func TestBudgetsRejectBeforeRetention(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.budget.reserve", Budgets{MaxBytes: 100, MaxEntries: 8, MaxDepth: 4})

	if _, err := o.Materialize("op.budget.mat.1", h, []MaterialEntry{{Path: "big", Data: make([]byte, 101)}}, ModeCreate); !errors.Is(err, ErrBudget) {
		t.Fatalf("byte budget: want ErrBudget, got %v", err)
	}
	if _, err := o.Stat(h, "big"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("rejected bytes must not be retained: %v", err)
	}
	if _, err := o.Materialize("op.budget.mat.2", h, []MaterialEntry{{Path: "a/b/c/d/e", Data: []byte("x")}}, ModeCreate); !errors.Is(err, ErrBudget) {
		t.Fatalf("depth budget: want ErrBudget, got %v", err)
	}
	many := make([]MaterialEntry, 0, 10)
	for i := 0; i < 10; i++ {
		many = append(many, MaterialEntry{Path: "f" + string(rune('0'+i)), Data: []byte("x")})
	}
	if _, err := o.Materialize("op.budget.mat.3", h, many, ModeCreate); !errors.Is(err, ErrBudget) {
		t.Fatalf("entry budget: want ErrBudget, got %v", err)
	}
	// Prior charges are unchanged by the rejections.
	nbytes, entries, _, err := o.Charges(h)
	if err != nil || nbytes != 0 || entries != 2 { // root + staging
		t.Fatalf("rejected ops must not move charges: %d bytes %d entries %v", nbytes, entries, err)
	}
	if _, err := o.Dispose("op.budget.dispose", h); err != nil {
		t.Fatal(err)
	}
}

func TestStagingBytesAreCharged(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.stage.reserve", Budgets{MaxBytes: 100, MaxEntries: 64, MaxDepth: 8})
	if _, err := o.Write("op.stage.w1", h, "a", make([]byte, 90), ModeCreate, 0); err != nil {
		t.Fatal(err)
	}
	// Final size 95 would fit, but staging needs 90 + 95 transiently.
	if _, err := o.Write("op.stage.w2", h, "a", make([]byte, 95), ModeReplace, 0); !errors.Is(err, ErrBudget) {
		t.Fatalf("staging budget: want ErrBudget, got %v", err)
	}
	rd, err := o.Read(h, "a", 0, MaxReadBytes, nil, "")
	if err != nil || len(rd.Data) != 90 {
		t.Fatalf("failed replacement must keep prior bytes: %+v %v", rd, err)
	}
	// A replacement that fits including staging succeeds atomically.
	if _, err := o.Write("op.stage.w3", h, "a", make([]byte, 10), ModeReplace, 0); err != nil {
		t.Fatal(err)
	}
	nbytes, _, _, _ := o.Charges(h)
	if nbytes != 10 {
		t.Fatalf("charges after replace: got %d bytes", nbytes)
	}
	if _, err := o.Dispose("op.stage.dispose", h); err != nil {
		t.Fatal(err)
	}
}

func TestFailedDeletionStaysUnresolved(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-based deletion failure needs a non-root user")
	}
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.rmfail.reserve", testBudgets())
	if _, err := o.Materialize("op.rmfail.mat", h, []MaterialEntry{{Path: "locked/f", Data: []byte("x")}}, ModeCreate); err != nil {
		t.Fatal(err)
	}
	root, _ := o.RootForTesting(h)
	if err := os.Chmod(filepath.Join(root, "locked"), 0o555); err != nil {
		t.Fatal(err)
	}
	dr, err := o.Dispose("op.rmfail.dispose.1", h)
	if !errors.Is(err, ErrUnresolved) {
		t.Fatalf("failed deletion: want ErrUnresolved, got %v", err)
	}
	if dr == nil || dr.Complete || len(dr.Remaining) == 0 || len(dr.Errors) == 0 {
		t.Fatalf("failed deletion must report remaining paths/errors: %+v", dr)
	}
	if _, _, _, cerr := o.Charges(h); cerr != nil {
		t.Fatalf("unresolved workspace must stay charged: %v", cerr)
	}
	// A later safe cleanup retries (incomplete receipts never join) and
	// preserves the earlier failure in its own outcome.
	if err := os.Chmod(filepath.Join(root, "locked"), 0o755); err != nil {
		t.Fatal(err)
	}
	dr2, err := o.Dispose("op.rmfail.dispose.2", h)
	if err != nil || !dr2.Complete || dr2.Joined {
		t.Fatalf("retry after repair: %+v %v", dr2, err)
	}
}

func TestSealStrengthDegradesOnOutOfBandChange(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.seal.reserve", testBudgets())
	if _, err := o.Materialize("op.seal.mat", h, []MaterialEntry{{Path: "in/a.txt", Data: []byte("v1")}}, ModeCreate); err != nil {
		t.Fatal(err)
	}
	root, _ := o.RootForTesting(h)
	// Out-of-band mutation of tracked content.
	if err := os.WriteFile(filepath.Join(root, "in/a.txt"), []byte("v2!!"), 0o600); err != nil {
		t.Fatal(err)
	}
	rd, err := o.Read(h, "in/a.txt", 0, 64, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if rd.Stability != StabilityChangedObserved {
		t.Fatalf("changed content stability: got %s", rd.Stability)
	}
	seal, err := o.Seal("op.seal.seal.1", h, SealScope{Full: true})
	if err != nil {
		t.Fatal(err)
	}
	if seal.Stability != StabilityChangedObserved {
		t.Fatalf("seal after change: got %s", seal.Stability)
	}
	if _, err := o.Read(h, "in/a.txt", 0, 64, nil, StabilityQualifiedSnapshot); !errors.Is(err, ErrStrength) {
		t.Fatalf("qualified snapshot: want ErrStrength, got %v", err)
	}
}

func TestSealUnknownOnUntrackedFile(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.unk.reserve", testBudgets())
	if _, err := o.Materialize("op.unk.mat", h, []MaterialEntry{{Path: "in/a.txt", Data: []byte("v1")}}, ModeCreate); err != nil {
		t.Fatal(err)
	}
	root, _ := o.RootForTesting(h)
	// Out-of-band addition: never capability-tracked.
	if err := os.WriteFile(filepath.Join(root, "in/dropped.txt"), []byte("?"), 0o600); err != nil {
		t.Fatal(err)
	}
	seal, err := o.Seal("op.unk.seal", h, SealScope{Full: true})
	if err != nil {
		t.Fatal(err)
	}
	if seal.Stability != StabilityUnknown {
		t.Fatalf("seal with untracked file: got %s", seal.Stability)
	}
	page, err := o.List(h, "in", ListCursor{Revision: seal.Revision}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if page.Complete {
		t.Fatal("unknown stability must not claim a complete inventory")
	}
}

func TestPartialSealScope(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.pseal.reserve", testBudgets())
	if _, err := o.Materialize("op.pseal.mat", h, []MaterialEntry{
		{Path: "inputs/a.txt", Data: []byte("a")},
		{Path: "scratch/b.txt", Data: []byte("b")},
	}, ModeCreate); err != nil {
		t.Fatal(err)
	}
	seal, err := o.Seal("op.pseal.seal", h, SealScope{Paths: []string{"inputs", "inputs/missing.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(seal.Absent) != 1 || seal.Absent[0] != "inputs/missing.txt" {
		t.Fatalf("partial seal absence claims: %+v", seal.Absent)
	}
	if _, err := o.Write("op.pseal.w1", h, "inputs/c.txt", []byte("c"), ModeCreate, 0); !errors.Is(err, ErrSealed) {
		t.Fatalf("write under partial seal: want ErrSealed, got %v", err)
	}
	if _, err := o.Write("op.pseal.w2", h, "scratch/c.txt", []byte("c"), ModeCreate, 0); err != nil {
		t.Fatalf("scratch outside seal must stay writable: %v", err)
	}
}

func TestNoEscapeAndNoLinkTraversal(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.esc.reserve", testBudgets())
	for _, bad := range []string{"/abs", "../up", "a/../../up", "a\x00b", ""} {
		if _, err := o.Read(h, bad, 0, 8, nil, ""); bad == "" {
			if !errors.Is(err, ErrWrongKind) {
				t.Fatalf("read root: want ErrWrongKind, got %v", err)
			}
		} else if !errors.Is(err, ErrEscape) && !errors.Is(err, ErrInvalid) {
			t.Fatalf("read %q: want escape/invalid, got %v", bad, err)
		}
		if _, err := o.Write("op.esc.w."+strings.ReplaceAll(strings.ReplaceAll(bad, "/", "_"), "\x00", "N"), h, bad, []byte("x"), ModeCreate, 0); err == nil {
			t.Fatalf("write %q must fail", bad)
		}
	}
	// A link inside the workspace is literal data; traversal refuses it.
	if _, err := o.MakeNode("op.esc.ln", h, "link", NodeSpec{Kind: KindLink, Target: "in"}); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Write("op.esc.wlink", h, "link/x", []byte("x"), ModeCreate, 0); !errors.Is(err, ErrEscape) {
		t.Fatalf("write through link: want ErrEscape, got %v", err)
	}
	if _, err := o.MakeNode("op.esc.abs", h, "abs", NodeSpec{Kind: KindLink, Target: "/etc"}); !errors.Is(err, ErrEscape) {
		t.Fatalf("absolute link target: want ErrEscape, got %v", err)
	}
}

func TestOperationIDConflictAndJoin(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.conf.reserve", testBudgets())
	if _, err := o.Write("op.conf.w", h, "a.txt", []byte("a"), ModeCreate, 0); err != nil {
		t.Fatal(err)
	}
	// Same ID with different arguments is a protocol error, without effect.
	if _, err := o.Write("op.conf.w", h, "b.txt", []byte("b"), ModeCreate, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed args: want ErrConflict, got %v", err)
	}
	if _, err := o.Stat(h, "b.txt"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("conflicting replay must have no effect: %v", err)
	}
	// Same ID with identical arguments joins.
	wr, err := o.Write("op.conf.w", h, "a.txt", []byte("a"), ModeCreate, 0)
	if err != nil || !wr.Joined {
		t.Fatalf("identical replay must join: %+v %v", wr, err)
	}
}

func TestRecoverDisposesAbandonedAllocation(t *testing.T) {
	j := journal.OpenMemory()
	o := testOwner(t, j)
	h := mustReserve(t, o, "op.rec.reserve", testBudgets())
	if _, err := o.Materialize("op.rec.mat", h, []MaterialEntry{{Path: "f", Data: []byte("x")}}, ModeCreate); err != nil {
		t.Fatal(err)
	}
	root, _ := o.RootForTesting(h)
	rep, err := o.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Outcomes) != 1 || rep.Outcomes[0].Action != "disposed" {
		t.Fatalf("abandoned allocation must be disposed: %+v", rep)
	}
	if _, serr := os.Lstat(root); !os.IsNotExist(serr) {
		t.Fatal("abandoned root must be removed")
	}
	// Recovery only touches registered roots: an unrelated directory with
	// a similar name is never claimed.
	stray := filepath.Join(t.TempDir(), "ws-stray")
	if err := os.Mkdir(stray, 0o700); err != nil {
		t.Fatal(err)
	}
	rep2, err := o.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if len(rep2.Outcomes) != 0 {
		t.Fatalf("second recovery must find nothing: %+v", rep2)
	}
	if _, serr := os.Lstat(stray); serr != nil {
		t.Fatal("recovery must not touch unregistered paths")
	}
}

func TestForeignLeaseBlocksDisposal(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.flease.reserve", testBudgets())
	future := time.Now().Add(time.Hour).UnixMilli()
	if _, err := o.GrantLease("op.flease.grant", h, "foreign-1", 424242, "other-start", future); err != nil {
		t.Fatal(err)
	}
	dr, err := o.Dispose("op.flease.dispose", h)
	if !errors.Is(err, ErrForeignLease) {
		t.Fatalf("dispose under foreign lease: want ErrForeignLease, got %v", err)
	}
	if dr == nil || len(dr.Remaining) == 0 {
		t.Fatalf("foreign lease must stay unresolved: %+v", dr)
	}
	rep, err := o.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Outcomes) != 1 || rep.Outcomes[0].Action != "foreign" {
		t.Fatalf("recover must report foreign lease: %+v", rep)
	}
}

// TestFailedWriteKeepsBytesCharged is the integrator regression test for the
// charge restore on write failure: a truncated file keeps its visible bytes
// charged instead of leaking them, and the failed operation ID rejoins the
// same outcome instead of redispatching.
func TestFailedWriteKeepsBytesCharged(t *testing.T) {
	o := testOwner(t, nil)
	h := mustReserve(t, o, "op.charge.reserve", testBudgets())
	if _, err := o.Materialize("op.charge.mat", h, []MaterialEntry{{Path: "f.txt", Data: []byte("12345678")}}, ModeCreate); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	o.failSync = errors.New("injected sync failure")
	_, werr := o.Materialize("op.charge.replace", h, []MaterialEntry{{Path: "f.txt", Data: []byte("replacement-bytes")}}, ModeReplace)
	o.failSync = nil
	if werr == nil {
		t.Fatal("injected sync failure accepted")
	}
	after, _, _, err := o.Charges(h)
	if err != nil {
		t.Fatalf("Charges: %v", err)
	}
	// Bytes were written before the injected sync failure; charges must
	// reflect the 17 visible bytes rather than leaking them.
	if after != 17 {
		t.Fatalf("charges = %d, want 17 (visible bytes stay charged)", after)
	}
	rd, err := o.Read(h, "f.txt", 0, 64, nil, "")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rd.Data) != 17 {
		t.Fatalf("visible bytes = %d, want 17", len(rd.Data))
	}
	if _, rerr := o.Materialize("op.charge.replace", h, []MaterialEntry{{Path: "f.txt", Data: []byte("replacement-bytes")}}, ModeReplace); rerr == nil {
		t.Fatal("failed operation ID must rejoin its failure, not redispatch")
	}
}
