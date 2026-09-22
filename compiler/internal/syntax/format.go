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
		return "callable " + FormatExpression(n.Callee) + formatTypeArguments(n.Types)
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

type formatter struct{ strings.Builder }

func (f *formatter) line(level int, text string) {
	f.WriteString(strings.Repeat("    ", level))
	f.WriteString(text)
	f.WriteByte('\n')
}
func formatField(field Field) string      { return FormatType(field.Type) + " " + field.Name.Text }
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
	f.line(0, "package "+file.Header.Name.Text)
	provides := make([]string, len(file.Header.Provides))
	for i, p := range file.Header.Provides {
		provides[i] = p.Text
	}
	f.line(1, "provides ["+strings.Join(provides, ", ")+"]")
	uses := make([]string, len(file.Header.Uses))
	for i, u := range file.Header.Uses {
		uses[i] = u.Package.Text
		if u.Alias != nil {
			uses[i] += " as " + u.Alias.Text
		}
	}
	f.line(1, "uses ["+strings.Join(uses, ", ")+"]")
	for _, declaration := range file.Declarations {
		f.WriteByte('\n')
		switch n := declaration.(type) {
		case *JudgeDecl:
			f.nativeHeader("judge", n.NativeHeader)
			f.nativeState(n.State)
			for _, entry := range n.Registrations {
				text := FormatExpression(entry.Call)
				if entry.Binding != nil {
					text += " as " + formatField(*entry.Binding)
				}
				f.line(1, text)
			}
			f.body(1, "ok => ", n.Continuation)
		case *ChoiceArmDecl:
			f.line(0, "choice_arm "+FormatType(n.Result)+" "+n.Name.Text)
			f.line(1, formatBound(n.Errors))
			f.line(1, "describes "+FormatExpression(n.Description))
			f.block(1, n.Body)
		case *QuestionDecl:
			f.question(n)
		case *ConnectionDecl:
			f.connection(n)
		case *FetchDecl:
			f.nativeHeader("fetch", n.NativeHeader)
			f.line(1, n.Method.Text+" "+FormatExpression(n.Path))
			f.nativeEntries("query", n.Query)
			f.nativeEntries("headers", n.Headers)
			if n.BodyEncoding != nil {
				f.line(1, "body "+n.BodyEncoding.Text+" "+FormatExpression(n.Body))
			}
		case *LLMDecl:
			f.nativeHeader("llm", n.NativeHeader)
			f.nativeState(n.State)
			f.line(1, "asks "+FormatExpression(n.Asks))
		case *RecordDecl:
			f.line(0, "record "+n.Name.Text+formatParameters(n.Parameters))
			for _, field := range n.Fields {
				f.line(1, formatField(field))
			}
		case *VariantDecl:
			f.line(0, "variant "+n.Name.Text+formatParameters(n.Parameters))
			for _, t := range n.Alternatives {
				f.line(1, FormatType(t))
			}
		case *ErrorDecl:
			fields := make([]string, len(n.Fields))
			for i, field := range n.Fields {
				fields[i] = formatField(field)
			}
			f.line(0, "error "+n.ID.Text+" "+n.Name.Text+formatParameters(n.Parameters)+"("+strings.Join(fields, ", ")+")")
		case *ValueDecl:
			f.binding(0, n.Binding)
		case *FunctionDecl:
			f.line(0, "fn "+FormatType(n.Result)+" "+n.Name.Text+formatParameters(n.Parameters))
			if n.Receiver != nil {
				f.line(1, "on "+formatField(*n.Receiver))
			}
			f.line(1, formatBound(n.Errors))
			if len(n.Inputs) > 0 {
				f.line(1, "given")
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
					f.line(2, text)
				}
			}
			f.line(1, "asserts")
			for _, assertion := range n.Assertions {
				f.assertion(2, assertion)
			}
			f.block(1, n.Body)
		default:
			panic("unknown declaration")
		}
	}
	return f.String()
}

func (f *formatter) assertion(level int, a Assertion) {
	text := a.Name.Text + ": "
	if a.Receiver != nil {
		text += FormatExpression(a.Receiver) + " => "
	}
	text += formatArguments(a.Arguments) + " => "
	f.body(level, text, a.Expected)
}
func (f *formatter) binding(level int, b Binding) {
	prefix := FormatType(b.Type) + " " + b.Name.Text + " = "
	switch n := b.Value.(type) {
	case *MatchExpr:
		f.match(level, prefix, n.Match)
	case *CoordinationExpr:
		f.coordination(level, prefix, n.Coordination)
	default:
		f.line(level, prefix+FormatExpression(b.Value))
	}
}
func (f *formatter) block(level int, b Block) {
	for _, step := range b.Steps {
		switch n := step.(type) {
		case *BindingStep:
			f.binding(level, n.Binding)
		case *CallStep:
			f.line(level, FormatExpression(n.Call))
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
	f.line(level, prefix+header)
	for _, p := range c.Participants {
		if p.Spread != nil {
			f.line(level+1, "..."+FormatExpression(p.Spread))
		} else {
			f.line(level+1, strings.TrimPrefix(FormatExpression(p.Call), "call "))
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
		f.line(level, text)
	} else {
		f.body(level, text+" => ", arm.Body)
	}
}
func (f *formatter) body(level int, prefix string, body Body) {
	switch n := body.(type) {
	case *ValueBody:
		f.line(level, prefix+FormatExpression(n.Value))
	case *SuccessBody:
		text := "ok"
		if n.Value != nil {
			text += " " + FormatExpression(n.Value)
		}
		f.line(level, prefix+text)
	case *FailureBody:
		f.line(level, prefix+FormatExpression(n.Error))
	case *RelayBody:
		f.line(level, prefix+"relay "+FormatExpression(n.Call))
	case *DoBody:
		f.line(level, prefix+"do")
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
	f.line(level, prefix+header)
	for _, entry := range m.Chain {
		text := FormatExpression(entry.Call)
		if entry.Binding != nil {
			text += " as " + formatField(*entry.Binding)
		}
		f.line(level+1, text)
	}
	if len(m.When) > 0 {
		f.line(level+1, "when")
		for _, a := range m.When {
			f.assertion(level+2, a)
		}
	}
	for _, arm := range m.Arms {
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
			f.line(level+1, text)
		} else {
			f.body(level+1, text+" => ", arm.Body)
		}
	}
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
