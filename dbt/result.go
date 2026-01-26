package dbt

// Artifacts holds the raw dbt artifact JSON strings collected from the target directory.
type Artifacts struct {
	Manifest     string
	RunResults   string
	Catalog      string
	Sources      string
	InvocationId string
}
