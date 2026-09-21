package ir

// Artifact is an emitted file and its complete static module inventory.
type Artifact struct {
	Path  string
	Bytes []byte
	// Imports are complete structured emission edges, including import type.
	// Only verified distribution runtime artifacts may carry NativeImports.
	Imports       []string
	NativeImports []string
	Runtime       bool
}
