// Package services holds gh-exhibit's domain-layer transformation logic:
// classifying GitHub's heterogeneous REST timeline array into concrete
// valueobjects.Entry values and joining inline review comments to their
// parent review, plus detecting and rewriting GitHub `user-attachments`
// URLs inside already-rendered Markdown.
package services

type discriminator struct {
	Event string `json:"event"`
}

// eventKind lists only the event kinds classified into Tier 1 entries, not
// GitHub's full event space, so the exhaustive lint check only guards
// against forgetting a kind added here later.
type eventKind string

const (
	eventKindCommented    eventKind = "commented"
	eventKindReviewed     eventKind = "reviewed"
	eventKindLabeled      eventKind = "labeled"
	eventKindUnlabeled    eventKind = "unlabeled"
	eventKindClosed       eventKind = "closed"
	eventKindReopened     eventKind = "reopened"
	eventKindRenamed      eventKind = "renamed"
	eventKindMilestoned   eventKind = "milestoned"
	eventKindDemilestoned eventKind = "demilestoned"
)
