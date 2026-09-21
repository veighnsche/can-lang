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
	Mappings      []Mapping
}

// Mapping is a checked source span at a generated UTF16 coordinate.
type Mapping struct {
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Source    string `json:"source"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	Operation string `json:"operation"`
}
