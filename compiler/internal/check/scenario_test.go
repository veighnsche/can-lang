package check

import (
	"strings"
	"testing"
)

const scenarioHelper = "package helper\n    provides [read, checkout]\n    uses [text]\nscenario checkout\nfn str read\n    emits []\n    asserts\n        unit: => ok \"fixture\"\n    match call text::from_int(7)\n        when\n            scenario checkout: 7 => ok \"fixture\"\n            unit: 7 => ok \"fixture\"\n        ok str result => ok result\n"

const scenarioAppMain = "fn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"

func scenarioApp(link string) string {
	return "package app\n    provides []\n    uses [helper]\nfn str read_customer\n    emits []\n    asserts\n        customer: => ok \"fixture\"" + link + "\n    ok call helper::read()\n" + scenarioAppMain
}

func scenarioPackageID(t *testing.T, program *Program, name string) string {
	t.Helper()
	for _, function := range program.Functions {
		if function.Symbol.Package.Name == name {
			return function.Symbol.Package.ID
		}
	}
	t.Fatalf("package %s missing", name)
	return ""
}

func scenarioRootLinks(t *testing.T, program *Program, name string) (string, []string) {
	t.Helper()
	for _, assertion := range program.Assertions {
		if assertion.Root.Name == name {
			return assertion.Root.Package, assertion.Root.Links
		}
	}
	t.Fatalf("assertion root %s missing", name)
	return "", nil
}

func TestScenarioLinkResolves(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/helper/helper.can": scenarioHelper,
		"src/app/main.can":      scenarioApp(" link helper::checkout"),
	})
	if err != nil {
		t.Fatal(err)
	}
	appID := scenarioPackageID(t, program, "app")
	helperID := scenarioPackageID(t, program, "helper")
	if appID == helperID {
		t.Fatal("app and helper share one package identity")
	}
	owner, links := scenarioRootLinks(t, program, "customer")
	if owner != appID {
		t.Fatalf("customer root owned by %s, want %s", owner, appID)
	}
	if len(links) != 1 || links[0] != helperID+"::checkout" {
		t.Fatalf("customer links = %v, want [%s::checkout]", links, helperID)
	}
	if _, links := scenarioRootLinks(t, program, "unit"); len(links) != 0 {
		t.Fatalf("helper unit root gained links %v", links)
	}
	step := templateFixtureSteps(t, program, "read")
	if len(step.Fixtures.Rows) != 2 {
		t.Fatalf("helper table has %d rows", len(step.Fixtures.Rows))
	}
	tagged, plain := step.Fixtures.Rows[0], step.Fixtures.Rows[1]
	if tagged.Scenario != helperID+"::checkout" || tagged.Owner != helperID || tagged.Selector != "checkout" {
		t.Fatalf("tagged row lost its scenario identity: %+v", tagged)
	}
	if plain.Scenario != "" || plain.Owner != helperID || plain.Selector != "unit" {
		t.Fatalf("plain row lost its same-owner shape: %+v", plain)
	}
}

