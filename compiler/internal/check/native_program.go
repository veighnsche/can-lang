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

// Intrinsic catalogue error identities required by native declarations.
const (
	intrinsicInvalidRequest     = "can.std.http@1::invalid_request"
	intrinsicCredentialsMissing = "can.std.http@1::credentials_missing"
	intrinsicTransportFailed    = "can.std.http@1::transport_failed"
	intrinsicTimeout            = "can.std.http@1::timeout"
	intrinsicBodyLimit          = "can.std.http@1::body_limit"
	intrinsicStatusError        = "can.std.http@1::status_error"
	intrinsicRequestFailed      = "can.std.http@1::request_failed"
	intrinsicInvalidData        = "can.std.codec@1::invalid_data"
	intrinsicInvalidQuestion    = "can.std.ai@1::invalid_question"
	intrinsicInvalidAnswer      = "can.std.ai@1::invalid_answer"
	intrinsicRefused            = "can.std.llm@1::refused"
	intrinsicTruncated          = "can.std.llm@1::truncated"
	intrinsicInvalidResponse    = "can.std.llm@1::invalid_response"
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
	// Wrapper holds the resolved policy plan for wrapper declarations.
	Wrapper *WrapperPlan
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
		if native.Symbol.Kind == resolve.Wrapper {
			continue
		}
		for _, field := range native.State {
			if _, err := types.Schema(c.bindings[native.Symbol.ID+"/input/"+field.Name.Text]); err != nil {
				return fmt.Errorf("native state %s: %w", field.Name.Text, err)
			}
		}
		policy, connected := program.Connections[native.Connection]
		if native.Symbol.Kind != resolve.ChoiceArm && !connected {
			return fmt.Errorf("native declaration has no checked connection")
		}
		required := []string{}
		intrinsic := []string{}
		switch native.Symbol.Kind {
		case resolve.Question:
			if err := policy.CheckAI("typesafe_systemone_v1"); err != nil {
				return err
			}
			required = []string{intrinsicInvalidQuestion, intrinsicInvalidAnswer}
		case resolve.Judge:
			if err := policy.CheckAI("typesafe_systemone_v1"); err != nil {
				return err
			}
			required = []string{intrinsicRequestFailed, intrinsicInvalidQuestion, intrinsicInvalidAnswer}
			intrinsic = []string{intrinsicInvalidRequest, intrinsicTransportFailed, intrinsicTimeout, intrinsicBodyLimit, intrinsicStatusError, intrinsicInvalidData}
		case resolve.LLM:
			if err := policy.CheckAI("openai_responses_v1"); err != nil {
				return err
			}
			required = []string{intrinsicInvalidRequest, intrinsicTransportFailed, intrinsicTimeout, intrinsicBodyLimit, intrinsicStatusError, intrinsicInvalidData, intrinsicRefused, intrinsicTruncated, intrinsicInvalidResponse}
		case resolve.Fetch:
			required = []string{intrinsicRequestFailed}
			intrinsic = []string{intrinsicInvalidRequest, intrinsicTransportFailed, intrinsicTimeout, intrinsicBodyLimit}
			result := native.Signature.Result()
			envelope := result.Declaration() == "can.std.http@1::response"
			if envelope {
				result = result.Arguments()[0]
			} else {
				intrinsic = append(intrinsic, intrinsicStatusError)
			}
			if !(scalar(result, "str") || result.Declaration() == "can.std.bytes@1::buffer" || result.Kind() == types.Record && result.Declaration() != "can.std.http@1::response") {
				return fmt.Errorf("fetch result requires record, str, bytes or one response envelope")
			}
			declaration := native.Symbol.Declaration.(*syntax.FetchDecl)
			if declaration.Method.Text == "head" && !scalar(result, "str") && result.Declaration() != "can.std.bytes@1::buffer" {
				return fmt.Errorf("HEAD requires text or bytes result")
			}
			if result.Declaration() != "can.std.bytes@1::buffer" || declaration.BodyEncoding != nil && declaration.BodyEncoding.Text != "bytes" {
				intrinsic = append(intrinsic, intrinsicInvalidData)
			}
		}
		if native.Symbol.Kind == resolve.Fetch || native.Symbol.Kind == resolve.Judge {
			if policy.BearerEnvironment != "" {
				intrinsic = append(intrinsic, intrinsicCredentialsMissing)
			}
		} else if native.Symbol.Kind != resolve.Question && native.Symbol.Kind != resolve.ChoiceArm && policy.BearerEnvironment != "" {
			required = append(required, intrinsicCredentialsMissing)
		}
		seen := map[string]bool{}
		emitted := []string{}
		for _, typ := range native.Signature.Errors() {
			if _, ok := program.Registry.declarations[typ.Declaration()]; !ok {
				return fmt.Errorf("unregistered native error")
			}
			seen[typ.Declaration()] = true
			emitted = append(emitted, typ.Declaration())
		}
		for _, identity := range required {
			if !seen[identity] {
				return fmt.Errorf("native declaration %s emits omits required intrinsic error %s", native.Symbol.Name, identity)
			}
		}
		// N holds the raw infrastructure obligations subject to boundary
		// normalization; E holds the declared authored obligations. AI
		// validation is required but preserved, hence E-only; only fetch
		// and judge maintain a normalization boundary at all. The intrinsic
		// normalized contribution alone is not an emitted-origin entry.
		raw := map[string]bool{}
		for _, identity := range intrinsic {
			raw[identity] = true
		}
		registered := map[string]bool{}
		for _, declaration := range program.Registry.Declarations() {
			registered[declaration.Identity] = true
		}
		native.Native = []string{}
		for identity := range raw {
			if !registered[identity] {
				return fmt.Errorf("intrinsic error %s has no registered identity", identity)
			}
			native.Native = append(native.Native, identity)
		}
		sort.Strings(native.Native)
		if len(raw) > 0 {
			kept := emitted[:0]
			for _, identity := range emitted {
				if identity != intrinsicRequestFailed {
					kept = append(kept, identity)
				}
			}
			emitted = kept
		}
		sort.Strings(emitted)
		native.Emitted = emitted
	}
	// Wrappers share the original boundary sets of their root operation;
	// policy lookup is against the original boundary, not the parent bound.
	bySymbol := map[*resolve.Symbol]*NativeDeclaration{}
	for _, native := range program.Natives {
		bySymbol[native.Symbol] = native
	}
	for _, native := range program.Natives {
		if native.Symbol.Kind != resolve.Wrapper {
			continue
		}
		file := c.world.Files[native.Symbol.Source]
		_, root, _, err := c.world.WrapperOrigin(file, native.Symbol)
		if err != nil {
			return err
		}
		original := bySymbol[root]
		if original == nil {
			return fmt.Errorf("wrapper %s root %s is not a checked operation", native.Symbol.Name, root.ID)
		}
		native.Native = append([]string(nil), original.Native...)
		native.Emitted = append([]string(nil), original.Emitted...)
	}
	return nil
}
