package dispatch

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGrantValidation(t *testing.T) {
	gt := NewGrantTable()
	cases := []struct {
		name string
		run  func() error
		want error
	}{
		{"run grant bad run id", func() error { _, err := gt.IssueRunGrant("../x", 1000); return err }, ErrInvalid},
		{"run grant zero ttl", func() error { _, err := gt.IssueRunGrant("run-1", 0); return err }, ErrInvalid},
		{"run grant negative ttl", func() error { _, err := gt.IssueRunGrant("run-1", -5); return err }, ErrInvalid},
		{"run grant overlong ttl", func() error { _, err := gt.IssueRunGrant("run-1", MaxGrantTTLMs+1); return err }, ErrInvalid},
		{"case grant bad case id", func() error { _, err := gt.IssueCaseGrant("nope", "bad id!", 1000); return err }, ErrInvalid},
		{"case grant bad ttl", func() error { _, err := gt.IssueCaseGrant("nope", "case-1", 0); return err }, ErrInvalid},
		{"case grant unknown parent", func() error { _, err := gt.IssueCaseGrant("ng1-missing", "case-1", 1000); return err }, ErrNotFound},
		{"verify empty token", func() error { return gt.Verify("", ScopeRun, "run-1", "") }, ErrInvalid},
		{"verify bad scope", func() error { return gt.Verify("x", Scope("bogus"), "run-1", "") }, ErrInvalid},
		{"verify unknown token", func() error { return gt.Verify("ng1-missing", ScopeRun, "run-1", "") }, ErrNotFound},
		{"revoke unknown", func() error { return gt.Revoke("ng1-missing") }, ErrNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.run(); !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
}

func TestGrantLifecycle(t *testing.T) {
	now := int64(1_700_000_000_000)
	gt := NewGrantTableWithClock(func() int64 { return now })
	run, err := gt.IssueRunGrant("run-1", 60_000)
	if err != nil {
		t.Fatalf("IssueRunGrant: %v", err)
	}
	if !strings.HasPrefix(run.Token, grantTokenPrefix) || len(run.Token) > 256 {
		t.Fatalf("token %q unfit for owner_grant", run.Token)
	}
	if run.Scope != ScopeRun || run.RunID != "run-1" || run.CaseID != "" || run.Revoked {
		t.Fatalf("unexpected run grant: %+v", run)
	}
	if run.ExpiresWallMs != now+60_000 {
		t.Fatalf("expiry = %d, want %d", run.ExpiresWallMs, now+60_000)
	}

	verify := []struct {
		name  string
		token string
		scope Scope
		run   string
		case_ string
		want  error
	}{
		{"run verifies", run.Token, ScopeRun, "run-1", "", nil},
		{"wrong scope", run.Token, ScopeCase, "run-1", "case-1", ErrInvalid},
		{"wrong run", run.Token, ScopeRun, "run-2", "", ErrInvalid},
	}
	for _, c := range verify {
		t.Run(c.name, func(t *testing.T) {
			err := gt.Verify(c.token, c.scope, c.run, c.case_)
			if c.want == nil && err != nil {
				t.Fatalf("Verify err = %v, want nil", err)
			}
			if c.want != nil && !errors.Is(err, c.want) {
				t.Fatalf("Verify err = %v, want %v", err, c.want)
			}
		})
	}

	// Case grants derive from a live run grant and never outlive it.
	cg, err := gt.IssueCaseGrant(run.Token, "case-1", MaxGrantTTLMs)
	if err != nil {
		t.Fatalf("IssueCaseGrant: %v", err)
	}
	if cg.Scope != ScopeCase || cg.RunID != "run-1" || cg.CaseID != "case-1" {
		t.Fatalf("unexpected case grant: %+v", cg)
	}
	if cg.ExpiresWallMs != run.ExpiresWallMs {
		t.Fatalf("case expiry = %d, want parent cap %d", cg.ExpiresWallMs, run.ExpiresWallMs)
	}
	if err := gt.Verify(cg.Token, ScopeCase, "run-1", "case-1"); err != nil {
		t.Fatalf("case Verify: %v", err)
	}
	if err := gt.Verify(cg.Token, ScopeCase, "run-1", "case-2"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("case bound to another case: %v", err)
	}
	if _, err := gt.IssueCaseGrant(cg.Token, "case-2", 1000); !errors.Is(err, ErrInvalid) {
		t.Fatalf("case-from-case err = %v, want ErrInvalid", err)
	}

	// Revocation is per-token and idempotent.
	if err := gt.Revoke(run.Token); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if err := gt.Revoke(run.Token); err != nil {
		t.Fatalf("double Revoke: %v", err)
	}
	if err := gt.Verify(run.Token, ScopeRun, "run-1", ""); !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked verify err = %v, want ErrRevoked", err)
	}
	if _, err := gt.IssueCaseGrant(run.Token, "case-3", 1000); !errors.Is(err, ErrRevoked) {
		t.Fatalf("derive from revoked err = %v, want ErrRevoked", err)
	}

	// Expiry is enforced; expired entries are pruned on the next issue.
	now = run.ExpiresWallMs
	if err := gt.Verify(cg.Token, ScopeCase, "run-1", "case-1"); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired verify err = %v, want ErrExpired", err)
	}
	if _, err := gt.IssueRunGrant("run-2", 1000); err != nil {
		t.Fatalf("IssueRunGrant after expiry: %v", err)
	}
	if _, ok := gt.Lookup(cg.Token); ok {
		t.Fatal("expired grant survived pruning")
	}
	if _, ok := gt.Lookup("ng1-missing"); ok {
		t.Fatal("Lookup of unknown token reported a grant")
	}
}

func TestGrantTokensUnique(t *testing.T) {
	gt := NewGrantTable()
	seen := make(map[string]bool, 128)
	for i := 0; i < 128; i++ {
		g, err := gt.IssueRunGrant(fmt.Sprintf("run-%d", i), 1000)
		if err != nil {
			t.Fatalf("issue %d: %v", i, err)
		}
		if len(g.Token) == 0 || len(g.Token) > 256 {
			t.Fatalf("token length %d unfit for owner_grant", len(g.Token))
		}
		if seen[g.Token] {
			t.Fatal("duplicate grant token")
		}
		seen[g.Token] = true
	}
}

func TestGrantCapacity(t *testing.T) {
	gt := NewGrantTable()
	for i := 0; i < MaxGrants; i++ {
		if _, err := gt.IssueRunGrant(fmt.Sprintf("run-%d", i), 1000); err != nil {
			t.Fatalf("fill %d: %v", i, err)
		}
	}
	if _, err := gt.IssueRunGrant("run-overflow", 1000); !errors.Is(err, ErrCapacity) {
		t.Fatalf("overflow err = %v, want ErrCapacity", err)
	}
}
