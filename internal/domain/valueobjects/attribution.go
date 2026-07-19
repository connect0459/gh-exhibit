package valueobjects

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Attribution is the (author, created, url) triple shared by every Tier 1
// entry, corresponding to the meta:{...} line's common fields.
type Attribution struct {
	author  string
	created time.Time
	url     string
}

func NewAttribution(author string, created time.Time, url string) (Attribution, error) {
	if author == "" {
		return Attribution{}, errors.New("entry: attribution author must not be empty")
	}
	if !isASCII(author) {
		return Attribution{}, fmt.Errorf("entry: attribution author %q must contain only ASCII characters", author)
	}
	if url == "" {
		return Attribution{}, errors.New("entry: attribution url must not be empty")
	}
	return Attribution{author: author, created: created, url: url}, nil
}

// isASCII rejects non-ASCII letters that strings.EqualFold's Unicode case
// folding would otherwise conflate with an ASCII letter (e.g. U+212A KELVIN
// SIGN folds to "k"), which could make Attribution.Equals treat two
// distinct real GitHub logins as the same author.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func (a Attribution) Author() string {
	return a.author
}

func (a Attribution) CreatedAt() time.Time {
	return a.created
}

func (a Attribution) URL() string {
	return a.url
}

func (a Attribution) Equals(other Attribution) bool {
	return strings.EqualFold(a.author, other.author) &&
		a.url == other.url &&
		a.created.Equal(other.created)
}
