package valueobjects

// equalPointers reports whether two optional values are equal: both nil, or
// both non-nil and equal under eq. Equality is delegated to eq rather than
// == on the dereferenced values so a type needing special-case comparison
// (e.g. time.Time's monotonic-reading pitfall) isn't tempted to skip it.
func equalPointers[T any](a, b *T, eq func(T, T) bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return eq(*a, *b)
}
