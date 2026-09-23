package emit

import (
	"encoding/json"
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// s3OperationBindings maps the B1-10 S3 operations to their state-module
// targets. Clients, uploads, presigned handles and continuations are
// opaque; metadata, entries, pages and presigned info cross as records.
func s3OperationBindings() bindingContribution {
	functions := map[string]string{
		"can.std.s3@1::client_open":   "$canS3.clientOpen",
		"can.std.s3@1::read_bytes":    "$canS3.readBytes",
		"can.std.s3@1::read_range":    "$canS3.readRange",
		"can.std.s3@1::read_stream":   "$canS3.readStream",
		"can.std.s3@1::write_bytes":   "$canS3.writeBytes",
		"can.std.s3@1::write_stream":  "$canS3.writeStream",
		"can.std.s3@1::stat":          "$canS3.stat",
		"can.std.s3@1::exists":        "$canS3.exists",
		"can.std.s3@1::delete":        "$canS3.remove",
		"can.std.s3@1::list":          "$canS3.list",
		"can.std.s3@1::presign":       "$canS3.presign",
		"can.std.s3@1::describe":      "$canS3.describe",
		"can.std.s3@1::begin_upload":  "$canS3.beginUpload",
		"can.std.s3@1::upload_write":  "$canS3.uploadWrite",
		"can.std.s3@1::upload_finish": "$canS3.uploadFinish",
		"can.std.s3@1::cancel_upload": "$canS3.cancelUpload",
	}
	return bindingContribution{domain: "s3", functions: functions}
}

// s3StateImports lists the S3 factory module the shared state module needs.
func (assembly *programAssembly) s3StateImports(runtime string) []ModuleImport {
	return []ModuleImport{
		{Target: runtime + "/platform/s3.ts", Names: []ImportName{{"createS3", "$canCreateS3"}, {"isS3Value", "$canIsS3"}}},
	}
}

// s3StateValueImportNames lists the S3 factory value authored and
// assertion modules import from the state module.
func s3StateValueImportNames() []ImportName {
	return []ImportName{{"$canS3", "$canS3"}}
}

// declareS3State emits the S3 factory binding.
func (builder *stateBuilder) declareS3State() {
	builder.out.WriteString("export let $canS3:ReturnType<typeof $canCreateS3>;\n")
}

// initializeS3State constructs the S3 factory inside the shared
// initializer, after the domain runtime exists.
func (builder *stateBuilder) initializeS3State() {
	fmt.Fprintf(&builder.out, "$canS3=$canCreateS3($canDomain,{invalid:%s,missing:%s,denied:%s,service:%s,closed:%s,overLimit:%s,metadata:%s,entry:%s,page:%s,info:%s,methodGet:%s,methodPut:%s,methodDelete:%s,methodHead:%s,some:%s,none:%s,readFailed:%s,cancelled:%s,closeFailed:%s});\n", quote(builder.numberIDs["can.std.s3@1::invalid_config"]), quote(builder.numberIDs["can.std.s3@1::missing_key"]), quote(builder.numberIDs["can.std.s3@1::access_denied"]), quote(builder.numberIDs["can.std.s3@1::service_error"]), quote(builder.numberIDs["can.std.s3@1::upload_closed"]), quote(builder.numberIDs["can.std.s3@1::over_limit"]), quote(builder.numberIDs["can.std.s3@1::metadata"]), quote(builder.numberIDs["can.std.s3@1::entry"]), quote(builder.numberIDs["can.std.s3@1::page"]), quote(builder.numberIDs["can.std.s3@1::presigned_info"]), quote(builder.numberIDs["can.std.s3@1::method_get"]), quote(builder.numberIDs["can.std.s3@1::method_put"]), quote(builder.numberIDs["can.std.s3@1::method_delete"]), quote(builder.numberIDs["can.std.s3@1::method_head"]), quote(builder.optionIDs["can.std.option@1::some"]), quote(builder.optionIDs["can.std.option@1::none"]), quote(builder.numberIDs["can.std.stream@1::read_failed"]), quote(builder.numberIDs["can.std.stream@1::cancelled"]), quote(builder.numberIDs["can.std.stream@1::close_failed"]))
}

// emitS3Kinds emits the opaque-handle kind table the domain predicate
// uses to recognize S3 values at boundaries.
func (builder *stateBuilder) emitS3Kinds() error {
	s3Kinds := map[string]string{}
	for _, typ := range builder.assembly.program.Model.Types() {
		if typ.Kind() != types.Opaque {
			continue
		}
		switch typ.Declaration() {
		case "can.std.s3@1::client":
			s3Kinds[typ.Identity()] = "client"
		case "can.std.s3@1::upload":
			s3Kinds[typ.Identity()] = "upload"
		case "can.std.s3@1::presigned":
			s3Kinds[typ.Identity()] = "presigned"
		case "can.std.s3@1::continuation":
			s3Kinds[typ.Identity()] = "continuation"
		}
	}
	s3KindsJSON, err := json.Marshal(s3Kinds)
	if err != nil {
		return err
	}
	fmt.Fprintf(&builder.out, "const $canS3Kinds:Readonly<Record<string,string>>=%s;\n", s3KindsJSON)
	return nil
}
