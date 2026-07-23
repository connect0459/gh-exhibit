package valueobjects

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	maxOwnerLength = 39
	maxRepoLength  = 100
)

var ownerPattern = regexp.MustCompile(`^[A-Za-z0-9](-?[A-Za-z0-9])*$`)

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// IssueRef identifies which issue or pull request to fetch evidence for.
// Named for GitHub's own REST convention of addressing pull requests via
// /issues/{number} for the shared resource.
type IssueRef struct {
	owner  string
	repo   string
	number int
}

// NewIssueRef constructs an IssueRef from owner, repo, and number. It
// returns an error if owner or repo fails GitHub's own username/
// repository-name rules (length, character set) or if number is not
// positive.
func NewIssueRef(owner, repo string, number int) (IssueRef, error) {
	if err := validateOwner(owner, "issue ref owner"); err != nil {
		return IssueRef{}, err
	}
	if err := validateRepoName(repo); err != nil {
		return IssueRef{}, err
	}
	if number <= 0 {
		return IssueRef{}, errors.New("issue ref number must be positive")
	}
	return IssueRef{owner: owner, repo: repo, number: number}, nil
}

// validateOwner validates owner against GitHub's own username rules. label
// names the calling type's own field in any returned error (e.g. "issue ref
// owner", "search criteria author") — every caller names its own field
// rather than sharing IssueRef's wording, since validateOwner's error is
// part of each caller's own public contract, not IssueRef's alone.
func validateOwner(owner, label string) error {
	if owner == "" {
		return fmt.Errorf("%s must not be empty", label)
	}
	if len(owner) > maxOwnerLength {
		return fmt.Errorf("%s must be at most %d characters, got %d", label, maxOwnerLength, len(owner))
	}
	if !ownerPattern.MatchString(owner) {
		return fmt.Errorf("%s %q is not a valid GitHub username", label, owner)
	}
	return nil
}

func validateRepoName(repo string) error {
	if repo == "" {
		return errors.New("issue ref repo must not be empty")
	}
	if isAllDotsWithOptionalTrailingSpaces(repo) {
		return fmt.Errorf("issue ref repo must not be %q", repo)
	}
	if len(repo) > maxRepoLength {
		return fmt.Errorf("issue ref repo must be at most %d characters, got %d", maxRepoLength, len(repo))
	}
	if !repoPattern.MatchString(repo) {
		return fmt.Errorf("issue ref repo %q is not a valid GitHub repository name", repo)
	}
	return nil
}

// Owner returns the repository's owner (user or organization login).
func (r IssueRef) Owner() string {
	return r.owner
}

// Repo returns the repository name.
func (r IssueRef) Repo() string {
	return r.repo
}

// Number returns the issue or pull request number.
func (r IssueRef) Number() int {
	return r.number
}

// AssetPath returns the relative path — from the rendered document's own
// on-disk location — to a downloaded attachment named filename
// (assets/{filename}), for referencing an issue's own downloaded
// attachments from its rendered Markdown. Both the document and its assets
// live under the same {number}/ directory, so no number prefix is needed.
func (r IssueRef) AssetPath(filename AssetFilename) string {
	return fmt.Sprintf("assets/%s", filename.String())
}

// Equals reports whether r and other identify the same owner, repo, and
// number. Owner and repo are compared case-insensitively
// (strings.EqualFold), matching GitHub's own case-insensitive uniqueness
// rule for both.
func (r IssueRef) Equals(other IssueRef) bool {
	return strings.EqualFold(r.owner, other.owner) &&
		strings.EqualFold(r.repo, other.repo) &&
		r.number == other.number
}
