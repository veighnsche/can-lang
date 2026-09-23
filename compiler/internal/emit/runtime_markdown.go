package emit

import (
	"fmt"
)

// markdownOperationBindings maps the B1-12 Markdown operations to their
// state-module targets. String rendering stays an ordinary str; only the
// safe renderer mints html::safe through the HTML factory.
func markdownOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.markdown@1::render_text_html": "$canMarkdown.renderTextHTML",
		"can.std.markdown@1::render_safe":      "$canMarkdown.renderSafe",
	}
	return bindingContribution{domain: "markdown", functions: functions}
}

// markdownStateImports lists the Markdown factory module the shared state
// module needs.
func (assembly *programAssembly) markdownStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/markdown.ts", Names: []ImportName{{"createMarkdown", "$canCreateMarkdown"}}},
	}
}

// markdownStateValueImportNames lists the Markdown factory value authored
// and assertion modules import from the state module.
func markdownStateValueImportNames() []ImportName {
	return []ImportName{{"$canMarkdown", "$canMarkdown"}}
}

// declareMarkdownState emits the Markdown factory binding.
func (builder *stateBuilder) declareMarkdownState() {
	builder.out.WriteString("export let $canMarkdown:ReturnType<typeof $canCreateMarkdown>;\n")
}

// initializeMarkdownState constructs the Markdown factory inside the
// shared initializer, after the domain runtime exists. The factory builds
// its own HTML instance from the shared contracts, so safe assembly needs
// no cross-factory ordering.
func (builder *stateBuilder) initializeMarkdownState() {
	fmt.Fprintf(&builder.out, "$canMarkdown=$canCreateMarkdown($canDomain,{overLimit:%s,htmlStructure:%s,htmlURL:%s});\n", quote(builder.numberIDs["can.std.markdown@1::over_limit"]), quote(builder.numberIDs["can.std.html@1::invalid_structure"]), quote(builder.numberIDs["can.std.html@1::invalid_url"]))
}
