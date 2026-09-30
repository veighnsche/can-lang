package driver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	compileresolve "github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// Structured product diagnostics (task P20): `canlc check --json` reports
// load/resolve/check results over identified inputs without assertions,
// emission, publication, application execution or dependency fetching.
//
// Wire contract (capabilities section 8): one bounded versioned JSON
// document on stdout. Completed reports carry status plus an accepted or
// rejected result; failed envelopes carry a failure kind and no result.
// Exits: 0 accepted, 1 completed rejection, 2 execution failure.
const (
	CheckSchemaVersion   = "1"
	CheckKind            = "can.check"
	MaxCheckDiagnostics  = 256
	MaxCheckReportBytes  = 1 << 20
	CheckStatusCompleted = "completed"
	CheckStatusFailed    = "failed"
)

// Check failure kinds. Invocation and resource failures are produced by CLI
// dispatch (P21), never here.
const (
	CheckFailureIO         = "io-identity"
	CheckFailureDependency = "dependency"
	CheckFailureInternal   = "internal"
	CheckFailureCancelled  = "cancelled"
	CheckFailureIncomplete = "incomplete-report"
)

// Check pipeline phases.
const (
	CheckPhaseProject = "project"
	CheckPhaseLex     = "lex"
	CheckPhaseParse   = "parse"
	CheckPhaseResolve = "resolve"
	CheckPhaseCheck   = "check"
)

// Check stage states.
const (
	CheckStageFinished   = "finished"
	CheckStageStopped    = "stopped"
	CheckStageNotEntered = "not-entered"
)

// CheckReport is the top-level JSON document.
type CheckReport struct {
	SchemaVersion string        `json:"schemaVersion"`
	Kind          string        `json:"kind"`
	Status        string        `json:"status"`
	Result        *CheckResult  `json:"result,omitempty"`
	Failure       *CheckFailure `json:"failure,omitempty"`
}

// CheckResult is present only on completed status.
type CheckResult struct {
	Accepted     bool              `json:"accepted"`
	Compiler     CheckCompiler     `json:"compiler"`
	Inputs       CheckInputs       `json:"inputs"`
	Diagnostics  []CheckDiagnostic `json:"diagnostics"`
	Completeness CheckCompleteness `json:"completeness"`
}

// CheckCompiler identifies the checking toolchain. Callers bind the actual
// supervisor executable lease separately; this never rests on
// self-identification alone.
type CheckCompiler struct {
	Go        string `json:"go"`
	Catalogue string `json:"catalogue"`
}

// CheckInputs binds declared roots, actual reads and enumeration facts.
type CheckInputs struct {
	DeclaredRoots []CheckDeclaredRoot `json:"declaredRoots"`
	Files         []CheckInputFile    `json:"files"`
	Enumerations  []CheckEnumeration  `json:"enumerations"`
	OverallDigest string              `json:"overallDigest"`
}

// CheckDeclaredRoot identifies one input root snapshot.
type CheckDeclaredRoot struct {
	Path     string  `json:"path"`
	Manifest *string `json:"manifest"`
	Lock     *string `json:"lock"`
}

// CheckInputFile is one actual read or failed lookup by logical path.
type CheckInputFile struct {
	Path   string  `json:"path"`
	Length *int    `json:"length,omitempty"`
	SHA256 *string `json:"sha256,omitempty"`
	Absent bool    `json:"absent"`
}

// CheckEnumeration records one directory enumeration affecting resolution.
type CheckEnumeration struct {
	Dir     string `json:"dir"`
	Entries int    `json:"entries"`
}

// CheckDiagnostic is one ordered diagnostic with producer phase.
type CheckDiagnostic struct {
	Phase          string         `json:"phase"`
	Severity       string         `json:"severity"`
	Code           string         `json:"code"`
	Message        string         `json:"message"`
	File           *string        `json:"file"`
	Location       *CheckLocation `json:"location"`
	SpanlessReason *string        `json:"spanlessReason,omitempty"`
	Related        []CheckRelated `json:"related"`
}

// CheckLocation uses a logical input path with zero-based UTF-16 start/end
// line/column, end exclusive.
type CheckLocation struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

// CheckRelated is one ordered distinct secondary location.
type CheckRelated struct {
	File     string        `json:"file"`
	Location CheckLocation `json:"location"`
	Message  string        `json:"message"`
}