func TestScenarioLinkSamePackageNeedsNoExport(t *testing.T) {
	helper := strings.Replace(scenarioHelper, "provides [read, checkout]", "provides [read]", 1)
	helper = strings.Replace(helper, "unit: => ok \"fixture\"", "unit: => ok \"fixture\"\n        linked: => ok \"fixture\" link checkout", 1)
	program, err := programFixture(t, map[string]string{
		"src/helper/helper.can": helper,
		"src/app/main.can":      scenarioApp(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	helperID := scenarioPackageID(t, program, "helper")
	owner, links := scenarioRootLinks(t, program, "linked")
	if owner != helperID || len(links) != 1 || links[0] != helperID+"::checkout" {
		t.Fatalf("same-package link resolved to %s %v", owner, links)
	}
}

func TestScenarioLinkStale(t *testing.T) {
	renamed := strings.Replace(scenarioHelper, "scenario checkout", "scenario checkout_v2", 1)
	renamed = strings.Replace(renamed, "provides [read, checkout]", "provides [read, checkout_v2]", 1)
	renamed = strings.Replace(renamed, "scenario checkout:", "scenario checkout_v2:", 1)
	aliased := "package app\n    provides []\n    uses [helper as h]\nfn str read_customer\n    emits []\n    asserts\n        customer: => ok \"fixture\" link helper::checkout\n    ok call h::read()\n" + scenarioAppMain
	private := strings.Replace(scenarioHelper, "provides [read, checkout]", "provides [read]", 1)
	for name, tc := range map[string]struct {
		helper, app, want string
	}{
		"unknown scenario":  {scenarioHelper, scenarioApp(" link helper::missing"), "stale scenario link"},
		"renamed scenario":  {renamed, scenarioApp(" link helper::checkout"), "stale scenario link"},
		"renamed qualifier": {scenarioHelper, aliased, "stale scenario link"},
		"unexported":        {private, scenarioApp(" link helper::checkout"), "does not export"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{
				"src/helper/helper.can": tc.helper,
				"src/app/main.can":      tc.app,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("stale link admitted: %v", err)
			}
		})
	}
}

func TestScenarioLinkRespectsAlias(t *testing.T) {
	app := "package app\n    provides []\n    uses [helper as h]\nfn str read_customer\n    emits []\n    asserts\n        customer: => ok \"fixture\" link h::checkout\n    ok call h::read()\n" + scenarioAppMain
	program, err := programFixture(t, map[string]string{
		"src/helper/helper.can": scenarioHelper,
		"src/app/main.can":      app,
	})
	if err != nil {
		t.Fatal(err)
	}
	helperID := scenarioPackageID(t, program, "helper")
	if _, links := scenarioRootLinks(t, program, "customer"); len(links) != 1 || links[0] != helperID+"::checkout" {
		t.Fatalf("aliased link resolved to %v", links)
	}
}

func TestScenarioLinkAmbiguous(t *testing.T) {
	left := "package left\n    provides [flow]\n    uses []\nscenario flow\nfn int ping\n    emits []\n    asserts\n        sample: => ok 1\n    ok 1\n"
	right := "package right\n    provides [flow]\n    uses []\nscenario flow\nfn int pong\n    emits []\n    asserts\n        sample: => ok 2\n    ok 2\n"
	app := func(link string) string {
		return "package app\n    provides []\n    uses [left, right]\nfn int both\n    emits []\n    asserts\n        combined: => ok 3" + link + "\n    ok 3\n" + scenarioAppMain
	}
	if _, err := programFixture(t, map[string]string{
		"src/left/left.can":   left,
		"src/right/right.can": right,
		"src/app/main.can":    app(" link flow"),
	}); err == nil || !strings.Contains(err.Error(), "ambiguous scenario link") {
		t.Fatalf("ambiguous link admitted: %v", err)
	}
	program, err := programFixture(t, map[string]string{
		"src/left/left.can":   left,
		"src/right/right.can": right,
		"src/app/main.can":    app(" link left::flow"),
	})
	if err != nil {
		t.Fatal(err)
	}
	leftID := scenarioPackageID(t, program, "left")
	if _, links := scenarioRootLinks(t, program, "combined"); len(links) != 1 || links[0] != leftID+"::flow" {
		t.Fatalf("qualified link resolved to %v", links)
	}
}

func TestScenarioPlacementRejects(t *testing.T) {
	taggedAssert := strings.Replace(scenarioHelper, "unit: => ok \"fixture\"", "scenario checkout: => ok \"fixture\"", 1)
	linkedWhen := strings.Replace(scenarioHelper, "unit: 7 => ok \"fixture\"", "unit: 7 => ok \"fixture\" link checkout", 1)
	unknownTag := strings.Replace(scenarioHelper, "scenario checkout:", "scenario missing:", 1)
	duplicate := scenarioApp(" link helper::checkout, helper::checkout")
	for name, tc := range map[string]struct {
		helper, app, want string
	}{
		"tag in asserts":   {taggedAssert, scenarioApp(""), "belong to when tables"},
		"link in when":     {linkedWhen, scenarioApp(""), "not when rows"},
		"unknown tag":      {unknownTag, scenarioApp(""), "not declared in this package"},
		"duplicate link":   {scenarioHelper, duplicate, "duplicate scenario link"},
		"non-scenario use": {scenarioHelper, scenarioApp(" link helper::read"), "stale scenario link"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{
				"src/helper/helper.can": tc.helper,
				"src/app/main.can":      tc.app,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("misplaced scenario form admitted: %v", err)
			}
		})
	}
}

func TestScenarioTwoSequentialLinks(t *testing.T) {
	helper := "package helper\n    provides [read, checkout, retry]\n    uses [text]\nscenario checkout\nscenario retry\nfn str read\n    emits []\n    asserts\n        unit: => ok \"fixture\"\n    match call text::from_int(7)\n        when\n            scenario checkout: 7 => ok \"first\"\n            scenario retry: 7 => ok \"second\"\n        ok str result => ok result\n"
	app := "package app\n    provides []\n    uses [helper]\nfn str read_customer\n    emits []\n    asserts\n        customer: => ok \"first\" link helper::checkout, helper::retry\n    ok call helper::read()\n" + scenarioAppMain
	program, err := programFixture(t, map[string]string{
		"src/helper/helper.can": helper,
		"src/app/main.can":      app,
	})
	if err != nil {
		t.Fatal(err)
	}
	helperID := scenarioPackageID(t, program, "helper")
	_, links := scenarioRootLinks(t, program, "customer")
	if len(links) != 2 || links[0] != helperID+"::checkout" || links[1] != helperID+"::retry" {
		t.Fatalf("sequential links = %v", links)
	}
	step := templateFixtureSteps(t, program, "read")
	if len(step.Fixtures.Rows) != 2 || step.Fixtures.Rows[0].Scenario != links[0] || step.Fixtures.Rows[1].Scenario != links[1] {
		t.Fatalf("sequential rows lost their scenario order: %+v", step.Fixtures.Rows)
	}
}

func TestScenarioUseRowInheritsTag(t *testing.T) {
	source := programHeader + "fixture doubled for double\n    cases\n        2 => ok 4\nfn int double\n    emits []\n    given\n        int value\n    asserts\n        sample: 1 => ok 2\n    ok value + value\nscenario pair\nfn int first_use\n    emits []\n    asserts\n        paired: => ok 4 link pair\n    match call double(2)\n        when\n            scenario pair: use doubled()\n        ok int got => ok got\n" + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": source})
	if err != nil {
		t.Fatal(err)
	}
	appID := scenarioPackageID(t, program, "app")
	_, links := scenarioRootLinks(t, program, "paired")
	if len(links) != 1 || links[0] != appID+"::pair" {
		t.Fatalf("use-site links = %v", links)
	}
	step := templateFixtureSteps(t, program, "first_use")
	if len(step.Fixtures.Rows) != 1 {
		t.Fatalf("expanded table has %d rows", len(step.Fixtures.Rows))
	}
	row := step.Fixtures.Rows[0]
	if row.Scenario != appID+"::pair" || row.Owner != appID || row.Selector != "pair" {
		t.Fatalf("expanded row lost its use-site identity: %+v", row)
	}
}

func TestScenarioMalformedSidecar(t *testing.T) {
	source := "package app\n    provides []\n    uses [http, codec]\nrecord receipt\n    int count\nconnection service\n    endpoint \"http://127.0.0.1:1/\"\n    timeout_ms 5000\nfetch receipt load_json from service\n    emits [http::request_failed]\n    asserts\n        decoded: => ok receipt(7)\n            using raw \"fixtures/load_json.json\"\n    get \"/json\"\nscenario flow\nfn receipt cached\n    emits [http::request_failed]\n    asserts\n        sample: => ok receipt(7) link flow\n    match call load_json()\n        when\n            scenario flow: => ok receipt(7)\n                using raw \"fixtures/broken.json\"\n        http::request_failed\n        ok receipt found => ok found\n" + programMain + "    ok\n"
	files := withNativeRaw(map[string]string{"src/main.can": source}, "load_json")
	files["src/fixtures/broken.json"] = "{oops"
	_, err := programFixture(t, files)
	if err == nil || !strings.Contains(err.Error(), "fixture flow") || !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("malformed scenario sidecar admitted: %v", err)
	}
}
