package check

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// UP17: the locked generic chain checks to exactly the reachable canonical
// instance set. Four request sites for nested<int> (its assertion row, the
// row's synthesized actual invocation, plus inferred and explicit calls
// from the root package) share one instance, and identity<box<int>> is
// shared between the nested<int> body and the root's direct inferred call.
// Every edge below crosses the dependency lock; no symbolic proof or
// opaque parameter leaks into the checked program.
func TestGenericLockedChainInstances(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile("../../testdata/current/generics/" + name + ".can")
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	deps := map[string]coreDep{"vendor": {files: map[string]string{
		"core/core.can":       read("chain-core"),
		"helpers/helpers.can": read("chain-helpers"),
		"mail/mail.can":       read("chain-mail"),
	}, registry: `{"active":["core::denied"],"retired":[]}`}}
	_, program, err := coreFixture(t, map[string]string{"src/app/main.can": read("chain-main")}, deps)
	if err != nil {
		t.Fatal(err)
	}
	byName := symbolicInstances(t, program)
	if len(program.Functions) != 14 {
		t.Fatalf("checked %d functions, want 14 canonical instances", len(program.Functions))
	}
	// describe renders one-letter-per-argument instance shapes: i for int,
	// b for box<...>, e for the owner email, - for non-generic functions.
	describe := func(fn *ProgramFunction) string {
		if len(fn.TypeArguments) == 0 {
			return "-"
		}
		var out strings.Builder
		for _, argument := range fn.TypeArguments {
			switch declaration := argument.Declaration(); {
			case declaration == "int":
				out.WriteByte('i')
			case strings.HasSuffix(declaration, "::box"):
				out.WriteByte('b')
			case strings.HasSuffix(declaration, "::email"):
				out.WriteByte('e')
			default:
				t.Fatalf("instance %s carries unexpected argument %s", fn.Instance, declaration)
			}
		}
		return out.String()
	}
	shapes := map[string][]string{}
	for name, fns := range byName {
		for _, fn := range fns {
			shapes[name] = append(shapes[name], describe(fn))
		}
	}
	want := map[string][]string{
		"identity": {"i", "b", "e"}, "refuse": {"i"},
		"pass": {"i", "e"}, "wrap": {"i", "e"}, "nested": {"i"}, "probe": {"i"},
		"top": {"i"}, "make_email": {"-"}, "email_address": {"-"}, "main": {"-"},
	}
	if len(shapes) != len(want) {
		t.Fatalf("instance declarations = %v, want %v", shapes, want)
	}
	for name, instances := range want {
		got := append([]string(nil), shapes[name]...)
		sort.Strings(got)
		sorted := append([]string(nil), instances...)
		sort.Strings(sorted)
		if strings.Join(got, ",") != strings.Join(sorted, ",") {
			t.Fatalf("%s instances = [%s], want [%s]", name, strings.Join(got, ","), strings.Join(sorted, ","))
		}
	}
	find := func(name, shape string) *ProgramFunction {
		t.Helper()
		for _, fn := range byName[name] {
			if describe(fn) == shape {
				return fn
			}
		}
		t.Fatalf("no %s<%s> instance", name, shape)
		return nil
	}
	// nested<int> is requested from four sites but checked once: its
	// assertion row, the row's synthesized actual invocation, and the
	// inferred plus explicit root calls.
	nested := find("nested", "i")
	if len(nested.Requests) != 4 {
		t.Fatalf("nested<int> has %d requests, want 4", len(nested.Requests))
	}
	sites := strings.Join(nested.Requests, "\n")
	if strings.Count(sites, "main.can") != 2 || strings.Count(sites, "helpers.can") != 2 {
		t.Fatalf("nested<int> requests miss a call site:\n%s", sites)
	}
	// The boxed identity is shared between the nested<int> body and the
	// root's direct inferred call: two syntactic paths, one instance.
	boxed := find("identity", "b")
	if len(boxed.TypeArguments[0].Arguments()) != 1 || boxed.TypeArguments[0].Arguments()[0].Declaration() != "int" {
		t.Fatalf("identity box argument is not box<int>: %s", boxed.TypeArguments[0].Declaration())
	}
	if len(boxed.Requests) != 2 {
		t.Fatalf("identity<box<int>> has %d requests, want 2", len(boxed.Requests))
	}
	boxedSites := strings.Join(boxed.Requests, "\n")
	if !strings.Contains(boxedSites, "helpers.can") || !strings.Contains(boxedSites, "main.can") {
		t.Fatalf("identity<box<int>> requests miss a call path:\n%s", boxedSites)
	}
	// A reached nested<int> directly calls concrete identity<box<int>>;
	// probe<int> relays concrete refuse<int>. No call targets a proof.
	edge := func(caller, callee *ProgramFunction) {
		t.Helper()
		found := false
		for _, target := range symbolicCallTargets(caller.Region) {
			if target == callee.Instance {
				found = true
			}
			if strings.Contains(target, "/symbolic") {
				t.Fatalf("%s calls symbolic proof %s", caller.Instance, target)
			}
		}
		if !found {
			t.Fatalf("%s never calls %s", caller.Instance, callee.Instance)
		}
	}
	edge(nested, boxed)
	edge(find("probe", "i"), find("refuse", "i"))
	edge(find("wrap", "e"), find("pass", "e"))
	edge(find("pass", "e"), find("identity", "e"))
}
