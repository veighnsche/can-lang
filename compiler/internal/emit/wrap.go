package emit

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Wrapper renders one operation wrapper dispatcher: invoke the base exactly
// once (or consume an injected assertion failure), bypass success and let
// standard faults escape, then dispatch domain failures by (origin, exact
// error identity) to the effective rule. Unhandled keys fall through to the
// origin default: the already normalized failure forwards for native keys,
// the failure forwards unchanged for emitted keys.
func (e *RegionEmitter) Wrapper(name string, native *check.NativeDeclaration, functions, names map[string]string) (string, error) {
	plan := native.Wrapper
	if plan == nil || native.Signature == nil || !jsBinding.MatchString(name) {
		return "", fmt.Errorf("invalid wrapper plan")
	}
	base, ok := functions[plan.Base]
	if !ok {
		return "", fmt.Errorf("missing wrapper base %s", plan.Base)
	}
	inputs := native.Signature.Inputs()
	if len(inputs) != len(native.Descriptor.Names) {
		return "", fmt.Errorf("wrapper input names do not fit its inherited signature")
	}
	var args []string
	var forward []string
	for i, typ := range inputs {
		if !types.Equal(typ, typ) {
			return "", fmt.Errorf("invalid wrapper input")
		}
		param := fmt.Sprintf("$canArg%d", i)
		args = append(args, param+": "+TypeName(typ))
		forward = append(forward, param)
	}
	args = append(args, "$canContext?: $canAssertionContext")
	span := native.Symbol.Declaration.DeclSpan()
	e.region = &ir.Region{ID: native.Symbol.ID, Source: native.Symbol.Source.ID, Span: span}
	origin := e.origin(span)
	prefix := ""
	if e.SourceID != "" {
		prefix = mappingMark(e.SourceID, span, "function") + "let $canOrigin = " + origin + ";\n"
		origin = "$canOrigin"
	}
	var dispatch strings.Builder
	first := true
	for _, rule := range plan.Rules {
		if rule.Default || rule.Region == nil {
			continue
		}
		target := names[rule.Region.ID]
		if target == "" {
			return "", fmt.Errorf("missing wrapper rule %s", rule.Region.ID)
		}
		keyword := "else if"
		if first {
			keyword = "if"
			first = false
		}
		if len(rule.Region.Inputs) == 0 {
			return "", fmt.Errorf("wrapper rule %s binds no error value", rule.Region.ID)
		}
		payload := TypeName(rule.Region.Inputs[len(rule.Region.Inputs)-1].Type)
		value := "($canErrorPayload($canOutcome) as " + payload + ")"
		if rule.Key.Origin == check.WrapperNative {
			value = "($canPolicyLeaf($canOutcome) as " + payload + ")"
		}
		call := append(append([]string(nil), forward...), value, "$canOutcome", "$canContext")
		fmt.Fprintf(&dispatch, "%s ($canKey.origin === %s && $canKey.identity === %s) { return await %s(%s); }\n", keyword, quote(rule.Key.Origin), quote(rule.Key.Identity), target, strings.Join(call, ", "))
	}
	body := fmt.Sprintf("const $canOutcome = await $canPolicyBase($canContext, %s, %s, () => %s(%s));\nif ($canOutcome.kind !== 'domain') return $canOutcome;\nconst $canKey = $canPolicyKey($canOutcome, %s);\n%sreturn $canOutcome;\n", quote(native.Symbol.ID), origin, base, strings.Join(append(forward, "$canContext"), ", "), quote(plan.Failed), dispatch.String())
	return fmt.Sprintf("async function %s(%s): Promise<$canCompletion<%s>> {\n%stry {\n%s} catch ($canCause) { return $canCaught($canCause, %s); }\n}\n", name, strings.Join(args, ", "), TypeName(native.Signature.Result()), prefix, body, origin), nil
}

// WrapperRule renders one policy rule region. Rule regions take the wrapper
// inputs, the bound error payload and the preserved original outcome before
// the assertion context, so inherit forwards the same occurrence.
func (e *RegionEmitter) WrapperRule(name string, region *ir.Region) (string, error) {
	if region == nil || region.ID == "" || region.Body == nil || !types.Equal(region.Result, region.Result) || !jsBinding.MatchString(name) {
		return "", fmt.Errorf("invalid checked region")
	}
	args, err := e.configure(region)
	if err != nil {
		return "", err
	}
	// configure appends the optional assertion context last; the original
	// outcome is a required parameter ahead of it.
	args = append(args[:len(args)-1], "$canOriginal: $canCompletion<"+TypeName(region.Result)+">", args[len(args)-1])
	body, err := e.block(region.Body)
	if err != nil {
		return "", err
	}
	origin := e.origin(region.Span)
	prefix := ""
	if e.SourceID != "" {
		prefix = mappingMark(e.SourceID, region.Span, "function") + "let $canOrigin = " + origin + ";\n"
		origin = "$canOrigin"
	}
	return fmt.Sprintf("async function %s(%s): Promise<$canCompletion<%s>> {\n%stry {\n%s} catch ($canCause) { return $canCaught($canCause, %s); }\n}\n", name, strings.Join(args, ", "), TypeName(region.Result), prefix, body, origin), nil
}
