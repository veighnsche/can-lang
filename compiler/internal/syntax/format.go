package syntax

import "strings"

func formatName(name QualifiedName) string {
	if name.Package != "" {
		return name.Package + "::" + name.Name
	}
	return name.Name
}
func formatTypes(types []TypeNode) string {
	parts := make([]string, len(types))
	for i, t := range types {
		parts[i] = FormatType(t)
	}
	return strings.Join(parts, ", ")
}
func formatTypeArguments(types []TypeNode) string {
	if len(types) == 0 {
		return ""
	}
	return "<" + formatTypes(types) + ">"
}
func formatArguments(arguments []Argument) string {
	parts := make([]string, len(arguments))
	for i, a := range arguments {
		if a.Group != nil {
			values := make([]string, len(a.Group.Values))
			for j, v := range a.Group.Values {
				values[j] = FormatExpression(v)
			}
			parts[i] = "(" + strings.Join(values, ", ") + ")"
		} else {
			parts[i] = FormatExpression(a.Value)
		}
		if a.Spread {
			parts[i] = "..." + parts[i]
		}
	}
	return strings.Join(parts, ", ")
}

// FormatExpression preserves authored grouping. Inserting new parentheses
// around call arguments could turn an ordinary expression into state-group
// syntax; removing them could do the reverse. The parser's explicit GroupExpr
// nodes retain the grouping needed by the original precedence structure.
func FormatExpression(expression Expr) string {
	switch n := expression.(type) {
	case *ProbabilityExpr:
		return "%"
	case *LiteralExpr:
		return n.Token.Text
	case *NameExpr:
		return formatName(n.Name)
	case *GroupExpr:
		return "(" + FormatExpression(n.Value) + ")"
	case *UnaryExpr:
		separator := ""
		if n.Operator == "not" {
			separator = " "
		}
		return n.Operator + separator + FormatExpression(n.Operand)
	case *BinaryExpr:
		return FormatExpression(n.Left) + " " + n.Operator + " " + FormatExpression(n.Right)
	case *ComparisonExpr:
		out := FormatExpression(n.Operands[0])
		for i, op := range n.Operators {
			out += " " + op + " " + FormatExpression(n.Operands[i+1])
		}
		return out
	case *ArrayExpr:
		return "[" + formatArguments(n.Elements) + "]"
	case *FieldExpr:
		separator := "."
		if literal, ok := n.Receiver.(*LiteralExpr); ok && (literal.Token.Kind == Integer || literal.Token.Kind == Float) {
			// Preserve lexical separation even in diagnostic fixtures where a
			// numeric receiver has no such field. Parentheses change the AST.
			separator = " ."
		}
		return FormatExpression(n.Receiver) + separator + n.Field.Text
	case *IndexExpr:
		return FormatExpression(n.Receiver) + "[" + FormatExpression(n.Index) + "]"
	case *SliceExpr:
		start, end := "", ""
		if n.Start != nil {
			start = FormatExpression(n.Start)
		}
		if n.End != nil {
			end = FormatExpression(n.End)
		}
		return FormatExpression(n.Receiver) + "[" + start + ":" + end + "]"
	case *ConstructorExpr:
		return formatName(n.Name) + formatTypeArguments(n.Types) + "(" + formatArguments(n.Arguments) + ")"
	case *CallExpr:
		out := "call " + FormatExpression(n.Invocation.Callee) + formatTypeArguments(n.Invocation.Types) + "(" + formatArguments(n.Invocation.Arguments) + ")"
		for _, m := range n.Methods {
			out += "." + m.Name.Text + formatTypeArguments(m.Types) + "(" + formatArguments(m.Arguments) + ")"
		}
		return out
	case *ReferenceExpr:
		out := "callable " + FormatExpression(n.Callee) + formatTypeArguments(n.Types)
		if len(n.Bindings) != 0 {
			pairs := make([]string, len(n.Bindings))
			for i, binding := range n.Bindings {
				pairs[i] = binding.Name.Text + " = " + FormatExpression(binding.Value)
			}
			out += " with " + strings.Join(pairs, ", ")
		}
		return out
	case *ScopeExpr:
		return "scope"
	case *UpdateExpr:
		fields := make([]string, len(n.Fields))
		for i, f := range n.Fields {
			fields[i] = f.Name.Text + " = " + FormatExpression(f.Value)
		}
		tail := strings.Join(fields, ", ")
		if len(fields) > 1 {
			tail = "(" + tail + ")"
		}
		return FormatExpression(n.Receiver) + " with " + tail
	default:
		panic("block or unknown expression requires the file formatter")
	}
}

