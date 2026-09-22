package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"reflect"
	"sort"
)

// NativeDeclaration retains checked contracts independently from provider
// lowering. Questions are registration targets, never ordinary calls.
type NativeDeclaration struct {
	Symbol         *resolve.Symbol
	Signature      *types.Type
	Connection     string
	Descriptor     CallableDeclaration
	State          []syntax.Field
	Regions        []*ir.Region
	Registrations  []ir.JudgeRegistration
	LLM            *ir.LLM
	Fetch          *ir.Fetch
	Question       *ir.Question
	ArmDescription *ir.Expression
	Judge          *ir.Judge
	// Native holds the raw intrinsic obligations N and Emitted the declared
	// authored obligations E, both by exact error identity. Membership is
	// tracked per set: one identity may belong to both.
	Native  []string
	Emitted []string
}

func (c *programChecker) gatherNative(file *resolve.File, declaration syntax.Declaration) (*NativeDeclaration, error) {
	header := syntax.NativeSignature(declaration)
	var state []syntax.Field
	switch d := declaration.(type) {
	case *syntax.JudgeDecl:
		state = d.State
	case *syntax.LLMDecl:
		state = d.State
	case *syntax.ChoiceArmDecl:
		header = &syntax.NativeHeader{Name: d.Name, Result: d.Result, Errors: d.Errors}
	}
	if header == nil {
		return nil, fmt.Errorf("missing native signature")
	}
	if _, err := c.gather(file, &syntax.ArrayType{Element: &syntax.NamedType{Name: syntax.QualifiedName{Name: "str"}}}); err != nil {
		return nil, err
	}
	symbol := file.Package.Scope.Symbols[header.Name.Text]
	native := &NativeDeclaration{Symbol: symbol, State: state, Descriptor: CallableDeclaration{Kind: symbol.Kind, State: len(state), Grouped: symbol.Kind == resolve.Judge || symbol.Kind == resolve.LLM}}
	if symbol.Kind != resolve.ChoiceArm {
		connection, err := file.Lookup(nil, header.Connection, resolve.ConnectionUse)
		if err != nil {
			return nil, err
		}
		native.Connection = connection.ID
	}
	result := header.Result
	if q, ok := declaration.(*syntax.QuestionDecl); ok && q.RecordName != nil {
		result = &syntax.NamedType{Name: syntax.QualifiedName{Name: q.RecordName.Text}}
	}
	signature := &syntax.CallableType{Result: result, Errors: header.Errors}
	for i, input := range header.Inputs {
		field := input.Field
		if input.Variadic {
			if i != len(header.Inputs)-1 {
				return nil, fmt.Errorf("variadic input must be last")
			}
			field.Type = &syntax.ArrayType{Element: field.Type}
			c.variadic[symbol.ID] = true
		}
		signature.Inputs = append(signature.Inputs, field.Type)
		typ, err := c.gather(file, field.Type)
		if err != nil {
			return nil, err
		}
		c.bindings[symbol.ID+"/input/"+field.Name.Text] = typ
		native.Descriptor.Names = append(native.Descriptor.Names, field.Name.Text)
		native.Descriptor.Near = append(native.Descriptor.Near, input.Near)
	}
	for _, field := range state {
		signature.Inputs = append(signature.Inputs, field.Type)
		typ, err := c.gather(file, field.Type)
		if err != nil {
			return nil, err
		}
		c.bindings[symbol.ID+"/input/"+field.Name.Text] = typ
		native.Descriptor.Names = append(native.Descriptor.Names, field.Name.Text)
		native.Descriptor.Near = append(native.Descriptor.Near, false)
	}
	if q, ok := declaration.(*syntax.QuestionDecl); ok {
		for _, binder := range q.Binders {
			c.bindings[symbol.ID+"/metadata/"+binder.Name.Text] = c.annotations[file]["float"]
		}
	}
	var err error
	native.Signature, err = c.gather(file, signature)
	if err != nil {
		return nil, err
	}
	native.Descriptor.Contract = native.Signature
	c.bindings[symbol.ID] = native.Signature
	if symbol.Kind == resolve.ChoiceArm {
		c.bindings[symbol.ID], err = c.gather(file, &syntax.ChoiceArmType{Result: header.Result, Errors: header.Errors})
		if err != nil {
			return nil, err
		}
	}
	if err = c.gatherBody(file, reflect.ValueOf(declaration)); err != nil {
		return nil, err
	}
	return native, nil
}