// CheckCompleteness records per-stage state and unvisited units.
type CheckCompleteness struct {
	Stages    []CheckStage     `json:"stages"`
	Unvisited []CheckUnvisited `json:"unvisited"`
}

// CheckStage is one pipeline stage outcome.
type CheckStage struct {
	Stage         string  `json:"stage"`
	State         string  `json:"state"`
	StoppingCause *string `json:"stoppingCause,omitempty"`
}

// CheckUnvisited names a source and the stages never completed for it.
type CheckUnvisited struct {
	Path   string   `json:"path"`
	Stages []string `json:"stages"`
}

// CheckFailure is present only on failed status, with no result.
type CheckFailure struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// CheckProjectJSON loads, resolves and checks the named project directory
// and returns the report document plus the process exit code (0 accepted,
// 1 rejected, 2 failed). It never emits, publishes or executes the project.
func CheckProjectJSON(ctx context.Context, directory string) (doc []byte, exit int) {
	defer func() {
		if recovered := recover(); recovered != nil {
			doc = mustEncodeFailure(CheckFailureInternal, fmt.Sprintf("check panic: %v", recovered))
			exit = 2
		}
	}()
	if err := ctx.Err(); err != nil {
		return mustEncodeFailure(CheckFailureCancelled, err.Error()), 2
	}
	root, err := canonicalRoot(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return mustEncodeFailure(CheckFailureIO, "missing project root"), 2
		}
		if errors.Is(err, os.ErrPermission) {
			return mustEncodeFailure(CheckFailureIO, "project root not accessible"), 2
		}
		return mustEncodeFailure(CheckFailureIO, "project root identity unavailable"), 2
	}
	checker := &checker{ctx: ctx, root: root}
	return checker.run(directory)
}

func canonicalRoot(directory string) (string, error) {
	root, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", err
	}
	return filepath.Abs(root)
}

func mustEncodeFailure(kind, detail string) []byte {
	raw, err := json.Marshal(CheckReport{
		SchemaVersion: CheckSchemaVersion,
		Kind:          CheckKind,
		Status:        CheckStatusFailed,
		Failure:       &CheckFailure{Kind: kind, Detail: detail},
	})
	if err != nil {
		return []byte(`{"schemaVersion":"1","kind":"can.check","status":"failed","failure":{"kind":"internal","detail":"encode failure"}}`)
	}
	return raw
}

type checker struct {
	ctx  context.Context
	root string
}

type stagedDiagnostic struct {
	phase      string
	issue      source.Issue
	spanFile   string
	spanReason string
}

