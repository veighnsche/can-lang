package emit

import "testing"

func TestImageInspectionBindingTargetsNativeAdapter(t *testing.T) {
	const identity = "can.std.image@1::inspect"
	if got := coreOperationBindings().functions[identity]; got != "$canImage.inspect" {
		t.Fatalf("image inspection binding = %q, want %q", got, "$canImage.inspect")
	}
}
