package project

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// Fixture is one statically referenced raw fixture file captured with its
// project. Relative is the normalized manifest-relative slash path used by
// the P15.1 fixture-tree digest; Bytes are the immutable input bytes workers
// observe. Capture resolves confinement before reading, exactly like
// check-time loading, so the two can never disagree on which file a path
// names.
type Fixture struct {
	Relative string
	Bytes    []byte
}

// MaxFixtureBytes bounds one captured raw fixture, matching check-time
// loading. Capture and loading enforce the same bound.
const MaxFixtureBytes = 8388608

// FixtureDigest implements the P15.1 fixture-tree format: it hashes
// `can-fixture-tree-v1` plus a zero byte, then every file ordered by the
// UTF-8 bytes of its normalized manifest-relative path. Each entry encodes
// path length, path bytes, content length and content bytes with P2's
// unsigned 64-bit big-endian framing. An empty set hashes the prefix alone.
// Duplicate paths fail; distinct paths with equal bytes are retained.
func FixtureDigest(files []Fixture) (string, error) {
	ordered := append([]Fixture(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Relative < ordered[j].Relative })
	hash := sha256.New()
	hash.Write([]byte("can-fixture-tree-v1\x00"))
	var length [8]byte
	previous := ""
	for _, file := range ordered {
		if err := NormalizePath(file.Relative); err != nil {
			return "", err
		}
		if file.Relative == previous {
			return "", fmt.Errorf("fixture digest requires unique paths")
		}
		previous = file.Relative
		binary.BigEndian.PutUint64(length[:], uint64(len([]byte(file.Relative))))
		hash.Write(length[:])
		hash.Write([]byte(file.Relative))
		binary.BigEndian.PutUint64(length[:], uint64(len(file.Bytes)))
		hash.Write(length[:])
		hash.Write(file.Bytes)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// captureFixtures reads every raw fixture file statically referenced by the
// project's declarations and templates into CheckedFixtures and records the
// P15.1 digest. Raw paths resolve relative to the defining source file and
// stay confined to the project root. Missing files, escapes and symlink
// violations fail the load; check-time loading serves these captured bytes
// instead of rereading the working tree.
func (p *Project) captureFixtures() error {
	confined, err := filepath.EvalSymlinks(p.Root)
	if err != nil {
		return err
	}
	captured := map[string][]byte{}
	for _, src := range p.Sources {
		paths, err := rawReferences(src.Syntax)
		if err != nil {
			return err
		}
		directory := filepath.Dir(src.Path)
		for _, path := range paths {
			if path == "" || filepath.IsAbs(path) {
				return fmt.Errorf("raw fixture %q must be a source-relative path", path)
			}
			real, err := filepath.EvalSymlinks(filepath.Join(directory, filepath.FromSlash(path)))
			if err != nil {
				return fmt.Errorf("raw fixture %q is not readable", path)
			}
			if !Contains(confined, real) {
				return fmt.Errorf("raw fixture %q escapes its project", path)
			}
			rel, err := filepath.Rel(confined, real)
			if err != nil {
				return fmt.Errorf("raw fixture %q escapes its project", path)
			}
			relative := filepath.ToSlash(rel)
			if err := NormalizePath(relative); err != nil {
				return fmt.Errorf("raw fixture %q escapes its project", path)
			}
			if _, ok := captured[relative]; ok {
				continue
			}
			data, err := os.ReadFile(real)
			if err != nil {
				return fmt.Errorf("raw fixture %q is not readable", path)
			}
			if len(data) > MaxFixtureBytes {
				return fmt.Errorf("raw fixture %q exceeds its size bound", path)
			}
			captured[relative] = append([]byte(nil), data...)
		}
	}
	p.CheckedFixtures = p.CheckedFixtures[:0]
	for _, relative := range sortedKeys(captured) {
		p.CheckedFixtures = append(p.CheckedFixtures, Fixture{Relative: relative, Bytes: captured[relative]})
	}
	p.FixturesSHA256, err = FixtureDigest(p.CheckedFixtures)
	return err
}

// FixtureBytes serves one captured fixture to check-time loading. The path
// resolves exactly as at capture; a missing entry fails closed instead of
// rereading the working tree, so workers always observe captured bytes.
func (p *Project) FixtureBytes(sourceDir, path string) ([]byte, error) {
	if path == "" || filepath.IsAbs(path) {
		return nil, fmt.Errorf("raw fixture %q must be a source-relative path", path)
	}
	real, err := filepath.EvalSymlinks(filepath.Join(sourceDir, filepath.FromSlash(path)))
	if err != nil {
		return nil, fmt.Errorf("raw fixture %q is not readable", path)
	}
	confined, err := filepath.EvalSymlinks(p.Root)
	if err != nil {
		return nil, fmt.Errorf("raw fixture %q escapes its project", path)
	}
	if !Contains(confined, real) {
		return nil, fmt.Errorf("raw fixture %q escapes its project", path)
	}
	rel, err := filepath.Rel(confined, real)
	if err != nil {
		return nil, fmt.Errorf("raw fixture %q escapes its project", path)
	}
	relative := filepath.ToSlash(rel)
	for _, fixture := range p.CheckedFixtures {
		if fixture.Relative == relative {
			return fixture.Bytes, nil
		}
	}
	return nil, fmt.Errorf("raw fixture %q was not captured with its project inputs", path)
}

// rawReferences collects every `using raw` path in a parsed source file:
// attached assertion rows, lexical when rows in any body, and fixture
// template cases. Unknown nodes fail the walk so a future syntax addition
// cannot silently escape capture.
func rawReferences(file *syntax.File) ([]string, error) {
	w := &rawWalker{}
	for _, decl := range file.Declarations {
		if err := w.declaration(decl); err != nil {
			return nil, err
		}
	}
	return w.paths, nil
}

type rawWalker struct {
	paths []string
}

func (w *rawWalker) mode(mode *syntax.AssertionMode) {
	if mode != nil && mode.Raw.Value != "" {
		w.paths = append(w.paths, mode.Raw.Value)
	}
}

func (w *rawWalker) assertion(row *syntax.Assertion) error {
	if row.Receiver != nil {
		if err := w.expr(row.Receiver); err != nil {
			return err
		}
	}
	for _, argument := range row.Arguments {
		if err := w.argument(argument); err != nil {
			return err
		}
	}
	if row.Expected != nil {
		if err := w.body(row.Expected); err != nil {
			return err
		}
	}
	w.mode(row.Mode)
	return nil
}

func (w *rawWalker) declaration(decl syntax.Declaration) error {
	switch n := decl.(type) {
	case *syntax.RecordDecl, *syntax.VariantDecl, *syntax.ErrorDecl, *syntax.ScenarioDecl, *syntax.ActionDecl:
		return nil
	case *syntax.FunctionDecl:
		for i := range n.Assertions {
			if err := w.assertion(&n.Assertions[i]); err != nil {
				return err
			}
		}
		return w.block(n.Body)
	case *syntax.ValueDecl:
		return w.expr(n.Binding.Value)
	case *syntax.ConnectionDecl:
		for _, setting := range n.Settings {
			if err := w.connectionSetting(setting); err != nil {
				return err
			}
		}
		return nil
	case *syntax.FetchDecl:
		for i := range n.Assertions {
			if err := w.assertion(&n.Assertions[i]); err != nil {
				return err
			}
		}
		for _, entry := range n.Query {
			if err := w.expr(entry.Value); err != nil {
				return err
			}
		}
		for _, entry := range n.Headers {
			if err := w.expr(entry.Value); err != nil {
				return err
			}
		}
		if n.Path != nil {
			if err := w.expr(n.Path); err != nil {
				return err
			}
		}
		if n.Body != nil {
			return w.expr(n.Body)
		}
		return nil
	case *syntax.LLMDecl:
		for i := range n.Assertions {
			if err := w.assertion(&n.Assertions[i]); err != nil {
				return err
			}
		}
		if n.Asks != nil {
			return w.expr(n.Asks)
		}
		return nil
	case *syntax.FixtureDecl:
		for _, kase := range n.Cases {
			for _, argument := range kase.Arguments {
				if err := w.argument(argument); err != nil {
					return err
				}
			}
			if kase.Expected != nil {
				if err := w.body(kase.Expected); err != nil {
					return err
				}
			}
			w.mode(kase.Mode)
		}
		return nil
	case *syntax.WrapDecl:
		for i := range n.Assertions {
			if err := w.assertion(&n.Assertions[i]); err != nil {
				return err
			}
		}
		for _, arm := range n.Native {
			if arm.Body != nil {
				if err := w.body(arm.Body); err != nil {
					return err
				}
			}
		}
		for _, arm := range n.Emitted {
			if arm.Body != nil {
				if err := w.body(arm.Body); err != nil {
					return err
				}
			}
		}
		return nil
	case *syntax.JudgeDecl:
		for i := range n.Assertions {
			if err := w.assertion(&n.Assertions[i]); err != nil {
				return err
			}
		}
		for _, registration := range n.Registrations {
			if registration.Call != nil {
				if err := w.expr(registration.Call); err != nil {
					return err
				}
			}
		}
		if n.Continuation != nil {
			return w.body(n.Continuation)
		}
		return nil
	case *syntax.ChoiceArmDecl:
		if n.Description != nil {
			if err := w.expr(n.Description); err != nil {
				return err
			}
		}
		return w.block(n.Body)
	case *syntax.QuestionDecl:
		if n.Asks != nil {
			if err := w.expr(n.Asks); err != nil {
				return err
			}
		}
		if n.Minimum != nil {
			if err := w.expr(n.Minimum); err != nil {
				return err
			}
		}
		if n.Fallback != nil {
			if err := w.body(n.Fallback); err != nil {
				return err
			}
		}
		for _, option := range n.Options {
			if option.Description != nil {
				if err := w.expr(option.Description); err != nil {
					return err
				}
			}
			if option.Spread != nil {
				if err := w.expr(option.Spread); err != nil {
					return err
				}
			}
			if option.Body != nil {
				if err := w.body(option.Body); err != nil {
					return err
				}
			}
		}
		if n.Shared != nil {
			return w.body(n.Shared)
		}
		return nil
	default:
		return fmt.Errorf("raw reference walk reached an unhandled declaration %T", decl)
	}
}

func (w *rawWalker) connectionSetting(setting syntax.ConnectionSetting) error {
	if setting.Value != nil {
		if err := w.expr(setting.Value); err != nil {
			return err
		}
	}
	for _, entry := range setting.Entries {
		if err := w.connectionSetting(entry); err != nil {
			return err
		}
	}
	return nil
}

func (w *rawWalker) block(block syntax.Block) error {
	for _, step := range block.Steps {
		if err := w.step(step); err != nil {
			return err
		}
	}
	if block.Terminal != nil {
		return w.body(block.Terminal)
	}
	return nil
}

func (w *rawWalker) step(step syntax.Step) error {
	switch n := step.(type) {
	case *syntax.BindingStep:
		return w.expr(n.Value)
	case *syntax.CallStep:
		if n.Call != nil {
			return w.expr(n.Call)
		}
		return nil
	case *syntax.CoordinationStep:
		return w.coordination(n.Coordination)
	default:
		return fmt.Errorf("raw reference walk reached an unhandled step %T", step)
	}
}

func (w *rawWalker) body(body syntax.Body) error {
	switch n := body.(type) {
	case *syntax.ValueBody:
		if n.Value == nil {
			return nil
		}
		return w.expr(n.Value)
	case *syntax.SuccessBody:
		// Bare `ok` carries no value expression.
		if n.Value == nil {
			return nil
		}
		return w.expr(n.Value)
	case *syntax.FailureBody:
		if n.Error != nil {
			return w.expr(n.Error)
		}
		return nil
	case *syntax.RelayBody:
		if n.Call != nil {
			return w.expr(n.Call)
		}
		return nil
	case *syntax.InheritBody:
		return nil
	case *syntax.DoBody:
		return w.block(n.Block)
	case *syntax.MatchBody:
		return w.match(n.Match)
	default:
		return fmt.Errorf("raw reference walk reached an unhandled body %T", body)
	}
}

func (w *rawWalker) coordination(coordination syntax.Coordination) error {
	for _, participant := range coordination.Participants {
		if participant.Call != nil {
			if err := w.expr(participant.Call); err != nil {
				return err
			}
		}
		if participant.Spread != nil {
			if err := w.expr(participant.Spread); err != nil {
				return err
			}
		}
		for _, arm := range participant.Arms {
			if arm.Body != nil {
				if err := w.body(arm.Body); err != nil {
					return err
				}
			}
		}
	}
	for _, arm := range coordination.Arms {
		if arm.Body != nil {
			if err := w.body(arm.Body); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *rawWalker) match(match syntax.Match) error {
	for _, value := range match.Values {
		if err := w.expr(value); err != nil {
			return err
		}
	}
	if match.Call != nil {
		if err := w.expr(match.Call); err != nil {
			return err
		}
	}
	for _, entry := range match.Chain {
		if entry.Call != nil {
			if err := w.expr(entry.Call); err != nil {
				return err
			}
		}
	}
	for i := range match.When {
		if err := w.assertion(&match.When[i]); err != nil {
			return err
		}
	}
	for _, arm := range match.Arms {
		if arm.Body != nil {
			if err := w.body(arm.Body); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *rawWalker) argument(argument syntax.Argument) error {
	if argument.Group != nil {
		for _, value := range argument.Group.Values {
			if err := w.expr(value); err != nil {
				return err
			}
		}
	}
	if argument.Value != nil {
		return w.expr(argument.Value)
	}
	return nil
}

func (w *rawWalker) call(call *syntax.CallExpr) error {
	if call.Invocation.Callee != nil {
		if err := w.expr(call.Invocation.Callee); err != nil {
			return err
		}
	}
	for _, argument := range call.Invocation.Arguments {
		if err := w.argument(argument); err != nil {
			return err
		}
	}
	for _, method := range call.Methods {
		for _, argument := range method.Arguments {
			if err := w.argument(argument); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *rawWalker) expr(node syntax.Expr) error {
	switch n := node.(type) {
	case *syntax.LiteralExpr, *syntax.ScopeExpr, *syntax.NameExpr, *syntax.ProbabilityExpr:
		return nil
	case *syntax.GroupExpr:
		return w.expr(n.Value)
	case *syntax.UnaryExpr:
		return w.expr(n.Operand)
	case *syntax.BinaryExpr:
		if err := w.expr(n.Left); err != nil {
			return err
		}
		return w.expr(n.Right)
	case *syntax.ComparisonExpr:
		for _, operand := range n.Operands {
			if err := w.expr(operand); err != nil {
				return err
			}
		}
		return nil
	case *syntax.ArrayExpr:
		for _, element := range n.Elements {
			if err := w.argument(element); err != nil {
				return err
			}
		}
		return nil
	case *syntax.FieldExpr:
		return w.expr(n.Receiver)
	case *syntax.IndexExpr:
		if err := w.expr(n.Receiver); err != nil {
			return err
		}
		return w.expr(n.Index)
	case *syntax.SliceExpr:
		if err := w.expr(n.Receiver); err != nil {
			return err
		}
		if n.Start != nil {
			if err := w.expr(n.Start); err != nil {
				return err
			}
		}
		if n.End != nil {
			if err := w.expr(n.End); err != nil {
				return err
			}
		}
		return nil
	case *syntax.ConstructorExpr:
		for _, argument := range n.Arguments {
			if err := w.argument(argument); err != nil {
				return err
			}
		}
		return nil
	case *syntax.CallExpr:
		return w.call(n)
	case *syntax.ReferenceExpr:
		return w.expr(n.Callee)
	case *syntax.UpdateExpr:
		if err := w.expr(n.Receiver); err != nil {
			return err
		}
		for _, field := range n.Fields {
			if err := w.expr(field.Value); err != nil {
				return err
			}
		}
		return nil
	case *syntax.MatchExpr:
		return w.match(n.Match)
	case *syntax.CoordinationExpr:
		return w.coordination(n.Coordination)
	default:
		return fmt.Errorf("raw reference walk reached an unhandled expression %T", node)
	}
}
