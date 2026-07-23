package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.MilestoneEvent{}

func newMilestoneEventAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/issues/1")
}

func mustNewMilestoneEvent(t *testing.T, attribution valueobjects.Attribution, action valueobjects.MilestoneAction, title string) valueobjects.MilestoneEvent {
	t.Helper()
	event, err := valueobjects.NewMilestoneEvent(attribution, action, title)
	if err != nil {
		t.Fatalf("NewMilestoneEvent(): unexpected error: %v", err)
	}
	return event
}

func TestNewMilestoneEvent_RejectsAnEmptyTitle(t *testing.T) {
	_, err := valueobjects.NewMilestoneEvent(newMilestoneEventAttribution(t), valueobjects.MilestoneActionMilestoned, "")

	if err == nil {
		t.Fatal("expected an error for an empty milestone title, got nil")
	}
}

func TestMilestoneEvent_Render_IncludesTheActionAndTitleInTheMetaLine(t *testing.T) {
	cases := []struct {
		name   string
		action valueobjects.MilestoneAction
		want   string
		body   string
	}{
		{"milestoned", valueobjects.MilestoneActionMilestoned, "milestoned", "Milestoned `v1.0`"},
		{"demilestoned", valueobjects.MilestoneActionDemilestoned, "demilestoned", "Demilestoned `v1.0`"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			event := mustNewMilestoneEvent(t, newMilestoneEventAttribution(t), c.action, "v1.0")

			var buf strings.Builder
			if err := event.Render(&buf); err != nil {
				t.Fatalf("unexpected error rendering milestone event: %v", err)
			}

			want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","action":"` + c.want + `","milestone":"v1.0","url":"https://github.com/example/repo/issues/1"}} -->

` + c.body + `
`
			if buf.String() != want {
				t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
			}
		})
	}
}

func TestMilestoneEvent_Render_FencesATitleContainingABacktick(t *testing.T) {
	event := mustNewMilestoneEvent(t, newMilestoneEventAttribution(t), valueobjects.MilestoneActionMilestoned, "v1`0")

	var buf strings.Builder
	if err := event.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering milestone event: %v", err)
	}

	want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","action":"milestoned","milestone":"v1` + "`" + `0","url":"https://github.com/example/repo/issues/1"}} -->

Milestoned ` + "``v1`0``" + `
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestMilestoneEvent_Render_FallsBackToTheActionsStringForAnUnrecognizedMilestoneAction(t *testing.T) {
	event := mustNewMilestoneEvent(t, newMilestoneEventAttribution(t), valueobjects.MilestoneAction(99), "v1.0")

	var buf strings.Builder
	if err := event.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering milestone event: %v", err)
	}

	if !strings.Contains(buf.String(), "MilestoneAction(99) `v1.0`") {
		t.Fatalf("Render() = %q, want it to contain %q", buf.String(), "MilestoneAction(99) `v1.0`")
	}
}

func TestMilestoneEvent_ExposesTheAttributionActionAndTitleItWasConstructedWith(t *testing.T) {
	attribution := newMilestoneEventAttribution(t)
	event := mustNewMilestoneEvent(t, attribution, valueobjects.MilestoneActionMilestoned, "v1.0")

	if !event.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", event.Attribution(), attribution)
	}
	if event.Action() != valueobjects.MilestoneActionMilestoned {
		t.Fatalf("Action() = %v, want %v", event.Action(), valueobjects.MilestoneActionMilestoned)
	}
	if event.Title() != "v1.0" {
		t.Fatalf("Title() = %q, want %q", event.Title(), "v1.0")
	}
}

func TestMilestoneEvent_Equals_TreatsMatchingAttributionActionAndTitleAsEqual(t *testing.T) {
	attribution := newMilestoneEventAttribution(t)
	a := mustNewMilestoneEvent(t, attribution, valueobjects.MilestoneActionMilestoned, "v1.0")
	b := mustNewMilestoneEvent(t, attribution, valueobjects.MilestoneActionMilestoned, "v1.0")

	if !a.Equals(b) {
		t.Fatal("expected milestone events with matching attribution, action, and title to be equal")
	}
}

func TestMilestoneEvent_Equals_TreatsDifferentActionsAsNotEqual(t *testing.T) {
	attribution := newMilestoneEventAttribution(t)
	a := mustNewMilestoneEvent(t, attribution, valueobjects.MilestoneActionMilestoned, "v1.0")
	b := mustNewMilestoneEvent(t, attribution, valueobjects.MilestoneActionDemilestoned, "v1.0")

	if a.Equals(b) {
		t.Fatal("expected milestone events with different actions to not be equal")
	}
}

func TestMilestoneEvent_Equals_TreatsDifferentTitlesAsNotEqual(t *testing.T) {
	attribution := newMilestoneEventAttribution(t)
	a := mustNewMilestoneEvent(t, attribution, valueobjects.MilestoneActionMilestoned, "v1.0")
	b := mustNewMilestoneEvent(t, attribution, valueobjects.MilestoneActionMilestoned, "v2.0")

	if a.Equals(b) {
		t.Fatal("expected milestone events with different titles to not be equal")
	}
}