func (c *checker) run(directory string) ([]byte, int) {
	stages := []CheckStage{
		{Stage: CheckPhaseProject, State: CheckStageNotEntered},
		{Stage: CheckPhaseLex, State: CheckStageNotEntered},
		{Stage: CheckPhaseParse, State: CheckStageNotEntered},
		{Stage: CheckPhaseResolve, State: CheckStageNotEntered},
		{Stage: CheckPhaseCheck, State: CheckStageNotEntered},
	}
	setStage := func(name, state string, cause *string) {
		for i := range stages {
			if stages[i].Stage == name {
				stages[i].State = state
				stages[i].StoppingCause = cause
			}
		}
	}
	failCause := func(cause project.LoadCause) *string {
		name := loadCauseName(cause)
		return &name
	}

	graph, loadErr := project.LoadWithOverlayContext(c.ctx, directory, nil)
	cause, verdict := project.ClassifyLoadError(loadErr)
	if c.ctx.Err() != nil {
		cause, verdict = project.CauseCancelled, project.VerdictFail
	}
	if graph == nil || verdict == project.VerdictFail {
		if graph == nil && verdict != project.VerdictFail {
			cause, verdict = project.CauseUnknown, project.VerdictFail
		}
		setStage(CheckPhaseProject, CheckStageStopped, failCause(cause))
		return c.encodeFailure(cause, loadErr)
	}
	setStage(CheckPhaseProject, CheckStageFinished, nil)

	var staged []stagedDiagnostic
	for _, leaf := range project.ErrorLeaves(loadErr) {
		for _, issue := range loadLeafIssues(leaf) {
			phase := CheckPhaseProject
			if issue.fromSource {
				phase = CheckPhaseParse
				if strings.HasPrefix(issue.Code, "CAN-LEX-") {
					phase = CheckPhaseLex
				}
			}
			if issue.phase != "" {
				phase = issue.phase
			}
			staged = append(staged, stagedDiagnostic{phase: phase, issue: issue.Issue, spanFile: issue.spanFile, spanReason: issue.spanReason})
		}
	}
	setStage(CheckPhaseLex, CheckStageFinished, nil)
	setStage(CheckPhaseParse, CheckStageFinished, nil)

	if c.ctx.Err() != nil {
		stopped := CheckFailureCancelled
		setStage(CheckPhaseResolve, CheckStageStopped, &stopped)
		return mustEncodeFailure(CheckFailureCancelled, c.ctx.Err().Error()), 2
	}
	if graph.Root != nil {
		world := compileresolve.BuildRecoveringContext(c.ctx, graph)
		for _, err := range world.Errors {
			for _, issue := range errorIssues(err) {
				staged = append(staged, stagedDiagnostic{phase: CheckPhaseResolve, issue: issue})
			}
		}
		setStage(CheckPhaseResolve, CheckStageFinished, nil)
		if c.ctx.Err() != nil {
			stopped := CheckFailureCancelled
			setStage(CheckPhaseCheck, CheckStageStopped, &stopped)
			return mustEncodeFailure(CheckFailureCancelled, c.ctx.Err().Error()), 2
		}
		program, err := check.AnalyzeProgramContext(c.ctx, graph, world)
		for _, issue := range errorIssues(err) {
			staged = append(staged, stagedDiagnostic{phase: CheckPhaseCheck, issue: issue})
		}
		if program != nil {
			for _, warning := range program.Warnings {
				severity := warning.Severity
				if severity == "" {
					severity = source.SeverityWarning
				}
				staged = append(staged, stagedDiagnostic{phase: CheckPhaseCheck, issue: source.Issue{File: warning.File, Span: warning.Span, Code: warning.Code, Severity: severity, Message: warning.Message}})
			}
		}
		setStage(CheckPhaseCheck, CheckStageFinished, nil)
	}

	diagnostics := make([]CheckDiagnostic, 0, len(staged))
	for _, item := range staged {
		converted := issueDiagnostic(graph, item.issue)
		diagnostics = append(diagnostics, c.convertDiagnostic(item, converted))
	}
	if len(diagnostics) > MaxCheckDiagnostics {
		overflow := "report-overflow"
		setStage(CheckPhaseCheck, CheckStageStopped, &overflow)
		return mustEncodeFailure(CheckFailureIncomplete, fmt.Sprintf("diagnostic cap exceeded: %d > %d", len(diagnostics), MaxCheckDiagnostics)), 2
	}
	accepted := true
	for _, item := range staged {
		if item.issue.Severity == source.SeverityError {
			accepted = false
			break
		}
	}
	report := CheckReport{
		SchemaVersion: CheckSchemaVersion,
		Kind:          CheckKind,
		Status:        CheckStatusCompleted,
		Result: &CheckResult{
			Accepted:     accepted,
			Compiler:     CheckCompiler{Go: runtime.Version(), Catalogue: catalogue.SourceHash()},
			Inputs:       c.convertInputs(graph),
			Diagnostics:  diagnostics,
			Completeness: c.completeness(graph, stages),
		},
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return mustEncodeFailure(CheckFailureInternal, fmt.Sprintf("encode report: %v", err)), 2
	}
	if len(raw) > MaxCheckReportBytes {
		return mustEncodeFailure(CheckFailureIncomplete, fmt.Sprintf("report cap exceeded: %d > %d", len(raw), MaxCheckReportBytes)), 2
	}
	if accepted {
		return raw, 0
	}
	return raw, 1
}

