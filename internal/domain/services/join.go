package services

import (
	"encoding/json"
	"fmt"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// BuildEntries classifies rawTimeline and joins rawReviewComments (from
// GET /pulls/{number}/comments) to their parent review via
// pull_request_review_id, inserting each inline comment immediately after
// its parent review. A comment whose review id matches nothing fetched is
// appended at the end rather than dropped, so it isn't silently lost. A
// timeline item or review comment that cannot be classified is recorded as
// a SkipNote and skipped rather than aborting the whole call. issueURL is
// the issue/PR's own html_url (see classifyLabelEvent).
func BuildEntries(rawTimeline, rawReviewComments []json.RawMessage, issueURL string) ([]valueobjects.Entry, []SkipNote) {
	items, knownReviewIDs, skipped := classify(rawTimeline, issueURL)

	byReview := map[int64][]valueobjects.Entry{}
	var orphaned []valueobjects.Entry
	seenCommentIDs := make(map[int64]bool)

	for _, raw := range rawReviewComments {
		comment, id, reviewID, err := buildReviewComment(raw)
		if err != nil {
			skipped = append(skipped, SkipNote{Reason: err.Error(), Raw: raw})
			continue
		}
		// See markSeen: without this, a duplicate id would render the same
		// InlineReviewComment twice.
		if markSeen(seenCommentIDs, id) {
			skipped = append(skipped, SkipNote{
				Reason: fmt.Sprintf("duplicate review comment id %d", id),
				Raw:    raw,
			})
			continue
		}

		if knownReviewIDs[reviewID] {
			byReview[reviewID] = append(byReview[reviewID], comment)
		} else {
			orphaned = append(orphaned, comment)
		}
	}

	result := make([]valueobjects.Entry, 0, len(items)+len(rawReviewComments))
	for _, it := range items {
		switch {
		case it.direct != nil:
			result = append(result, it.direct)
		case it.review != nil:
			result = append(result, it.review.review)
			result = append(result, byReview[it.review.id]...)
		}
	}
	result = append(result, orphaned...)

	return result, skipped
}

func buildReviewComment(raw json.RawMessage) (valueobjects.InlineReviewComment, int64, int64, error) {
	var w reviewCommentWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return valueobjects.InlineReviewComment{}, 0, 0, fmt.Errorf("unmarshal review comment: %w", err)
	}

	attribution, err := valueobjects.NewAttribution(w.User.resolvedLogin(), w.CreatedAt, w.HTMLURL)
	if err != nil {
		return valueobjects.InlineReviewComment{}, 0, 0, fmt.Errorf("review comment attribution: %w", err)
	}
	line, startLine, outdated := w.resolvedLine()
	ctx, err := valueobjects.NewInlineContext(w.Path, line, startLine, w.DiffHunk, outdated)
	if err != nil {
		return valueobjects.InlineReviewComment{}, 0, 0, fmt.Errorf("review comment context: %w", err)
	}

	return valueobjects.NewInlineReviewComment(attribution, ctx, w.Body), w.ID, w.PullRequestReviewID, nil
}
