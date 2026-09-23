package syntax

func (f *formatter) connection(n *ConnectionDecl) {
	f.line(0, "connection "+n.Name.Text)
	for _, setting := range n.Settings {
		if setting.Name.Text == "headers" || setting.Name.Text == "metadata" {
			f.line(1, setting.Name.Text)
			for _, entry := range setting.Entries {
				separator := " "
				if setting.Name.Text == "headers" {
					separator = " = "
				}
				f.line(2, entry.Name.Text+separator+FormatExpression(entry.Value))
			}
		} else {
			prefix := setting.Name.Text + " "
			if setting.Name.Text == "auth" {
				prefix += "bearer env "
			}
			f.line(1, prefix+FormatExpression(setting.Value))
		}
	}
}
func (f *formatter) nativeAssertions(assertions []Assertion) {
	if len(assertions) > 0 {
		f.line(1, "asserts")
		for _, assertion := range assertions {
			f.assertion(2, assertion)
		}
	}
}
func (f *formatter) nativeHeader(kind string, n NativeHeader) {
	f.line(0, kind+" "+FormatType(n.Result)+" "+n.Name.Text+" from "+formatName(n.Connection))
	f.line(1, formatBound(n.Errors))
	if len(n.Inputs) > 0 {
		f.line(1, "given")
		for _, input := range n.Inputs {
			prefix := ""
			if input.Near {
				prefix = "near "
			}
			middle := " "
			if input.Variadic {
				middle = " ..."
			}
			f.line(2, prefix+FormatType(input.Type)+middle+input.Name.Text)
		}
	}
}
func (f *formatter) nativeState(fields []Field) {
	if len(fields) > 0 {
		f.line(1, "state")
		for _, field := range fields {
			f.line(2, formatField(field))
		}
	}
}
func (f *formatter) nativeEntries(section string, entries []NativeEntry) {
	if len(entries) > 0 {
		f.line(1, section)
		for _, entry := range entries {
			f.line(2, entry.Name.Text+" = "+FormatExpression(entry.Value))
		}
	}
}

func (f *formatter) question(q *QuestionDecl) {
	kind := q.Kind
	if q.RecordName != nil {
		kind = "record " + q.RecordName.Text + " " + kind
	}
	f.nativeHeader(kind, q.NativeHeader)
	for _, binder := range q.Binders {
		f.line(1, binder.Kind.Text+" as "+binder.Name.Text)
	}
	minimum := func() {
		if q.Fallback != nil {
			f.body(1, "minimum "+FormatExpression(q.Minimum)+" => ", q.Fallback)
		} else {
			f.line(1, "minimum "+FormatExpression(q.Minimum))
		}
	}
	if q.Kind == "score" && q.Minimum != nil {
		minimum()
	}
	f.line(1, "asks "+FormatExpression(q.Asks))
	if q.Kind != "score" && q.Minimum != nil {
		minimum()
	}
	for _, option := range q.Options {
		if option.Spread != nil {
			f.line(2, "..."+FormatExpression(option.Spread))
			continue
		}
		text := option.Name.Text + " " + FormatExpression(option.Description)
		if option.Body != nil {
			f.body(2, text+" => ", option.Body)
		} else {
			f.line(2, text)
		}
	}
	if q.Shared != nil {
		prefix := "ok"
		if q.Selected != nil {
			prefix += " " + formatField(*q.Selected)
		}
		f.body(2, prefix+" => ", q.Shared)
	}
}
