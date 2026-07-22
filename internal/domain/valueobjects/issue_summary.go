package valueobjects

import (
	"errors"
	"fmt"
	"strings"
)

// IssueSummary is a lightweight reference to an issue related to the one
// being exported — either its parent or one of its sub-issues, sourced from
// GET /issues/{number} (parent_issue_url) or GET
// /issues/{number}/sub_issues (each element) respectively. Unlike
// IssueRef, which identifies an issue by owner/repo/number for fetching, an
// IssueSummary is display data already resolved from a fetched resource.
type IssueSummary struct {
	number int
	title  string
	state  IssueState
	url    Url
}

// NewIssueSummary constructs an IssueSummary from its number, title, state,
// and url. It returns an error if number is not positive or url fails to
// parse as an absolute http/https URL.
func NewIssueSummary(number int, title string, state IssueState, url string) (IssueSummary, error) {
	if number <= 0 {
		return IssueSummary{}, errors.New("issue summary number must be positive")
	}
	parsedURL, err := NewUrl(url)
	if err != nil {
		return IssueSummary{}, err
	}
	return IssueSummary{number: number, title: title, state: state, url: parsedURL}, nil
}

// Number returns the issue's number.
func (s IssueSummary) Number() int {
	return s.number
}

// Title returns the issue's title.
func (s IssueSummary) Title() string {
	return s.title
}

// State returns whether the issue is open or closed.
func (s IssueSummary) State() IssueState {
	return s.state
}

// URL returns the issue's own html_url.
func (s IssueSummary) URL() Url {
	return s.url
}

// Equals reports whether s and other have the same number, title, state,
// and url.
func (s IssueSummary) Equals(other IssueSummary) bool {
	return s.number == other.number &&
		s.title == other.title &&
		s.state == other.state &&
		s.url.Equals(other.url)
}

// issueSummaryLine formats s as a single bullet-list line, used by
// SubIssues' Render. It reuses the title-first, backtick-wrapped,
// linked-number shape services.RewriteIssueReferences established for a
// bare issue/PR reference elsewhere in a rendered document, rather than
// leaving s's title as unlinked plain text.
func issueSummaryLine(s IssueSummary) string {
	return fmt.Sprintf("%s [#%d](%s) (%s)", titleCodeSpan(s.title), s.number, s.url, s.state.String())
}

// titleCodeSpan returns title wrapped in an inline code span, using a
// backtick fence one character longer than the longest run of backticks
// already inside title, with a padding space added when title starts or
// ends with a backtick so the fence's own delimiter does not merge with
// title's. This duplicates services.titleCodeSpan rather than sharing it:
// services already depends on valueobjects, so the reverse dependency
// this shape would need is unavailable, and this project prefers
// duplication over a premature cross-package abstraction for a handful of
// similar lines.
func titleCodeSpan(title string) string {
	fence := strings.Repeat("`", longestBacktickRun(title)+1)
	if strings.HasPrefix(title, "`") || strings.HasSuffix(title, "`") {
		return fence + " " + title + " " + fence
	}
	return fence + title + fence
}

// longestBacktickRun returns the length of the longest run of consecutive
// backtick characters in s.
func longestBacktickRun(s string) int {
	longest, current := 0, 0
	for _, r := range s {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	return longest
}
