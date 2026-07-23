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

// copyPointer returns a fresh pointer to a copy of *p, or nil when p is
// nil — a defensive copy for an optional field pointer, mirroring this
// package's own append([]T(nil), s...) copy for slice-typed fields. Used
// both when a constructor stores a caller-supplied pointer and when an
// accessor returns one, so neither direction lets a caller alias a value
// object's internal state through a shared *T (including via a mutating
// pointer-receiver method the pointed-to type happens to expose, e.g.
// time.Time's UnmarshalJSON).
func copyPointer[T any](p *T) *T {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}
