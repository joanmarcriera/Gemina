package bootstrap

import "testing"

func TestComponentStage(t *testing.T) {
	if got := ComponentStage("gateway"); got != "gateway:stage-0-bootstrap" {
		t.Fatalf("ComponentStage() = %q", got)
	}
}
