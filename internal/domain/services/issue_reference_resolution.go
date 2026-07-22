package services

import "github.com/connect0459/gh-exhibit/internal/domain/valueobjects"

// ResolvedIssueReference is the outcome of attempting to resolve a single
// detected IssueReference's title: either the referenced issue/PR's title
// and url, or an unresolved marker (its target could not be fetched — e.g.
// deleted, made private, or a transient fetch failure). Success is tracked
// by its own field rather than inferred from title being empty, the same
// reason Resolution tracks succeeded explicitly rather than inferring it
// from reason (see Resolution's own Godoc): an issue/PR with an empty
// title is a real, if unusual, possibility.
type ResolvedIssueReference struct {
	reference IssueReference
	title     string
	url       valueobjects.Url
	resolved  bool
}

// Resolved builds a ResolvedIssueReference for a reference whose title and
// url were successfully fetched.
func Resolved(reference IssueReference, title string, url valueobjects.Url) ResolvedIssueReference {
	return ResolvedIssueReference{reference: reference, title: title, url: url, resolved: true}
}

// Unresolved builds a ResolvedIssueReference for a reference whose target
// could not be fetched. RewriteIssueReferences leaves reference's original
// text untouched for this case — unlike a failed attachment fetch, no
// placeholder is substituted; the reference was already valid, readable
// text before this feature existed, so a resolution failure only forgoes
// the readability improvement rather than needing to flag anything lost.
func Unresolved(reference IssueReference) ResolvedIssueReference {
	return ResolvedIssueReference{reference: reference}
}
