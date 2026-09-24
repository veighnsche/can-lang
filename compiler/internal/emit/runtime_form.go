package emit

import (
	"sort"
	"strings"
)

// formOperationBindings maps the total keyed-row name operations to the
// form factory. The generic builders and the adapter binder specialize
// per concrete type through formSpecializationBindings.
func formOperationBindings() bindingContribution {
	return bindingContribution{domain: "form", functions: map[string]string{
		"can.std.form@1::row_key":    "$canForm.rowKey",
		"can.std.form@1::order_name": "$canForm.orderName",
		"can.std.form@1::input_name": "$canForm.inputName",
		"can.std.form@1::field_name": "$canForm.fieldName",
	}}
}

// formSpecializationBindings assigns the checked form specializations of
// this program to their factory targets. Builder contracts carry no
// per-type runtime data, so every specialization shares its operation
// target; the adapter splices its frozen contract per call site.
func (assembly *programAssembly) formSpecializationBindings() bindingContribution {
	functions := map[string]string{}
	ids := make([]string, 0, len(assembly.program.Forms))
	for id := range assembly.program.Forms {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		operation, _, _ := strings.Cut(id, "<")
		switch operation {
		case "can.std.form@1::named_collection":
			functions[id] = "$canForm.namedCollection"
		case "can.std.form@1::named_field":
			functions[id] = "$canForm.namedField"
		case "can.std.http@1::serve_form_action":
			functions[id] = "$canFormActions.serve"
		}
	}
	return bindingContribution{domain: "form-specializations", functions: functions}
}
