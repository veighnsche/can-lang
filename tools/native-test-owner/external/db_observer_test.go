// Executed K22 observer examples. Moved verbatim from db_observer.go by the
// integrator: Go only runs Example* functions from _test.go files.

package external

import (
	"fmt"

	"github.com/veighnsche/can-lang/tools/native-test-owner/journal"
)

// ExampleObserveNamespace pins one raw connection, runs one read
// conversation, and drains the release. Only deterministic facts are
// printed: tokens and digests never cross into output.
func ExampleObserveNamespace() {
	j := journal.OpenMemory()
	r, _ := New(j, 4242, "start-token-1", Config{Namespaces: []string{"tenant-alpha"}})
	g, _ := r.GrantNamespace("op-ns", "tenant-alpha", 3600000)
	o, _ := ObserveNamespace(r, g.Token)
	rc, _ := o.Receipt()
	fmt.Println(rc.Namespace, rc.Pinned)
	conn, tok, _ := o.PinConnection("op-raw", "raw-1", 3600000)
	fmt.Println(conn.Name, conn.Namespace, conn.ConversationOpen)
	_ = o.BeginRead("op-raw", tok, "widgets")
	c2, _ := o.Connection("op-raw")
	fmt.Println(c2.ConversationOpen)
	_ = o.EndRead("op-raw", tok)
	_ = o.ReleaseConn("op-raw", tok, r.Owner())
	c3, _ := o.Connection("op-raw")
	fmt.Println(c3.Released)
	// Output:
	// tenant-alpha 0
	// raw-1 tenant-alpha false
	// true
	// true
}

// ExampleObserverError shows layered provenance: every observer failure
// names its layer in the message prefix.
func ExampleObserverError() {
	j := journal.OpenMemory()
	r, _ := New(j, 4242, "start-token-1", Config{Namespaces: []string{"tenant-alpha"}})
	g, _ := r.GrantNamespace("op-ns", "tenant-alpha", 3600000)
	o, _ := ObserveNamespace(r, g.Token)
	_, tok, _ := o.PinConnection("op-raw", "raw-1", 3600000)
	_ = o.BeginRead("op-raw", tok, "widgets")
	fmt.Println(o.BeginRead("op-raw", tok, "widgets"))
	fmt.Println(o.BeginRead("op-raw", "forged-token", "widgets"))
	// Output:
	// [driver:connection-busy] external: refused: one conversation is already open
	// [driver:forged-token] external: refused: connection token is not the pinned token
}
