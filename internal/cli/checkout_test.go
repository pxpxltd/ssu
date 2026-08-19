package cli

import (
	"errors"
	"testing"

	"github.com/pxpxltd/ssu/internal/engine"
)

// Counts are derived from the completed action list rather than from an
// OnProgress callback, which the engine fires from several goroutines.
func TestCountCheckoutActions(t *testing.T) {
	actions := []engine.CheckoutAction{
		{Path: "a", Action: "checked out develop"},
		{Path: "b", Action: "detached at abc1234"},
		{Path: "c", Action: "skipped (not detached)"},
		{Path: "d", Action: "skipped (dirty working tree)"},
		{Path: "e", Action: "checkout failed", Error: errors.New("boom")},
		// An error action wins over its "skipped" wording.
		{Path: "f", Action: "skipped (scan failed)", Error: errors.New("fetch failed")},
	}

	checked, skipped, failed := countCheckoutActions(actions)
	if checked != 2 {
		t.Errorf("checked = %d, want 2", checked)
	}
	if skipped != 2 {
		t.Errorf("skipped = %d, want 2", skipped)
	}
	if failed != 2 {
		t.Errorf("failed = %d, want 2", failed)
	}
}

func TestCountCheckoutActionsEmpty(t *testing.T) {
	checked, skipped, failed := countCheckoutActions(nil)
	if checked != 0 || skipped != 0 || failed != 0 {
		t.Errorf("expected all zero, got %d/%d/%d", checked, skipped, failed)
	}
}
