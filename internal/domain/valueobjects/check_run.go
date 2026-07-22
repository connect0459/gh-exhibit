package valueobjects

import "errors"

// CheckRun is one check run associated with a commit, sourced from GET
// /repos/{owner}/{repo}/commits/{sha}/check-runs.
type CheckRun struct {
	name    string
	outcome CheckOutcome
	url     Url
}

// NewCheckRun constructs a CheckRun from its name, outcome, and rawURL (the
// run's own html_url). It returns an error if name is empty or rawURL is
// not a well-formed absolute http(s) URL.
func NewCheckRun(name string, outcome CheckOutcome, rawURL string) (CheckRun, error) {
	if name == "" {
		return CheckRun{}, errors.New("check run name must not be empty")
	}
	url, err := NewUrl(rawURL)
	if err != nil {
		return CheckRun{}, err
	}
	return CheckRun{name: name, outcome: outcome, url: url}, nil
}

// Name returns the check run's name (e.g. "build").
func (r CheckRun) Name() string {
	return r.name
}

// Outcome returns the check run's outcome.
func (r CheckRun) Outcome() CheckOutcome {
	return r.outcome
}

// URL returns the check run's own html_url.
func (r CheckRun) URL() Url {
	return r.url
}

// Equals reports whether r and other have the same name, outcome, and url.
func (r CheckRun) Equals(other CheckRun) bool {
	return r.name == other.name &&
		r.outcome == other.outcome &&
		r.url.Equals(other.url)
}
