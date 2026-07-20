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
	if err := validateOwner(owner); err != nil {
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

func validateOwner(owner string) error {
	if owner == "" {
		return errors.New("issue ref owner must not be empty")
	}
	if len(owner) > maxOwnerLength {
		return fmt.Errorf("issue ref owner must be at most %d characters, got %d", maxOwnerLength, len(owner))
	}
	if !ownerPattern.MatchString(owner) {
		return fmt.Errorf("issue ref owner %q is not a valid GitHub username", owner)
	}
	return nil
}

func validateRepoName(repo string) error {
	if repo == "" {
		return errors.New("issue ref repo must not be empty")
	}
	if repo == "." || repo == ".." {
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
// on-disk location — to a downloaded attachment named filename, per
// docs/SPEC.md's {number}/assets/{filename} layout for referencing an
// issue's own downloaded attachments from its rendered Markdown.
func (r IssueRef) AssetPath(filename string) string {
	return fmt.Sprintf("%d/assets/%s", r.number, filename)
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
