package valueobjects

import (
	"strings"
	"testing"
)

// TestWriteMetaLine_WrapsAMetaMarshalFailure is a deliberate exception to
// testing only through exported entry points: no Tier 1 type's Render()
// can ever pass writeMetaLine a value that fails to marshal (each only ever
// passes its own always-marshalable attributionMeta-embedding struct), so
// this branch is unreachable from any public caller. It is exercised here
// directly, by constructing an unmarshalable value, since that is the only
// way to verify it at all.
func TestWriteMetaLine_WrapsAMetaMarshalFailure(t *testing.T) {
	var buf strings.Builder
	err := writeMetaLine(&buf, make(chan int), "unused")

	if err == nil {
		t.Fatal("expected an error for an unmarshalable meta value, got nil")
	}
	if !strings.Contains(err.Error(), "marshal meta") {
		t.Fatalf("expected the error to mention meta marshaling, got: %v", err)
	}
}
