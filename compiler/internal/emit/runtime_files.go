package emit

import "fmt"

// fileOperationBindings maps the B1-01 filesystem/path operations to their
// state-module targets.
func fileOperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.files@1::read_bytes":  "$canFileReads.readBytes",
		"can.std.files@1::read_text":   "$canFileReads.readText",
		"can.std.files@1::write_bytes": "$canFileWrites.writeBytes",
		"can.std.files@1::write_text":  "$canFileWrites.writeText",
		"can.std.files@1::copy":        "$canFileWrites.copy",
		"can.std.files@1::move":        "$canFileWrites.move",
		"can.std.files@1::stat":        "$canFileDirectory.stat",
		"can.std.files@1::exists":      "$canFileDirectory.exists",
		"can.std.files@1::list":        "$canFileDirectory.list",
		"can.std.files@1::mkdir":       "$canFileDirectory.mkdir",
		"can.std.files@1::remove":      "$canFileDirectory.remove",
		"can.std.files@1::glob":        "$canFileGlobs.glob",
		"can.std.path@1::resolve":      "$canPath.resolvePath",
		"can.std.path@1::join":         "$canPath.joinPath",
		"can.std.path@1::basename":     "$canPath.basename",
		"can.std.path@1::extension":    "$canPath.extension",
	}
	return bindingContribution{domain: "files", functions: functions}
}

// fileStateImports lists the filesystem factory modules the shared state
// module needs.
func (assembly *programAssembly) fileStateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/files/read.ts", Names: []ImportName{{"createFileReads", "$canCreateFileReads"}}},
		{Target: runtime + "/platform/files/write.ts", Names: []ImportName{{"createFileWrites", "$canCreateFileWrites"}}},
		{Target: runtime + "/platform/files/directory.ts", Names: []ImportName{{"createFileDirectory", "$canCreateFileDirectory"}}},
		{Target: runtime + "/platform/files/glob.ts", Names: []ImportName{{"createFileGlobs", "$canCreateFileGlobs"}}},
		{Target: runtime + "/platform/files/path.ts", Names: []ImportName{{"createPath", "$canCreatePath"}}},
	}
}

// fileStateValueImportNames lists the filesystem factory values authored and
// assertion modules import from the state module.
func fileStateValueImportNames() []ImportName {
	return []ImportName{{"$canFileReads", "$canFileReads"}, {"$canFileWrites", "$canFileWrites"}, {"$canFileDirectory", "$canFileDirectory"}, {"$canFileGlobs", "$canFileGlobs"}, {"$canPath", "$canPath"}}
}

// declareFileState emits the filesystem factory bindings.
func (builder *stateBuilder) declareFileState() {
	builder.out.WriteString("export let $canFileReads:ReturnType<typeof $canCreateFileReads>;\nexport let $canFileWrites:ReturnType<typeof $canCreateFileWrites>;\nexport let $canFileDirectory:ReturnType<typeof $canCreateFileDirectory>;\nexport let $canFileGlobs:ReturnType<typeof $canCreateFileGlobs>;\nexport let $canPath:ReturnType<typeof $canCreatePath>;\n")
}

// initializeFileState constructs the filesystem factories inside the shared
// initializer, after the domain runtime exists.
func (builder *stateBuilder) initializeFileState() {
	fmt.Fprintf(&builder.out, "$canFileReads=$canCreateFileReads($canDomain,{notFound:%s,denied:%s,invalidPath:%s,unexpectedKind:%s,limitExceeded:%s,invalidData:%s,ioError:%s});\n", quote(builder.numberIDs["can.std.files@1::not_found"]), quote(builder.numberIDs["can.std.files@1::denied"]), quote(builder.numberIDs["can.std.files@1::invalid_path"]), quote(builder.numberIDs["can.std.files@1::unexpected_kind"]), quote(builder.numberIDs["can.std.files@1::limit_exceeded"]), quote(builder.numberIDs["can.std.codec@1::invalid_data"]), quote(builder.numberIDs["can.std.files@1::io_error"]))
	fmt.Fprintf(&builder.out, "$canFileWrites=$canCreateFileWrites($canDomain,{notFound:%s,alreadyExists:%s,denied:%s,invalidPath:%s,unexpectedKind:%s,notEmpty:%s,crossDevice:%s,ioError:%s});\n", quote(builder.numberIDs["can.std.files@1::not_found"]), quote(builder.numberIDs["can.std.files@1::already_exists"]), quote(builder.numberIDs["can.std.files@1::denied"]), quote(builder.numberIDs["can.std.files@1::invalid_path"]), quote(builder.numberIDs["can.std.files@1::unexpected_kind"]), quote(builder.numberIDs["can.std.files@1::not_empty"]), quote(builder.numberIDs["can.std.files@1::cross_device"]), quote(builder.numberIDs["can.std.files@1::io_error"]))
	fmt.Fprintf(&builder.out, "$canFileDirectory=$canCreateFileDirectory($canDomain,{notFound:%s,alreadyExists:%s,denied:%s,invalidPath:%s,unexpectedKind:%s,limitExceeded:%s,notEmpty:%s,ioError:%s,fileInfo:%s,entry:%s});\n", quote(builder.numberIDs["can.std.files@1::not_found"]), quote(builder.numberIDs["can.std.files@1::already_exists"]), quote(builder.numberIDs["can.std.files@1::denied"]), quote(builder.numberIDs["can.std.files@1::invalid_path"]), quote(builder.numberIDs["can.std.files@1::unexpected_kind"]), quote(builder.numberIDs["can.std.files@1::limit_exceeded"]), quote(builder.numberIDs["can.std.files@1::not_empty"]), quote(builder.numberIDs["can.std.files@1::io_error"]), quote(builder.numberIDs["can.std.files@1::file_info"]), quote(builder.numberIDs["can.std.files@1::entry"]))
	fmt.Fprintf(&builder.out, "$canFileGlobs=$canCreateFileGlobs($canDomain,{notFound:%s,denied:%s,invalidPath:%s,limitExceeded:%s,ioError:%s});\n", quote(builder.numberIDs["can.std.files@1::not_found"]), quote(builder.numberIDs["can.std.files@1::denied"]), quote(builder.numberIDs["can.std.files@1::invalid_path"]), quote(builder.numberIDs["can.std.files@1::limit_exceeded"]), quote(builder.numberIDs["can.std.files@1::io_error"]))
	builder.out.WriteString("$canPath=$canCreatePath();\n")
}
