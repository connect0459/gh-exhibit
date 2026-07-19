package cli

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// ResolveRepo determines the target repository: flagRepo (an explicit
// --repo owner/repo value) when non-empty, otherwise the result of current
// (typically repository.Current, injected here so tests don't depend on
// real git remotes).
func ResolveRepo(flagRepo string, current func() (repository.Repository, error)) (repository.Repository, error) {
	if flagRepo != "" {
		repo, err := repository.Parse(flagRepo)
		if err != nil {
			return repository.Repository{}, fmt.Errorf("cli: parse --repo %q: %w", flagRepo, err)
		}
		return repo, nil
	}

	repo, err := current()
	if err != nil {
		return repository.Repository{}, fmt.Errorf("cli: could not determine the repository from the current directory: %w", err)
	}
	return repo, nil
}