type formatter struct {
	strings.Builder
	// anchors records the source byte offset each emitted line renders,
	// in emission order. Format ignores them; trivia attachment maps
	// every comment and blank run through them.
	anchors []int
	last    int
	// trivia stages lines instead of writing them so comments and blank
	// runs can splice between canonical lines; it also suppresses the
	// canonical blank separators, which the trivia renderer reproduces
	// verbatim from source.
	trivia bool
	staged []stagedLine
	// pin anchors reordered lines to one source offset so trivia
	// attachment keeps its monotonic anchor order. Only the outermost
	// reorder pins; nested lines share the pin.
	pin    int
	pinned bool
}

// stagedLine is one canonical line plus its indent level and source offset.
// Synthetic layout lines (given, asserts, when, cases) resolve their true
// keyword line during trivia attachment.
type stagedLine struct {
	level     int
	text      string
	offset    int
	synthetic bool
}

func (f *formatter) line(level int, text string, offset int) {
	if f.pinned {
		offset = f.pin
	}
	if f.trivia {
		f.staged = append(f.staged, stagedLine{level: level, text: text, offset: offset})
	} else {
		f.WriteString(strings.Repeat("    ", level))
		f.WriteString(text)
		f.WriteByte('\n')
	}
	f.anchors = append(f.anchors, offset)
	f.last = offset
}

// lineSame emits a synthetic layout line (given, asserts, when, cases).
// These keywords own no span of their own; trivia attachment resolves
// their true source line from the surrounding anchors.
func (f *formatter) lineSame(level int, text string) {
	f.line(level, text, f.last)
	if f.trivia {
		f.staged[len(f.staged)-1].synthetic = true
	}
}

// blank separates top-level declarations in trivia-free rendering. The
// trivia renderer reproduces source blank runs verbatim instead.
func (f *formatter) blank() {
	if f.trivia {
		return
	}
	f.WriteByte('\n')
}
func formatField(field Field) string { return FormatType(field.Type) + " " + field.Name.Text }
func formatActionInput(input *ActionInput) string {
	if input.Mode.Text == "input" {
		return "input none"
	}
	row := input.Mode.Text + " " + FormatType(input.Type) + " limit " + input.Limit.Text
	if input.RowsLimit != nil {
		row += " rows_limit " + input.RowsLimit.Text
	}
	return row
}
func formatBound(bound ErrorBound) string { return "emits [" + formatTypes(bound.Types) + "]" }
func formatParameters(parameters []Token) string {
	if len(parameters) == 0 {
		return ""
	}
	names := make([]string, len(parameters))
	for i, p := range parameters {
		names[i] = p.Text
	}
	return "<" + strings.Join(names, ", ") + ">"
}

// Format renders the explicit syntax tree, without evaluating bodies or looking
// up symbols. Comments remain available as exact source trivia on File.Comments;
// canonical rendering intentionally excludes trivia.
func Format(file *File) string {
	f := &formatter{}
	f.render(file)
	return f.String()
}

