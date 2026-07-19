// Package valueobjects models the Tier 1 render set (ADR-001, ADR-002): the
// issue/PR body, issue comments, PR reviews, and inline review comments
// that gh-exhibit renders into Markdown.
package valueobjects

import "io"

// Entry is the sealed set of Tier 1 render targets. The unexported
// entryNode method restricts implementers to this package, the closest Go
// analogue to a closed sum type (mirroring go/ast's Expr.exprNode pattern).
type Entry interface {
	Render(w io.Writer) error

	entryNode()
}