func (c *programChecker) checkNativeContracts(program *Program) error {
	for _, native := range program.Natives {
		for _, field := range native.State {
			if _, err := types.Schema(c.bindings[native.Symbol.ID+"/input/"+field.Name.Text]); err != nil {
				return fmt.Errorf("native state %s: %w", field.Name.Text, err)
			}
		}
		policy, connected := program.Connections[native.Connection]
		if native.Symbol.Kind != resolve.ChoiceArm && !connected {
			return fmt.Errorf("native declaration has no checked connection")
		}
		required := []uint64{}
		switch native.Symbol.Kind {
		case resolve.Question:
			if err := policy.CheckAI("typesafe_systemone_v1"); err != nil {
				return err
			}
			required = []uint64{1120, 1121}
		case resolve.Judge:
			if err := policy.CheckAI("typesafe_systemone_v1"); err != nil {
				return err
			}
			required = []uint64{1100, 1102, 1103, 1104, 1105, 1110, 1120, 1121}
		case resolve.LLM:
			if err := policy.CheckAI("openai_responses_v1"); err != nil {
				return err
			}
			required = []uint64{1100, 1102, 1103, 1104, 1105, 1110, 1130, 1131, 1132}
		case resolve.Fetch:
			required = []uint64{1100, 1102, 1103, 1104}
			result := native.Signature.Result()
			envelope := result.Declaration() == "can.std.http@1::response"
			if envelope {
				result = result.Arguments()[0]
			} else {
				required = append(required, 1105)
			}
			if !(scalar(result, "str") || result.Declaration() == "can.std.bytes@1::buffer" || result.Kind() == types.Record && result.Declaration() != "can.std.http@1::response") {
				return fmt.Errorf("fetch result requires record, str, bytes or one response envelope")
			}
			declaration := native.Symbol.Declaration.(*syntax.FetchDecl)
			if declaration.Method.Text == "head" && !scalar(result, "str") && result.Declaration() != "can.std.bytes@1::buffer" {
				return fmt.Errorf("HEAD requires text or bytes result")
			}
			if result.Declaration() != "can.std.bytes@1::buffer" || declaration.BodyEncoding != nil && declaration.BodyEncoding.Text != "bytes" {
				required = append(required, 1110)
			}
		}
		if native.Symbol.Kind != resolve.Question && native.Symbol.Kind != resolve.ChoiceArm && policy.BearerEnvironment != "" {
			required = append(required, 1101)
		}
		ids := map[uint64]bool{}
		emitted := []string{}
		for _, typ := range native.Signature.Errors() {
			allocation, ok := program.Registry.declarations[typ.Declaration()]
			if !ok {
				return fmt.Errorf("unregistered native error")
			}
			ids[allocation.ID] = true
			emitted = append(emitted, typ.Declaration())
		}
		for _, id := range required {
			if !ids[id] {
				return fmt.Errorf("native declaration %s emits omits required intrinsic error %d", native.Symbol.Name, id)
			}
		}
		// N holds the raw infrastructure obligations and E the declared
		// authored obligations, both by exact error identity. Fetch keeps
		// every required intrinsic; judges exclude AI validation, which is
		// required but preserved and hence E-only; questions, arms and LLM
		// declarations hold no raw obligations of their own.
		raw := map[uint64]bool{}
		switch native.Symbol.Kind {
		case resolve.Fetch:
			for _, id := range required {
				raw[id] = true
			}
		case resolve.Judge:
			for _, id := range required {
				if id != 1120 && id != 1121 {
					raw[id] = true
				}
			}
		}
		byID := map[uint64]string{}
		for _, declaration := range program.Registry.Declarations() {
			byID[declaration.ID] = declaration.Identity
		}
		native.Native = []string{}
		for id := range raw {
			identity, ok := byID[id]
			if !ok {
				return fmt.Errorf("intrinsic error %d has no registered identity", id)
			}
			native.Native = append(native.Native, identity)
		}
		sort.Strings(native.Native)
		sort.Strings(emitted)
		native.Emitted = emitted
	}
	return nil
}
