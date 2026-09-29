package emit

import (
	"strings"
	"testing"
)

func TestEmailHrefFixtureBindsToHTMLRuntime(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/html/main.can")
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if strings.Contains(string(artifact.Bytes), "$canHTML.emailHref") {
			return
		}
	}
	t.Fatal("authored html::email_href did not bind to the runtime constructor")
}
