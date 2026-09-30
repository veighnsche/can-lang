// Package workspace implements owned bounded scratch workspaces for the
// native-test owner (task P09): reservation, identity-safe file operations,
// seals/leases and disposal.
//
// Every mutating operation registers journal intent before its first
// externally visible effect and rejoins identical operation IDs instead of
// re-dispatching. All path operations are directory-anchored through os.Root
// with explicit per-component symlink checks; confinement never relies on
// string-prefix checks followed by unrestricted I/O. Symlinks are inspected
// with Lstat/Readlink and never traversed.
//
// Stability strengths follow the capability contract: this package issues
// scoped_revision (capability-mediated mutations accounted for),
// changed_observed and unknown. It never issues qualified_snapshot because
// no host snapshot guarantee is qualified here; callers that require it are
// rejected instead of receiving a weaker claim.
//
// Unresolved resources stay owned and charged: partial allocation, symlink
// replacement, foreign/active leases, exceeded budgets and failed deletion
// are reported with remaining paths and errors, never silently cleaned.
package workspace
