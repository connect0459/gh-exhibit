package services

import (
	"regexp"
	"strconv"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// IssueReference/DetectIssueReferences/ResolvedIssueReference/
// RewriteIssueReferences below implement resolving a bare (not already
// formatted as a link) "#123" or "owner/repo#123" issue/PR reference inside
// already-rendered Markdown into a link carrying the referenced issue/PR's
// own title, independent of Attachment/Detect/Resolution/Rewrite's own
// user-attachments-URL half of this package. Detection and rewriting run as
// a post-render pass over a Document's full output, the same shape
// Attachment/Detect/Rewrite already use, so no Tier 1 type needs a
// content-mutation path of its own for this either.

// IssueReference identifies a single bare or cross-repository issue/PR
// reference detected in already-rendered Markdown, at the byte range it
// occupies there.
type IssueReference struct {
	ref   valueobjects.IssueRef
	start int
	end   int
}

// Ref returns the issue or pull request this reference points to.
func (r IssueReference) Ref() valueobjects.IssueRef {
	return r.ref
}

// issueReferencePattern matches either a cross-repository reference
// ("owner/repo#123") or a bare same-repository reference ("#123"). \b/\B
// (word-boundary/non-word-boundary) assertions are used in place of
// lookaround, which Go's RE2-based regexp engine does not support: \B
// before "#" requires the character immediately before it (if any) to not
// be a word character, rejecting "build#42" as a glued-together token
// while still accepting "#42" at the very start of the text; \b after the
// digits rejects "#42abc"; \b before "owner" rejects a cross-repo
// reference glued to a preceding word character (e.g. "xowner/repo#42").
var issueReferencePattern = regexp.MustCompile(
	`\b(?P<owner>[A-Za-z0-9](?:-?[A-Za-z0-9])*)/(?P<repo>[A-Za-z0-9._-]+)#(?P<crossnum>[0-9]+)\b` +
		`|` +
		`\B#(?P<barenum>[0-9]+)\b`,
)

var (
	ownerGroup    = issueReferencePattern.SubexpIndex("owner")
	repoGroup     = issueReferencePattern.SubexpIndex("repo")
	crossNumGroup = issueReferencePattern.SubexpIndex("crossnum")
	bareNumGroup  = issueReferencePattern.SubexpIndex("barenum")
)

// DetectIssueReferences returns every bare/cross-repository issue or pull
// request reference in markdown, in first-to-last scan order (not
// deduplicated — every occurrence gets its own entry, since each carries
// its own byte range for RewriteIssueReferences to substitute). current is
// the ref markdown was rendered for, supplying the owner/repo a bare
// "#123" reference (no owner/repo prefix of its own) resolves against.
// Excluded: a reference inside a fenced code block or inline code span
// (this project's own already-established backtick-wrapped untrusted-text
// convention — see checkRunLine/changedFileLine/commitLine/
// issueSummaryLine — plus a diff patch or commit message's verbatim
// content), inside an HTML comment (a Tier 1 entry's own meta line), or
// already formatted as a markdown link. A reference whose owner/repo/
// number fails valueobjects.NewIssueRef's own validation (e.g. "#0") is
// silently skipped rather than treated as a malformed reference, matching
// this package's skip-and-continue handling elsewhere.
func DetectIssueReferences(markdown []byte, current valueobjects.IssueRef) []IssueReference {
	protected := protectedRanges(markdown)

	matches := issueReferencePattern.FindAllSubmatchIndex(markdown, -1)
	references := make([]IssueReference, 0, len(matches))
	for _, m := range matches {
		start, end := m[0], m[1]
		if overlapsAny(start, end, protected) {
			continue
		}

		var owner, repo string
		var numberRaw []byte
		if m[2*crossNumGroup] != -1 {
			owner = string(markdown[m[2*ownerGroup]:m[2*ownerGroup+1]])
			repo = string(markdown[m[2*repoGroup]:m[2*repoGroup+1]])
			numberRaw = markdown[m[2*crossNumGroup]:m[2*crossNumGroup+1]]
		} else {
			owner = current.Owner()
			repo = current.Repo()
			numberRaw = markdown[m[2*bareNumGroup]:m[2*bareNumGroup+1]]
		}

		number, err := strconv.Atoi(string(numberRaw))
		if err != nil {
			continue
		}
		ref, err := valueobjects.NewIssueRef(owner, repo, number)
		if err != nil {
			continue
		}

		references = append(references, IssueReference{ref: ref, start: start, end: end})
	}
	return references
}