// render emits every declaration through line calls so both Format and the
// trivia renderer share one canonical layout.
func (f *formatter) render(file *File) {
	f.line(0, "package "+file.Header.Name.Text, file.Header.Name.Span.Start)
	provides := make([]string, len(file.Header.Provides))
	for i, p := range file.Header.Provides {
		provides[i] = p.Text
	}
	if len(file.Header.Provides) != 0 {
		f.line(1, "provides ["+strings.Join(provides, ", ")+"]", file.Header.Provides[0].Span.Start)
	} else {
		f.lineSame(1, "provides []")
	}
	uses := make([]string, len(file.Header.Uses))
	for i, u := range file.Header.Uses {
		uses[i] = u.Package.Text
		if u.Dependency != nil {
			uses[i] = u.Dependency.Text + "::" + uses[i]
		}
		if u.Alias != nil {
			uses[i] += " as " + u.Alias.Text
		}
	}
	if len(file.Header.Uses) != 0 {
		f.line(1, "uses ["+strings.Join(uses, ", ")+"]", file.Header.Uses[0].Span.Start)
	} else {
		f.lineSame(1, "uses []")
	}
	for _, declaration := range file.Declarations {
		f.blank()
		switch n := declaration.(type) {
		case *JudgeDecl:
			f.nativeHeader("judge", n.NativeHeader)
			f.nativeState(n.State)
			f.nativeAssertions(n.Assertions)
			for _, entry := range n.Registrations {
				text := FormatExpression(entry.Call)
				if entry.Binding != nil {
					text += " as " + formatField(*entry.Binding)
				}
				f.line(1, text, entry.Span.Start)
			}
			f.body(1, "ok => ", n.Continuation)
		case *ChoiceArmDecl:
			f.line(0, "choice_arm "+FormatType(n.Result)+" "+n.Name.Text, n.DeclSpan().Start)
			f.line(1, formatBound(n.Errors), n.Errors.Span.Start)
			f.line(1, "describes "+FormatExpression(n.Description), n.Description.ExprSpan().Start)
			f.block(1, n.Body)
		case *QuestionDecl:
			f.question(n)
		case *ConnectionDecl:
			f.connection(n)
		case *FetchDecl:
			f.nativeHeader("fetch", n.NativeHeader)
			f.nativeAssertions(n.Assertions)
			f.line(1, n.Method.Text+" "+FormatExpression(n.Path), n.Method.Span.Start)
			f.nativeEntries("query", n.Query)
			f.nativeEntries("headers", n.Headers)
			if n.BodyEncoding != nil {
				f.line(1, "body "+n.BodyEncoding.Text+" "+FormatExpression(n.Body), n.BodyEncoding.Span.Start)
			}
		case *LLMDecl:
			f.nativeHeader("llm", n.NativeHeader)
			f.nativeState(n.State)
			f.nativeAssertions(n.Assertions)
			f.line(1, "asks "+FormatExpression(n.Asks), n.Asks.ExprSpan().Start)
		case *WrapDecl:
			f.line(0, "wrap "+n.Name.Text+" from "+formatName(n.Base), n.DeclSpan().Start)
			f.lineSame(1, "emits calculated")
			f.nativeAssertions(n.Assertions)
			f.wrapArms("native", n.Native)
			f.wrapArms("emitted", n.Emitted)
		case *RecordDecl:
			header := "record " + n.Name.Text + formatParameters(n.Parameters)
			if n.Owner {
				header = "owner " + header
			}
			f.line(0, header, n.DeclSpan().Start)
			for _, field := range n.Fields {
				f.line(1, formatField(field), field.Span.Start)
			}
		case *VariantDecl:
			f.line(0, "variant "+n.Name.Text+formatParameters(n.Parameters), n.DeclSpan().Start)
			for _, t := range n.Alternatives {
				f.line(1, FormatType(t), t.TypeSpan().Start)
			}
		case *ErrorDecl:
			fields := make([]string, len(n.Fields))
			for i, field := range n.Fields {
				fields[i] = formatField(field)
			}
			f.line(0, "error "+n.Name.Text+formatParameters(n.Parameters)+"("+strings.Join(fields, ", ")+")", n.DeclSpan().Start)
		case *ValueDecl:
			f.binding(0, n.Binding)
		case *FixtureDecl:
			header := "fixture " + n.Name.Text + " for " + formatName(n.Target)
			if len(n.Types) != 0 {
				types := make([]string, len(n.Types))
				for i, t := range n.Types {
					types[i] = FormatType(t)
				}
				header += "<" + strings.Join(types, ", ") + ">"
			}
			f.line(0, header, n.DeclSpan().Start)
			if len(n.Given) != 0 {
				f.lineSame(1, "given")
				for _, field := range n.Given {
					f.line(2, formatField(field), field.Span.Start)
				}
			}
			f.lineSame(1, "cases")
			for _, kase := range n.Cases {
				prefix := formatArguments(kase.Arguments)
				if prefix != "" {
					prefix += " "
				}
				f.body(2, prefix+"=> ", kase.Expected)
				if kase.Mode != nil {
					if kase.Mode.Failure != nil {
						f.line(3, "using failure "+kase.Mode.Failure.Origin.Text+" "+FormatExpression(kase.Mode.Failure.Value), kase.Mode.Span.Start)
					} else {
						f.line(3, "using raw "+kase.Mode.Raw.Text, kase.Mode.Span.Start)
					}
				}
			}
		case *ScenarioDecl:
			f.line(0, "scenario "+n.Name.Text, n.DeclSpan().Start)
		case *ActionDecl:
			f.line(0, "action "+n.Name.Text, n.DeclSpan().Start)
			f.line(1, n.Method.Text+" "+n.Path.Text, n.Method.Span.Start)
			if n.Captures != nil {
				f.line(1, "captures "+FormatType(n.Captures), n.Captures.TypeSpan().Start)
			}
			f.line(1, formatActionInput(n.Input), n.Input.Span.Start)
			f.line(1, "returns "+FormatType(n.Returns), n.Returns.TypeSpan().Start)
			f.line(1, "body "+n.Response.Text, n.Response.Span.Start)
			f.lineSame(1, "cases")
			for _, kase := range n.Cases {
				row := formatName(kase.Leaf) + " status " + kase.Status.Text
				if kase.Swap != nil {
					row += " swap inner"
				}
				f.line(2, row, kase.Span.Start)
			}
		case *FunctionDecl:
			f.line(0, "fn "+FormatType(n.Result)+" "+n.Name.Text+formatParameters(n.Parameters), n.DeclSpan().Start)
			if n.Receiver != nil {
				f.line(1, "on "+formatField(*n.Receiver), n.Receiver.Span.Start)
			}
			f.line(1, formatBound(n.Errors), n.Errors.Span.Start)
			if len(n.Inputs) > 0 {
				f.lineSame(1, "given")
				for _, input := range n.Inputs {
					text := ""
					if input.Near {
						text = "near "
					}
					text += FormatType(input.Type) + " "
					if input.Variadic {
						text += "..."
					}
					text += input.Name.Text
					f.line(2, text, input.Span.Start)
				}
			}
			f.lineSame(1, "asserts")
			for _, assertion := range n.Assertions {
				f.assertion(2, assertion)
			}
			f.block(1, n.Body)
		default:
			panic("unknown declaration")
		}
	}
}

