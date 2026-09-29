package check

import "testing"

func TestImageInspectIsCallableWithTypedMetadata(t *testing.T) {
	source := `package app
    provides []
    uses [bytes, codec, image]
fn image::metadata inspect_image
    emits [codec::invalid_data, image::invalid_image]
    asserts
        sample: => ok image::metadata("png", 1, 1)
    match chain
        call bytes::from_ints([0, 1, 2]) as bytes::buffer source
        call image::inspect(source, 100) as image::metadata metadata
        codec::invalid_data
        image::invalid_image
        ok => ok metadata
fn void main
    emits []
    given
        str[] args
    asserts
        empty: [] => ok
    ok
`
	if _, err := programFixture(t, map[string]string{"src/main.can": source}); err != nil {
		t.Fatal(err)
	}
}
