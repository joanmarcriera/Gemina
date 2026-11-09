package gateway

import (
	"strings"
	"testing"

	"github.com/joanmarcriera/gemina/internal/entitlement"
	"github.com/joanmarcriera/gemina/pkg/clientcore"
)

// TestDataGatewayExitMetrics pins that the exit.Metrics seam on DataGateway
// lands in the redacted Prometheus counters: drops are labelled by reason and
// forwarded/returned counts accumulate packets and bytes.
func TestDataGatewayExitMetrics(t *testing.T) {
	idPriv, _, err := clientcore.GenerateIdentity()
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}
	dg := NewDataGateway(idPriv, &entitlement.Service{Mode: entitlement.ModeOpen}, 64, nil)

	dg.Dropped("reverse-path")
	dg.Dropped("reverse-path")
	dg.Dropped("no-lease")
	dg.Forwarded(100)
	dg.Forwarded(20)
	dg.Returned(7)

	out := dg.Metrics().Render()
	for _, want := range []string{
		`gemina_exit_dropped_total{reason="reverse-path"} 2`,
		`gemina_exit_dropped_total{reason="no-lease"} 1`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, `reason="tun-write-error"`) {
		t.Errorf("unseen reason rendered:\n%s", out)
	}
}
