package syntax

func (p *parser) isCoordination() bool {
	if p.pending != nil {
		return false
	}
	if p.index+3 >= len(p.tokens) {
		return false
	}
	t := p.tokens[p.index:]
	return t[0].IsWord("match") && t[1].IsWord("call") && (t[2].IsWord("concurrent") || t[2].IsWord("race")) && (t[3].Kind == Newline || t[3].IsWord("with"))
}

func (p *parser) startsInvocation() (yes bool) {
	saved := *p
	defer func() {
		*p = saved
		if caught := recover(); caught != nil {
			if _, ok := caught.(parseFailure); !ok {
				panic(caught)
			}
			yes = false
		}
	}()
	p.callee()
	p.typeArguments()
	return p.at("(")
}

func (p *parser) coordination() Coordination {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 256 {
		p.fail("coordination nesting exceeds 256 levels")
	}
	start := p.expectWord("match").Span.Start
	p.expectWord("call")
	mode := p.take().Text
	withError := p.word("with")
	if withError {
		p.take()
		p.expectWord("error")
	}
	p.expect(Newline)
	p.expect(Indent)
	c := Coordination{Mode: mode, WithError: withError}
	for p.at("...") || p.startsInvocation() {
		start := p.peek().Span.Start
		participant := Participant{}
		if p.at("...") {
			p.take()
			participant.Spread = p.expression(1)
		} else {
			participant.Call = p.invocationExpression(start)
		}
		p.expect(Newline)
		if p.at(Indent) {
			if mode == "race" {
				p.fail("race completion arms belong after all participant entries")
			}
			p.take()
			for !p.at(Dedent) && !p.at(EOF) {
				arm := p.coordinationArm()
				if !withError && !arm.Outcome.Success {
					p.fail("plain concurrent participant arms may only handle success")
				}
				participant.Arms = append(participant.Arms, arm)
			}
			p.expect(Dedent)
		}
		participant.Span = p.span(start)
		c.Participants = append(c.Participants, participant)
	}
	if len(c.Participants) == 0 {
		p.fail("coordination requires at least one participant entry")
	}
	for !p.at(Dedent) && !p.at(EOF) {
		if mode == "concurrent" && withError {
			p.fail("concurrent with error requires arms beneath each participant")
		}
		arm := p.coordinationArm()
		if mode == "concurrent" && arm.Outcome.Success {
			p.fail("concurrent success arms belong beneath each participant")
		}
		c.Arms = append(c.Arms, arm)
	}
	p.expect(Dedent)
	c.Span = p.span(start)
	return c
}

func (p *parser) coordinationArm() MatchArm {
	start := p.peek().Span.Start
	outcome := p.outcomePattern()
	arm := MatchArm{Outcome: &outcome}
	if p.at(Newline) {
		if outcome.Binding != nil || outcome.StandardFailure {
			p.fail("only bare ok or named errors can forward")
		}
		p.take()
		arm.Forward = true
	} else {
		p.expect("=>")
		arm.Body = p.armBody(true)
	}
	arm.Span = p.span(start)
	return arm
}
