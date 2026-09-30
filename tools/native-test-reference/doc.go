// Package reference materializes the P04 reference seed (R-seed): it verifies
// the P03 proposal against a pristine selection checkout, builds the seed
// compiler distribution through the ordinary reviewed bootstrap tools, and
// stages it under run/acceptance ownership with a sealed manifest.
//
// Selection identity is strict: the source directory must be a clean git
// checkout at the proposal head, and the catalogue, target manifest and Bun
// pins must match the proposal hashes exactly. Ignored files (the three
// .DS_Store strays counted in the proposal aggregates) are never part of
// identity; the seal reconciles the count explicitly and then records
// per-file hashes for every tracked runtime input, so mutation after
// selection invalidates forward from the seal.
//
// The package never invokes candidate canlc run: distribution.Build only
// compiles the seed compiler, and this package only drives it through the
// normal distbuild command-line interface.
package reference
