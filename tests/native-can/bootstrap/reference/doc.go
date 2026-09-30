// Package reference holds the P05 independent acceptance controls for the
// staged reference seed R.
//
// The controls qualify the staged seed only for the bounded judge features
// it must execute: exact native observations (parse, render, project and
// type inspection) over small fixed fixtures, exact compiler/runtime digest
// identity against the P04 seal, and literal assert-row evaluation as a
// transitional host check until a native judge evaluates assertions.
//
// Separation rules, enforced by the tests in this package:
//
//   - The seed resolves to one fixed staged directory (or an explicit
//     CAN_R_SEED override that must itself carry the seal). It is never
//     found via PATH, never built, and never the candidate compiler.
//   - Fixtures are self-contained under fixtures/ and observations are
//     fixed under observations/. The live tree, the suite sources and any
//     promotion-lane inputs are never read and never trusted here.
//   - R acceptance is recorded only by these controls passing; candidate
//     status and suite-source trust are separate concerns owned elsewhere.
package reference
