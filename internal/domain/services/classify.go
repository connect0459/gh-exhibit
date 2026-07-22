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

// markSeen registers id in seen and reports whether it was already present,
// guarding a timeline event or review comment against being counted twice
// when overlapping pagination delivers it on more than one page. id<=0 is
// never treated as a duplicate: it marks a missing or malformed id rather
// than a genuine repeat, so registering it would let an unrelated event
// with the same defaulted id be wrongly flagged as a duplicate.
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
// issueURL is the issue/PR's own html_url, used as a labeled/unlabeled
// event's attribution url since GitHub gives that event kind no per-event
// permalink of its own (see classifyLabelEvent).
func classify(rawTimeline []json.RawMessage, issueURL string) ([]classifiedItem, map[int64]bool, []SkipNote) {
	items := make([]classifiedItem, 0, len(rawTimeline))
	var skipped []SkipNote
	seenReviewIDs := make(map[int64]bool)
	seenCommentedIDs := make(map[int64]bool)
	seenLabelIDs := make(map[int64]bool)

	for _, raw := range rawTimeline {
		var d discriminator
		if err := json.Unmarshal(raw, &d); err != nil {
			skipped = append(skipped, SkipNote{
				Reason: fmt.Sprintf("peek discriminator: %v", err),
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
			// See markSeen: without this, a duplicate id would render
			// the same IssueComment twice.
			if markSeen(seenCommentedIDs, id) {
				skipped = append(skipped, SkipNote{
					Reason: fmt.Sprintf("duplicate commented event id %d", id),
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
			// See markSeen: without this, a duplicate id would duplicate
			// both the review and every inline comment bucketed under it.
			if id := item.review.id; markSeen(seenReviewIDs, id) {
				skipped = append(skipped, SkipNote{
					Reason: fmt.Sprintf("duplicate reviewed event id %d", id),
					Raw:    raw,
				})
				continue
			}
			items = append(items, item)

		case eventKindLabeled, eventKindUnlabeled:
			item, id, err := classifyLabelEvent(raw, d.Event, issueURL)
			if err != nil {
				skipped = append(skipped, SkipNote{Reason: err.Error(), Raw: raw})
				continue
			}
			// See markSeen: without this, a duplicate id would render the
			// same LabelEvent twice.
			if markSeen(seenLabelIDs, id) {
				skipped = append(skipped, SkipNote{
					Reason: fmt.Sprintf("duplicate %s event id %d", d.Event, id),
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
		return classifiedItem{}, 0, fmt.Errorf("unmarshal commented event: %w", err)
	}

	attribution, err := valueobjects.NewAttribution(w.User.resolvedLogin(), w.CreatedAt, w.HTMLURL)
	if err != nil {
		return classifiedItem{}, 0, fmt.Errorf("commented event attribution: %w", err)
	}

	return classifiedItem{direct: valueobjects.NewIssueComment(attribution, w.Body)}, w.ID, nil
}

// classifyLabelEvent classifies a "labeled"/"unlabeled" timeline event into
// a LabelEvent, attributed to issueURL (the issue/PR's own html_url) since
// GitHub's payload for this event kind carries no per-event permalink of
// its own, unlike a "commented" or "reviewed" event's html_url.
func classifyLabelEvent(raw json.RawMessage, rawEvent string, issueURL string) (classifiedItem, int64, error) {
	var w labelEventWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return classifiedItem{}, 0, fmt.Errorf("unmarshal %s event: %w", rawEvent, err)
	}

	action, err := valueobjects.ParseLabelAction(rawEvent)
	if err != nil {
		return classifiedItem{}, 0, fmt.Errorf("%s event action: %w", rawEvent, err)
	}

	attribution, err := valueobjects.NewAttribution(w.Actor.resolvedLogin(), w.CreatedAt, issueURL)
	if err != nil {
		return classifiedItem{}, 0, fmt.Errorf("%s event attribution: %w", rawEvent, err)
	}

	event, err := valueobjects.NewLabelEvent(attribution, action, w.Label.Name, w.Label.Color)
	if err != nil {
		return classifiedItem{}, 0, fmt.Errorf("%s event: %w", rawEvent, err)
	}

	return classifiedItem{direct: event}, w.ID, nil
}

func classifyReviewedEvent(raw json.RawMessage) (classifiedItem, error) {
	var w reviewedEventWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return classifiedItem{}, fmt.Errorf("unmarshal reviewed event: %w", err)
	}

	attribution, err := valueobjects.NewAttribution(w.User.resolvedLogin(), w.SubmittedAt, w.HTMLURL)
	if err != nil {
		return classifiedItem{}, fmt.Errorf("reviewed event attribution: %w", err)
	}

	state, err := valueobjects.ParseReviewState(w.State)
	if err != nil {
		return classifiedItem{}, fmt.Errorf("reviewed event state: %w", err)
	}

	review := valueobjects.NewPullRequestReview(attribution, state, w.Body)
	return classifiedItem{review: &reviewCandidate{id: w.ID, review: review}}, nil
}
