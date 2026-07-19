package cli

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestResolveRepo_ParsesAnExplicitRepoFlagWithoutCallingCurrent(t *testing.T) {
	called := false
	current := func() (repository.Repository, error) {
		called = true
		return repository.Repository{}, nil
	}

	got, err := ResolveRepo("octocat/hello-world", current)
	if err != nil {
		t.Fatalf("ResolveRepo() error = %v", err)
	}
	if called {
		t.Error("current() was called even though an explicit repo flag was given")
	}
	if got.Owner != "octocat" || got.Name != "hello-world" {
		t.Errorf("got Owner=%q Name=%q, want Owner=%q Name=%q", got.Owner, got.Name, "octocat", "hello-world")
	}
}

func TestResolveRepo_RejectsAMalformedRepoFlag(t *testing.T) {
	current := func() (repository.Repository, error) {
		t.Fatal("current() was called even though the repo flag was malformed")
		return repository.Repository{}, nil
	}

	if _, err := ResolveRepo("not-a-valid-repo", current); err == nil {
		t.Fatal("ResolveRepo() error = nil, want an error for a malformed --repo value")
	}
}

func TestResolveRepo_DelegatesToCurrentWhenNoRepoFlagIsGiven(t *testing.T) {
	want := repository.Repository{Host: "github.com", Owner: "octocat", Name: "hello-world"}
	current := func() (repository.Repository, error) {
		return want, nil
	}

	got, err := ResolveRepo("", current)
	if err != nil {
		t.Fatalf("ResolveRepo() error = %v", err)
	}
	if got != want {
		t.Errorf("ResolveRepo() = %+v, want %+v", got, want)
	}
}

func TestResolveRepo_PropagatesCurrentsError(t *testing.T) {
	wantErr := errors.New("no git remotes configured")
	current := func() (repository.Repository, error) {
		return repository.Repository{}, wantErr
	}

	_, err := ResolveRepo("", current)
	if !errors.Is(err, wantErr) {
		t.Errorf("ResolveRepo() error = %v, want it to wrap %v", err, wantErr)
	}
}
