package emit

import (
	"regexp"
	"testing"
)

// TestActionClientInvocationShape pins the emitted request/post argument
// order the UP13 runtime adapter consumes: the authored value operands
// (captures record, then the POST wire body), the spliced client site
// object, and the trailing assertion context. The static action symbol
// never lowers to a value.
func TestActionClientInvocationShape(t *testing.T) {
	program := actionBindingsEmitProgram(t, map[string]string{
		"src/server/server.can": actionBindingsEmitServer,
		"src/client/client.can": actionBindingsEmitClient,
	})
	joined := actionBindingsBody(t, program)
	for _, want := range []string{
		`\$canActionClient\.request\(\$can\w+, \{"action":`,
		`\$canActionClient\.post\(\$can\w+, \$can\w+, \{"action":`,
	} {
		matched, err := regexp.MatchString(want, joined)
		if err != nil || !matched {
			t.Fatalf("emitted client call breaks %s", want)
		}
	}
}
