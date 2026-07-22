package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.RenameEvent{}

func newRenameEventAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/issues/1")
}

func mustNewRenameEvent(t *testing.T, attribution valueobjects.Attribution, from, to string) valueobjects.RenameEvent {
	t.Helper()
	event, err := valueobjects.NewRenameEvent(attribution, from, to)
	if err != nil {
		t.Fatalf("NewRenameEvent(): unexpected error: %v", err)
	}
	return event
}

func TestNewRenameEvent_RejectsAnEmptyTo(t *testing.T) {
	_, err := valueobjects.NewRenameEvent(newRenameEventAttribution(t), "Old title", "")

	if err == nil {
		t.Fatal("expected an error for an empty renamed-to title, got nil")
	}
}

func TestRenameEvent_Render_IncludesFromAndToInTheMetaLine(t *testing.T) {
	event := mustNewRenameEvent(t, newRenameEventAttribution(t), "Old title", "New title")

	var buf strings.Builder
	if err := event.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering rename event: %v", err)
	}

	want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","from":"Old title","to":"New title","url":"https://github.com/example/repo/issues/1"}} -->
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestRenameEvent_ExposesTheAttributionFromAndToItWasConstructedWith(t *testing.T) {
	attribution := newRenameEventAttribution(t)
	event := mustNewRenameEvent(t, attribution, "Old title", "New title")

	if !event.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", event.Attribution(), attribution)
	}
	if event.From() != "Old title" {
		t.Fatalf("From() = %q, want %q", event.From(), "Old title")
	}
	if event.To() != "New title" {
		t.Fatalf("To() = %q, want %q", event.To(), "New title")
	}
}

func TestRenameEvent_Equals_TreatsMatchingAttributionFromAndToAsEqual(t *testing.T) {
	attribution := newRenameEventAttribution(t)
	a := mustNewRenameEvent(t, attribution, "Old title", "New title")
	b := mustNewRenameEvent(t, attribution, "Old title", "New title")

	if !a.Equals(b) {
		t.Fatal("expected rename events with matching attribution, from, and to to be equal")
	}
}

func TestRenameEvent_Equals_TreatsDifferentFromAsNotEqual(t *testing.T) {
	attribution := newRenameEventAttribution(t)
	a := mustNewRenameEvent(t, attribution, "Old title", "New title")
	b := mustNewRenameEvent(t, attribution, "Different old title", "New title")

	if a.Equals(b) {
		t.Fatal("expected rename events with different from values to not be equal")
	}
}

func TestRenameEvent_Equals_TreatsDifferentToAsNotEqual(t *testing.T) {
	attribution := newRenameEventAttribution(t)
	a := mustNewRenameEvent(t, attribution, "Old title", "New title")
	b := mustNewRenameEvent(t, attribution, "Old title", "Different new title")

	if a.Equals(b) {
		t.Fatal("expected rename events with different to values to not be equal")
	}
}