func loadCauseName(cause project.LoadCause) string {
	switch cause {
	case project.CauseMissingInput:
		return "missing-input"
	case project.CauseDeniedRead:
		return "denied-read"
	case project.CauseDependencyUnavailable:
		return "dependency-unavailable"
	case project.CauseIOError:
		return "io-error"
	case project.CauseCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

func (c *checker) encodeFailure(cause project.LoadCause, loadErr error) ([]byte, int) {
	kind := CheckFailureInternal
	detail := "project load failed"
	switch cause {
	case project.CauseMissingInput:
		kind, detail = CheckFailureIO, missingDetail(loadErr)
	case project.CauseDeniedRead:
		kind, detail = CheckFailureIO, "project input not readable"
	case project.CauseDependencyUnavailable:
		kind, detail = CheckFailureDependency, "dependency input unavailable"
	case project.CauseIOError:
		kind, detail = CheckFailureIO, "project input error"
	case project.CauseCancelled:
		kind, detail = CheckFailureCancelled, "check cancelled"
	}
	return mustEncodeFailure(kind, c.sanitize(detail)), 2
}

func missingDetail(loadErr error) string {
	if loadErr == nil {
		return "project input missing"
	}
	for _, leaf := range project.ErrorLeaves(loadErr) {
		var pathErr *os.PathError
		if errors.As(leaf, &pathErr) {
			base := filepath.Base(pathErr.Path)
			if base == "can.project.json" {
				return "project manifest missing"
			}
			return "project input missing"
		}
	}
	return "project input missing"
}

// loadIssue is one load-stage issue with its source-file provenance. Spanless
// issues carry the implicated file and reason separately so conversion never
// fabricates a line-zero squiggle.
type loadIssue struct {
	source.Issue
	fromSource bool
	spanFile   string
	spanReason string
	phase      string
}

// loadLeafIssues converts one load error leaf to issues. Source-file
// failures split into lex/parse by producer code; graph, manifest and
// registry failures stay project-phase; infrastructure leaves yield no
// issues because the run already failed.
func loadLeafIssues(leaf error) []loadIssue {
	peeled := leaf
	if dep, ok := leaf.(*project.DependencyError); ok {
		peeled = dep.Err
	}
	if src, ok := peeled.(*project.SourceError); ok {
		return sourceIssues(src)
	}
	var graphErr *project.GraphError
	if errors.As(peeled, &graphErr) {
		return []loadIssue{{
			Issue:      source.Issue{Code: "CAN-PROJECT", Severity: source.SeverityError, Message: graphErr.Msg},
			spanFile:   graphErr.Path,
			spanReason: "project status has no source span",
		}}
	}
	var out []loadIssue
	for _, issue := range errorIssues(peeled) {
		out = append(out, loadIssue{Issue: issue})
	}
	return out
}

// sourceIssues converts a source-file failure to issues. Diagnostics from
// the lexer carry CAN-LEX- codes; parser failures carry the producer's
// grammar code. Undecodable bytes have no span.
func sourceIssues(src *project.SourceError) []loadIssue {
	if len(src.Diagnostics) == 0 {
		// Undecodable bytes fail before tokenization: lex phase, spanless.
		return []loadIssue{{
			Issue:      source.Issue{Code: "CAN-PROJECT", Severity: source.SeverityError, Message: src.Message},
			fromSource: true,
			spanFile:   src.Path,
			spanReason: "source bytes have no span",
			phase:      CheckPhaseLex,
		}}
	}
	var out []loadIssue
	for _, diagnostic := range src.Diagnostics {
		out = append(out, loadIssue{Issue: source.Issue{File: src.Path, Span: diagnostic.Span, Code: diagnostic.Code, Severity: source.SeverityError, Message: diagnostic.Message}, fromSource: true})
	}
	return out
}

func (c *checker) convertDiagnostic(item stagedDiagnostic, converted Diagnostic) CheckDiagnostic {
	diagnostic := CheckDiagnostic{
		Phase:    item.phase,
		Severity: converted.Severity,
		Code:     converted.Code,
		Message:  c.sanitize(converted.Message),
		Related:  []CheckRelated{},
	}
	if diagnostic.Severity == "" {
		diagnostic.Severity = item.issue.Severity
	}
	if diagnostic.Severity == "" {
		diagnostic.Severity = source.SeverityError
	}
	if converted.File != "" {
		logical := c.logical(converted.File)
		diagnostic.File = &logical
		diagnostic.Location = &CheckLocation{
			StartLine:   converted.Line,
			StartColumn: converted.Start,
			EndLine:     converted.EndLine,
			EndColumn:   converted.End,
		}
	} else {
		file := item.spanFile
		if file == "" {
			file = item.issue.File
		}
		if file != "" {
			logical := c.logical(file)
			diagnostic.File = &logical
		}
		reason := item.spanReason
		if reason == "" {
			reason = "spanless diagnostic"
			if item.issue.File != "" {
				reason = "source bytes unavailable"
			}
		}
		diagnostic.SpanlessReason = &reason
	}
	seen := map[CheckRelated]bool{}
	for _, related := range converted.Related {
		entry := CheckRelated{
			File:     c.logical(related.File),
			Location: CheckLocation{StartLine: related.Line, StartColumn: related.Start, EndLine: related.EndLine, EndColumn: related.End},
			Message:  c.sanitize(related.Message),
		}
		if seen[entry] {
			continue
		}
		seen[entry] = true
		diagnostic.Related = append(diagnostic.Related, entry)
	}
	return diagnostic
}

func (c *checker) convertInputs(graph *project.Graph) CheckInputs {
	inputs := CheckInputs{DeclaredRoots: []CheckDeclaredRoot{}, Files: []CheckInputFile{}, Enumerations: []CheckEnumeration{}}
	if graph == nil {
		return inputs
	}
	if graph.Root != nil {
		root := CheckDeclaredRoot{Path: "."}
		if graph.Root.ManifestSHA256 != "" {
			manifest := graph.Root.ManifestSHA256
			root.Manifest = &manifest
		}
		if graph.LockSHA256 != "" {
			lock := graph.LockSHA256
			root.Lock = &lock
		}
		inputs.DeclaredRoots = append(inputs.DeclaredRoots, root)
	}
	var digest bytes.Buffer
	facts := graph.ReadFacts()
	type logicalFact struct {
		logical string
		fact    project.ReadFact
	}
	ordered := make([]logicalFact, 0, len(facts))
	for _, fact := range facts {
		ordered = append(ordered, logicalFact{logical: c.logical(fact.Path), fact: fact})
	}
	sort.Slice(ordered, func(i, k int) bool { return ordered[i].logical < ordered[k].logical })
	for _, item := range ordered {
		entry := CheckInputFile{Path: item.logical, Absent: item.fact.Absent}
		if !item.fact.Absent {
			length, sha := item.fact.Length, item.fact.SHA256
			entry.Length, entry.SHA256 = &length, &sha
			fmt.Fprintf(&digest, "%s:%d:%s\n", item.logical, item.fact.Length, item.fact.SHA256)
		} else {
			fmt.Fprintf(&digest, "%s:absent\n", item.logical)
		}
		inputs.Files = append(inputs.Files, entry)
	}
	for _, dir := range graph.SortedEnumerated() {
		inputs.Enumerations = append(inputs.Enumerations, CheckEnumeration{Dir: c.logical(dir.Dir), Entries: dir.Entries})
	}
	sum := sha256.Sum256(digest.Bytes())
	inputs.OverallDigest = hex.EncodeToString(sum[:])
	return inputs
}

func (c *checker) completeness(graph *project.Graph, stages []CheckStage) CheckCompleteness {
	finished := map[string]bool{}
	for _, stage := range stages {
		finished[stage.Stage] = stage.State == CheckStageFinished
	}
	out := CheckCompleteness{Stages: stages, Unvisited: []CheckUnvisited{}}
	if graph == nil {
		return out
	}
	seen := map[string]bool{}
	var paths []string
	for _, proj := range graph.Projects {
		for _, src := range proj.Sources {
			logical := c.logical(src.Path)
			if seen[logical] {
				continue
			}
			seen[logical] = true
			paths = append(paths, logical)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		var missing []string
		for _, stage := range []string{CheckPhaseResolve, CheckPhaseCheck} {
			if !finished[stage] {
				missing = append(missing, stage)
			}
		}
		if len(missing) > 0 {
			out.Unvisited = append(out.Unvisited, CheckUnvisited{Path: path, Stages: missing})
		}
	}
	return out
}

// logical maps a canonical path to its project-relative logical spelling.
// Paths escaping the root are reported as external/<base> so reports never
// leak private absolute paths.
func (c *checker) logical(path string) string {
	rel, err := filepath.Rel(c.root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "external/" + filepath.Base(path)
	}
	return filepath.ToSlash(rel)
}

// sanitize redacts the canonical root prefix from messages so failures never
// leak private absolute paths.
func (c *checker) sanitize(message string) string {
	if c.root == "" {
		return message
	}
	return strings.ReplaceAll(message, c.root, ".")
}