func (f *formatter) assertion(level int, a Assertion) {
	prefix := ""
	if a.Scenario != nil {
		prefix = "scenario "
	}
	if a.Use != nil {
		f.line(level, prefix+a.Name.Text+": use "+formatName(a.Use.Template)+"("+formatArguments(a.Use.Arguments)+")", a.Span.Start)
		return
	}
	text := prefix + a.Name.Text + ": "
	if a.Receiver != nil {
		text += FormatExpression(a.Receiver) + " => "
	}
	text += formatArguments(a.Arguments) + " => "
	if len(a.Links) != 0 {
		targets := make([]string, len(a.Links))
		for i, link := range a.Links {
			targets[i] = formatName(link)
		}
		f.line(level, text+formatCompletion(a.Expected)+" link "+strings.Join(targets, ", "), a.Span.Start)
	} else {
		f.body(level, text, a.Expected)
	}
	if a.Mode != nil {
		if a.Mode.Failure != nil {
			f.line(level+1, "using failure "+a.Mode.Failure.Origin.Text+" "+FormatExpression(a.Mode.Failure.Value), a.Mode.Span.Start)
		} else {
			f.line(level+1, "using raw "+a.Mode.Raw.Text, a.Mode.Span.Start)
		}
	}
}

// formatCompletion renders the single-line completions the row parser
// admits, so link clauses stay on the row's physical line.
func formatCompletion(body Body) string {
	switch n := body.(type) {
	case *SuccessBody:
		text := "ok"
		if n.Value != nil {
			text += " " + FormatExpression(n.Value)
		}
		return text
	case *FailureBody:
		return FormatExpression(n.Error)
	default:
		panic("row completion must be ok or an error constructor")
	}
}
func (f *formatter) binding(level int, b Binding) {
	prefix := FormatType(b.Type) + " " + b.Name.Text + " = "
	switch n := b.Value.(type) {
	case *MatchExpr:
		f.match(level, prefix, n.Match)
	case *CoordinationExpr:
		f.coordination(level, prefix, n.Coordination)
	default:
		f.line(level, prefix+FormatExpression(b.Value), b.Span.Start)
	}
}
func (f *formatter) block(level int, b Block) {
	for _, step := range b.Steps {
		switch n := step.(type) {
		case *BindingStep:
			f.binding(level, n.Binding)
		case *CallStep:
			f.line(level, FormatExpression(n.Call), n.Call.ExprSpan().Start)
		case *CoordinationStep:
			f.coordination(level, "", n.Coordination)
		default:
			panic("unknown step")
		}
	}
	f.body(level, "", b.Terminal)
}

