package icons_test

import (
	"strings"
	"testing"

	"github.com/coreofscience/go-bibx/mcp/internal/icons"
)

func TestAll(t *testing.T) {
	all := icons.All()
	if len(all) != 6 {
		t.Fatalf("expected 6 icons, got %d", len(all))
	}
	for _, icon := range all {
		if !strings.HasPrefix(icon.Source, "data:image/png;base64,") {
			t.Errorf("unexpected source prefix: %.30s", icon.Source)
		}
		if icon.Theme == "" || len(icon.Sizes) != 1 {
			t.Errorf("incomplete icon: %+v", icon)
		}
	}
}
