package ir

// AssetResolution is the compile-time outcome of one asset::url call.
// A URL is present only for a name owned by the calling project. Missing and
// unowned names keep the html::url failure reason and never select a path.
type AssetResolution struct {
	URL    string
	Reason string
}
