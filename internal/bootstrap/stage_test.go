package bootstrap

import "testing"

func TestComponentStage(t *testing.T) {
	if got := ComponentStage("gateway"); got != "gateway:stage-1-probe" {
		t.Fatalf("ComponentStage() = %q", got)
	}
}
