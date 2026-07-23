package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.LabelEvent{}

func newLabelEventAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/issues/1")
}

func mustNewLabelEvent(t *testing.T, attribution valueobjects.Attribution, action valueobjects.LabelAction, name, color string) valueobjects.LabelEvent {
	t.Helper()
	event, err := valueobjects.NewLabelEvent(attribution, action, name, color)
	if err != nil {
		t.Fatalf("NewLabelEvent(): unexpected error: %v", err)
	}
	return event
}

func TestNewLabelEvent_RejectsAnEmptyName(t *testing.T) {
	_, err := valueobjects.NewLabelEvent(newLabelEventAttribution(t), valueobjects.LabelActionLabeled, "", "d73a4a")

	if err == nil {
		t.Fatal("expected an error for an empty label name, got nil")
	}
}

func TestLabelEvent_Render_IncludesTheActionLabelAndColorInTheMetaLine(t *testing.T) {
	cases := []struct {
		name   string
		action valueobjects.LabelAction
		want   string
		body   string
	}{
		{"labeled", valueobjects.LabelActionLabeled, "labeled", "Labeled `bug`"},
		{"unlabeled", valueobjects.LabelActionUnlabeled, "unlabeled", "Unlabeled `bug`"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			event := mustNewLabelEvent(t, newLabelEventAttribution(t), c.action, "bug", "d73a4a")

			var buf strings.Builder
			if err := event.Render(&buf); err != nil {
				t.Fatalf("unexpected error rendering label event: %v", err)
			}

			want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","action":"` + c.want + `","label":"bug","color":"d73a4a","url":"https://github.com/example/repo/issues/1"}} -->

` + c.body + `
`
			if buf.String() != want {
				t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
			}
		})
	}
}

func TestLabelEvent_Render_FencesALabelNameContainingABacktick(t *testing.T) {
	event := mustNewLabelEvent(t, newLabelEventAttribution(t), valueobjects.LabelActionLabeled, "foo`bar", "d73a4a")

	var buf strings.Builder
	if err := event.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering label event: %v", err)
	}

	want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","action":"labeled","label":"foo` + "`" + `bar","color":"d73a4a","url":"https://github.com/example/repo/issues/1"}} -->

Labeled ` + "``foo`bar``" + `
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestLabelEvent_Render_FallsBackToTheActionsStringForAnUnrecognizedLabelAction(t *testing.T) {
	event := mustNewLabelEvent(t, newLabelEventAttribution(t), valueobjects.LabelAction(99), "bug", "d73a4a")

	var buf strings.Builder
	if err := event.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering label event: %v", err)
	}

	if !strings.Contains(buf.String(), "LabelAction(99) `bug`") {
		t.Fatalf("Render() = %q, want it to contain %q", buf.String(), "LabelAction(99) `bug`")
	}
}

func TestLabelEvent_ExposesTheAttributionActionNameAndColorItWasConstructedWith(t *testing.T) {
	attribution := newLabelEventAttribution(t)
	event := mustNewLabelEvent(t, attribution, valueobjects.LabelActionLabeled, "bug", "d73a4a")

	if !event.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", event.Attribution(), attribution)
	}
	if event.Action() != valueobjects.LabelActionLabeled {
		t.Fatalf("Action() = %v, want %v", event.Action(), valueobjects.LabelActionLabeled)
	}
	if event.Name() != "bug" {
		t.Fatalf("Name() = %q, want %q", event.Name(), "bug")
	}
	if event.Color() != "d73a4a" {
		t.Fatalf("Color() = %q, want %q", event.Color(), "d73a4a")
	}
}

func TestLabelEvent_Equals_TreatsMatchingAttributionActionNameAndColorAsEqual(t *testing.T) {
	attribution := newLabelEventAttribution(t)
	a := mustNewLabelEvent(t, attribution, valueobjects.LabelActionLabeled, "bug", "d73a4a")
	b := mustNewLabelEvent(t, attribution, valueobjects.LabelActionLabeled, "bug", "d73a4a")

	if !a.Equals(b) {
		t.Fatal("expected label events with matching attribution, action, name, and color to be equal")
	}
}

func TestLabelEvent_Equals_TreatsDifferentActionsAsNotEqual(t *testing.T) {
	attribution := newLabelEventAttribution(t)
	a := mustNewLabelEvent(t, attribution, valueobjects.LabelActionLabeled, "bug", "d73a4a")
	b := mustNewLabelEvent(t, attribution, valueobjects.LabelActionUnlabeled, "bug", "d73a4a")

	if a.Equals(b) {
		t.Fatal("expected label events with different actions to not be equal")
	}
}

func TestLabelEvent_Equals_TreatsDifferentNamesAsNotEqual(t *testing.T) {
	attribution := newLabelEventAttribution(t)
	a := mustNewLabelEvent(t, attribution, valueobjects.LabelActionLabeled, "bug", "d73a4a")
	b := mustNewLabelEvent(t, attribution, valueobjects.LabelActionLabeled, "enhancement", "d73a4a")

	if a.Equals(b) {
		t.Fatal("expected label events with different names to not be equal")
	}
}

func TestLabelEvent_Equals_TreatsDifferentColorsAsNotEqual(t *testing.T) {
	attribution := newLabelEventAttribution(t)
	a := mustNewLabelEvent(t, attribution, valueobjects.LabelActionLabeled, "bug", "d73a4a")
	b := mustNewLabelEvent(t, attribution, valueobjects.LabelActionLabeled, "bug", "0dd8ac")

	if a.Equals(b) {
		t.Fatal("expected label events with different colors to not be equal")
	}
}
