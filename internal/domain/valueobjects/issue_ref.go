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

func (r IssueRef) Owner() string {
	return r.owner
}

func (r IssueRef) Repo() string {
	return r.repo
}

func (r IssueRef) Number() int {
	return r.number
}

func (r IssueRef) Equals(other IssueRef) bool {
	return strings.EqualFold(r.owner, other.owner) &&
		strings.EqualFold(r.repo, other.repo) &&
		r.number == other.number
}
