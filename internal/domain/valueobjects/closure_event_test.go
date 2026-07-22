package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.ClosureEvent{}

func newClosureEventAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/issues/1")
}

func TestClosureEvent_Render_IncludesTheActionAndReasonInTheMetaLine(t *testing.T) {
	cases := []struct {
		name   string
		action valueobjects.ClosureAction
		reason string
		want   string
	}{
		{"closed with reason", valueobjects.ClosureActionClosed, "completed", `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","action":"closed","reason":"completed","url":"https://github.com/example/repo/issues/1"}} -->
`},
		{"reopened with no reason", valueobjects.ClosureActionReopened, "", `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","action":"reopened","reason":"","url":"https://github.com/example/repo/issues/1"}} -->
`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			event := valueobjects.NewClosureEvent(newClosureEventAttribution(t), c.action, c.reason)

			var buf strings.Builder
			if err := event.Render(&buf); err != nil {
				t.Fatalf("unexpected error rendering closure event: %v", err)
			}

			if buf.String() != c.want {
				t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), c.want)
			}
		})
	}
}

func TestNewClosureEvent_NormalizesReasonToEmptyForAReopenedAction(t *testing.T) {
	event := valueobjects.NewClosureEvent(newClosureEventAttribution(t), valueobjects.ClosureActionReopened, "completed")

	if event.Reason() != "" {
		t.Fatalf("Reason() = %q, want empty: a reopened action has no reason regardless of what's passed in", event.Reason())
	}
}

func TestClosureEvent_ExposesTheAttributionActionAndReasonItWasConstructedWith(t *testing.T) {
	attribution := newClosureEventAttribution(t)
	event := valueobjects.NewClosureEvent(attribution, valueobjects.ClosureActionClosed, "not_planned")

	if !event.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", event.Attribution(), attribution)
	}
	if event.Action() != valueobjects.ClosureActionClosed {
		t.Fatalf("Action() = %v, want %v", event.Action(), valueobjects.ClosureActionClosed)
	}
	if event.Reason() != "not_planned" {
		t.Fatalf("Reason() = %q, want %q", event.Reason(), "not_planned")
	}
}

func TestClosureEvent_Equals_TreatsMatchingAttributionActionAndReasonAsEqual(t *testing.T) {
	attribution := newClosureEventAttribution(t)
	a := valueobjects.NewClosureEvent(attribution, valueobjects.ClosureActionClosed, "completed")
	b := valueobjects.NewClosureEvent(attribution, valueobjects.ClosureActionClosed, "completed")

	if !a.Equals(b) {
		t.Fatal("expected closure events with matching attribution, action, and reason to be equal")
	}
}

func TestClosureEvent_Equals_TreatsDifferentActionsAsNotEqual(t *testing.T) {
	attribution := newClosureEventAttribution(t)
	a := valueobjects.NewClosureEvent(attribution, valueobjects.ClosureActionClosed, "completed")
	b := valueobjects.NewClosureEvent(attribution, valueobjects.ClosureActionReopened, "completed")

	if a.Equals(b) {
		t.Fatal("expected closure events with different actions to not be equal")
	}
}

func TestClosureEvent_Equals_TreatsDifferentReasonsAsNotEqual(t *testing.T) {
	attribution := newClosureEventAttribution(t)
	a := valueobjects.NewClosureEvent(attribution, valueobjects.ClosureActionClosed, "completed")
	b := valueobjects.NewClosureEvent(attribution, valueobjects.ClosureActionClosed, "not_planned")

	if a.Equals(b) {
		t.Fatal("expected closure events with different reasons to not be equal")
	}
}
