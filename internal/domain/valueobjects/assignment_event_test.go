package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.AssignmentEvent{}

func newAssignmentEventAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/issues/1")
}

func mustNewAssignmentEvent(t *testing.T, attribution valueobjects.Attribution, action valueobjects.AssignmentAction, assignee string) valueobjects.AssignmentEvent {
	t.Helper()
	event, err := valueobjects.NewAssignmentEvent(attribution, action, assignee)
	if err != nil {
		t.Fatalf("NewAssignmentEvent(): unexpected error: %v", err)
	}
	return event
}

func TestNewAssignmentEvent_RejectsAnEmptyAssignee(t *testing.T) {
	_, err := valueobjects.NewAssignmentEvent(newAssignmentEventAttribution(t), valueobjects.AssignmentActionAssigned, "")

	if err == nil {
		t.Fatal("expected an error for an empty assignee, got nil")
	}
}

func TestAssignmentEvent_Render_IncludesTheActionAndAssigneeInTheMetaLine(t *testing.T) {
	cases := []struct {
		name   string
		action valueobjects.AssignmentAction
		want   string
	}{
		{"assigned", valueobjects.AssignmentActionAssigned, "assigned"},
		{"unassigned", valueobjects.AssignmentActionUnassigned, "unassigned"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			event := mustNewAssignmentEvent(t, newAssignmentEventAttribution(t), c.action, "hubot")

			var buf strings.Builder
			if err := event.Render(&buf); err != nil {
				t.Fatalf("unexpected error rendering assignment event: %v", err)
			}

			want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","action":"` + c.want + `","assignee":"hubot","url":"https://github.com/example/repo/issues/1"}} -->
`
			if buf.String() != want {
				t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
			}
		})
	}
}

func TestAssignmentEvent_ExposesTheAttributionActionAndAssigneeItWasConstructedWith(t *testing.T) {
	attribution := newAssignmentEventAttribution(t)
	event := mustNewAssignmentEvent(t, attribution, valueobjects.AssignmentActionAssigned, "hubot")

	if !event.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", event.Attribution(), attribution)
	}
	if event.Action() != valueobjects.AssignmentActionAssigned {
		t.Fatalf("Action() = %v, want %v", event.Action(), valueobjects.AssignmentActionAssigned)
	}
	if event.Assignee() != "hubot" {
		t.Fatalf("Assignee() = %q, want %q", event.Assignee(), "hubot")
	}
}

func TestAssignmentEvent_Equals_TreatsMatchingAttributionActionAndAssigneeAsEqual(t *testing.T) {
	attribution := newAssignmentEventAttribution(t)
	a := mustNewAssignmentEvent(t, attribution, valueobjects.AssignmentActionAssigned, "hubot")
	b := mustNewAssignmentEvent(t, attribution, valueobjects.AssignmentActionAssigned, "hubot")

	if !a.Equals(b) {
		t.Fatal("expected assignment events with matching attribution, action, and assignee to be equal")
	}
}

func TestAssignmentEvent_Equals_TreatsDifferentActionsAsNotEqual(t *testing.T) {
	attribution := newAssignmentEventAttribution(t)
	a := mustNewAssignmentEvent(t, attribution, valueobjects.AssignmentActionAssigned, "hubot")
	b := mustNewAssignmentEvent(t, attribution, valueobjects.AssignmentActionUnassigned, "hubot")

	if a.Equals(b) {
		t.Fatal("expected assignment events with different actions to not be equal")
	}
}

func TestAssignmentEvent_Equals_TreatsDifferentAssigneesAsNotEqual(t *testing.T) {
	attribution := newAssignmentEventAttribution(t)
	a := mustNewAssignmentEvent(t, attribution, valueobjects.AssignmentActionAssigned, "hubot")
	b := mustNewAssignmentEvent(t, attribution, valueobjects.AssignmentActionAssigned, "octocat")

	if a.Equals(b) {
		t.Fatal("expected assignment events with different assignees to not be equal")
	}
}