func (f *formatter) coordination(level int, prefix string, c Coordination) {
	header := "match call " + c.Mode
	if c.WithError {
		header += " with error"
	}
	f.line(level, prefix+header, c.Span.Start)
	for _, p := range c.Participants {
		if p.Spread != nil {
			f.line(level+1, "..."+FormatExpression(p.Spread), p.Span.Start)
		} else {
			f.line(level+1, strings.TrimPrefix(FormatExpression(p.Call), "call "), p.Span.Start)
		}
		for _, arm := range p.Arms {
			f.coordinationArm(level+2, arm)
		}
	}
	for _, arm := range c.Arms {
		f.coordinationArm(level+1, arm)
	}
}
func (f *formatter) coordinationArm(level int, arm MatchArm) {
	o := arm.Outcome
	text := ""
	switch {
	case o.Success:
		text = "ok"
	case o.StandardFailure:
		text = "[_]"
	default:
		text = FormatType(o.Error)
	}
	if o.Binding != nil {
		if o.StandardFailure {
			text += " as"
		}
		text += " " + formatField(*o.Binding)
	}
	if o.Alias != nil {
		text += " as " + o.Alias.Text
	}
	if arm.Forward {
		f.line(level, text, arm.Span.Start)
	} else {
		f.body(level, text+" => ", arm.Body)
	}
}
func (f *formatter) body(level int, prefix string, body Body) {
	switch n := body.(type) {
	case *ValueBody:
		f.line(level, prefix+FormatExpression(n.Value), n.Value.ExprSpan().Start)
	case *SuccessBody:
		text := "ok"
		if n.Value != nil {
			text += " " + FormatExpression(n.Value)
		}
		f.line(level, prefix+text, n.BodySpan().Start)
	case *FailureBody:
		f.line(level, prefix+FormatExpression(n.Error), n.Error.ExprSpan().Start)
	case *RelayBody:
		f.line(level, prefix+"relay "+FormatExpression(n.Call), n.Call.ExprSpan().Start)
	case *InheritBody:
		f.line(level, prefix+"inherit", n.BodySpan().Start)
	case *DoBody:
		f.line(level, prefix+"do", n.BodySpan().Start)
		f.block(level+1, n.Block)
	case *MatchBody:
		f.match(level, prefix, n.Match)
	default:
		panic("unknown arm body")
	}
}
func (f *formatter) match(level int, prefix string, m Match) {
	header := "match "
	switch m.Kind {
	case ValueMatch:
		values := make([]string, len(m.Values))
		for i, v := range m.Values {
			values[i] = FormatExpression(v)
		}
		header += strings.Join(values, ", ")
	case CallMatch:
		header += FormatExpression(m.Call)
	case ChainMatch:
		header += "chain"
	}
	f.line(level, prefix+header, m.Span.Start)
	for _, entry := range m.Chain {
		text := FormatExpression(entry.Call)
		if entry.Binding != nil {
			text += " as " + formatField(*entry.Binding)
		}
		f.line(level+1, text, entry.Span.Start)
	}
	if len(m.When) > 0 {
		f.lineSame(level+1, "when")
		for _, a := range m.When {
			f.assertion(level+2, a)
		}
	}
	// Q1: canonicalize ordinary single-scrutinee true-first Boolean pairs
	// to false-first. Only the exact complementary literal pair reorders,
	// so wildcards, alternatives and multi-arm matches keep their
	// semantics and positions; reordered lines share one anchor.
	arms := m.Arms
	if m.Kind == ValueMatch && len(m.Values) == 1 && len(arms) == 2 && booleanLiteralArm(arms[0], "true") && booleanLiteralArm(arms[1], "false") {
		arms = []MatchArm{arms[1], arms[0]}
		if !f.pinned {
			f.pin, f.pinned = m.Arms[0].Span.Start, true
			defer func() { f.pinned = false }()
		}
	}
	for _, arm := range arms {
		text := ""
		if arm.Outcome != nil {
			o := arm.Outcome
			switch {
			case o.Success:
				text = "ok"
			case o.StandardFailure:
				text = "[_]"
			default:
				text = FormatType(o.Error)
			}
			if o.Binding != nil {
				if o.StandardFailure {
					text += " as"
				}
				text += " " + formatField(*o.Binding)
			}
			if o.Alias != nil {
				text += " as " + o.Alias.Text
			}
		} else {
			patterns := make([]string, len(arm.Patterns))
			for i, p := range arm.Patterns {
				patterns[i] = formatPattern(p)
			}
			text = strings.Join(patterns, ", ")
		}
		if arm.Forward {
			f.line(level+1, text, arm.Span.Start)
		} else {
			f.body(level+1, text+" => ", arm.Body)
		}
	}
}

