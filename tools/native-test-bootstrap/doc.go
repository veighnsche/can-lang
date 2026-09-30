// Package bootstrap implements the bounded transitional bootstrap
// parent/witness T for native-test first acceptance (task P25).
//
// T lives outside the tested owner/worker killable subtree and parents the
// fixture processes under test. Before any effect it pre-registers finite
// child, descriptor and scratch authority; it retains stop/reap ownership
// over its children and an independent release witness that checks claimed
// receipts against actual release facts (observed exit plus reap for
// children, observed absence for scratch).
//
// T is transitional bootstrap machinery, not N: it covers an explicit
// bounded process/resource graph only and claims no containment of
// arbitrary detached descendants. If T itself is lost, nothing it
// recorded can issue a green receipt or grant new admission: Admit and
// CheckRelease fail closed with ErrWitnessLost, and Inspect exposes the
// retained record read-only for recovery under a new witness.
package bootstrap
