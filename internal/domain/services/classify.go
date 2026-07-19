package services

import (
	"encoding/json"
	"fmt"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type reviewCandidate struct {
	id     int64
	review valueobjects.PullRequestReview
}

type classifiedItem struct {
	direct valueobjects.Entry
	review *reviewCandidate
}

// markSeen registers id in seen and reports whether it was already present.
// id<=0 is never treated as a duplicate: it marks a missing or malformed id
// rather than a genuine repeat, so registering it would let an unrelated
// event with the same defaulted id be wrongly flagged as a duplicate.
func markSeen(seen map[int64]bool, id int64) (duplicate bool) {
	if id <= 0 {
		return false
	}
	if seen[id] {
		return true
	}
	seen[id] = true
	return false
}

// classify skips (rather than aborts on) an individual timeline item that
// cannot be classified, recording a SkipNote for it, so one malformed or
// unexpected item cannot take down classification of the rest. It also
// returns the set of review ids it accepted, so BuildEntries can join
// review comments against it without re-deriving the same set from items.
func classify(rawTimeline []json.RawMessage) ([]classifiedItem, map[int64]bool, []SkipNote) {
	items := make([]classifiedItem, 0, len(rawTimeline))
	var skipped []SkipNote
	seenReviewIDs := make(map[int64]bool)
	seenCommentedIDs := make(map[int64]bool)

	for _, raw := range rawTimeline {
		var d discriminator
		if err := json.Unmarshal(raw, &d); err != nil {
			skipped = append(skipped, SkipNote{
				Reason: fmt.Sprintf("timeline: peek discriminator: %v", err),
				Raw:    raw,
			})
			continue
		}

		switch eventKind(d.Event) {
		case eventKindCommented:
			item, id, err := classifyCommentedEvent(raw)
			if err != nil {
				skipped = append(skipped, SkipNote{Reason: err.Error(), Raw: raw})
				continue
			}
			// A comment id repeated across the timeline array (e.g.
			// overlapping pagination delivering the same event twice)
			// would otherwise render the same IssueComment twice.
			if markSeen(seenCommentedIDs, id) {
				skipped = append(skipped, SkipNote{
					Reason: fmt.Sprintf("timeline: duplicate commented event id %d", id),
					Raw:    raw,
				})
				continue
			}
			items = append(items, item)

		case eventKindReviewed:
			item, err := classifyReviewedEvent(raw)
			if err != nil {
				skipped = append(skipped, SkipNote{Reason: err.Error(), Raw: raw})
				continue
			}
			// A review id repeated across the timeline array (e.g.
			// overlapping pagination delivering the same event twice)
			// would otherwise duplicate both the review and every inline
			// comment bucketed under its id in the final render.
			if id := item.review.id; markSeen(seenReviewIDs, id) {
				skipped = append(skipped, SkipNote{
					Reason: fmt.Sprintf("timeline: duplicate reviewed event id %d", id),
					Raw:    raw,
				})
				continue
			}
			items = append(items, item)

		default:
		}
	}

	return items, seenReviewIDs, skipped
}

func classifyCommentedEvent(raw json.RawMessage) (classifiedItem, int64, error) {
	var w commentedEventWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return classifiedItem{}, 0, fmt.Errorf("timeline: unmarshal commented event: %w", err)
	}

	attribution, err := valueobjects.NewAttribution(w.User.resolvedLogin(), w.CreatedAt, w.HTMLURL)
	if err != nil {
		return classifiedItem{}, 0, fmt.Errorf("timeline: commented event attribution: %w", err)
	}

	return classifiedItem{direct: valueobjects.NewIssueComment(attribution, w.Body)}, w.ID, nil
}

func classifyReviewedEvent(raw json.RawMessage) (classifiedItem, error) {
	var w reviewedEventWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return classifiedItem{}, fmt.Errorf("timeline: unmarshal reviewed event: %w", err)
	}

	attribution, err := valueobjects.NewAttribution(w.User.resolvedLogin(), w.SubmittedAt, w.HTMLURL)
	if err != nil {
		return classifiedItem{}, fmt.Errorf("timeline: reviewed event attribution: %w", err)
	}

	state, err := valueobjects.ParseReviewState(w.State)
	if err != nil {
		return classifiedItem{}, fmt.Errorf("timeline: reviewed event state: %w", err)
	}

	review := valueobjects.NewPullRequestReview(attribution, state, w.Body)
	return classifiedItem{review: &reviewCandidate{id: w.ID, review: review}}, nil
}