// booleanLiteralArm reports whether the arm is one bare true/false literal
// data pattern: the only shape the Q1 canonicalization reorders.
func booleanLiteralArm(arm MatchArm, text string) bool {
	if arm.Outcome != nil || arm.Forward || len(arm.Patterns) != 1 {
		return false
	}
	literal, ok := arm.Patterns[0].(*LiteralPattern)
	return ok && !literal.Negative && literal.Literal.Text == text
}

func formatPattern(pattern PatternNode) string {
	switch n := pattern.(type) {
	case *WildcardPattern:
		return "_"
	case *LiteralPattern:
		if n.Negative {
			return "-" + n.Literal.Text
		}
		return n.Literal.Text
	case *NamePattern:
		return formatName(n.Name) + formatTypeArguments(n.Types)
	case *BindPattern:
		return "bind " + n.Name.Text
	case *ConstructorPattern:
		fields := make([]string, len(n.Fields))
		for i, p := range n.Fields {
			fields[i] = formatPattern(p)
		}
		return formatName(n.Name) + formatTypeArguments(n.Types) + "(" + strings.Join(fields, ", ") + ")"
	case *ArrayPattern:
		fields := make([]string, len(n.Elements))
		for i, p := range n.Elements {
			fields[i] = formatPattern(p)
		}
		if n.Rest != nil {
			fields = append(fields, "..."+n.Rest.Text)
		}
		return "[" + strings.Join(fields, ", ") + "]"
	case *RangePattern:
		return formatPattern(n.Lower) + ".." + formatPattern(n.Upper)
	case *AlternativePattern:
		patterns := make([]string, len(n.Alternatives))
		for i, p := range n.Alternatives {
			patterns[i] = formatPattern(p)
		}
		return strings.Join(patterns, " | ")
	default:
		panic("unknown pattern")
	}
}
