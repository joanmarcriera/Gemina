package gateway

import (
	"strings"
	"testing"

	"continuity-vpn/pkg/clientcore"
)

func TestDataGatewayHandshakeThenDataWithMetrics(t *testing.T) {
	idPriv, idPub, _ := clientcore.GenerateIdentity()
	service, key := hostedService(t)
	gw := NewDataGateway(idPriv, service, 64, nil)

	// 1. Client handshake -> gateway returns a ServerHello and admits.
	hello, hs, _ := clientcore.BeginClientHandshake(idPub, hostedToken(t, key))
	reply, rec := gw.HandleDatagram(hello)
	if reply == nil || rec.Kind != "client-hello" || !rec.Admitted {
		t.Fatalf("handshake not admitted: reply=%v rec=%+v", reply != nil, rec)
	}
	session, err := hs.Complete(reply, 64)
	if err != nil {
		t.Fatalf("client complete: %v", err)
	}

	// 2. Encrypted data flows; first copy delivered, second deduplicated.
	wire, _ := session.Outbound([]byte("hello gateway"))
	dreply, drec := gw.HandleDatagram(wire)
	if dreply != nil {
		t.Fatal("data datagram must not produce a reply")
	}
	if !drec.Deliver || string(drec.Payload) != "hello gateway" {
		t.Fatalf("data not delivered: %+v", drec)
	}
	if _, drec2 := gw.HandleDatagram(wire); drec2.Deliver {
		t.Fatal("duplicate datagram delivered twice")
	}

	// 3. Metrics reflect the real data path.
	out := gw.Metrics().Render()
	for _, want := range []string{
		`continuity_handshakes_total{result="admitted"} 1`,
		`continuity_data_packets_total{decision="first-copy"} 1`,
		`continuity_data_packets_total{decision="duplicate"} 1`,
		"continuity_active_sessions 1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics missing %q in:\n%s", want, out)
		}
	}
}

func TestDataGatewayRejectsUnentitledHandshake(t *testing.T) {
	idPriv, idPub, _ := clientcore.GenerateIdentity()
	service, _ := hostedService(t)
	gw := NewDataGateway(idPriv, service, 64, nil)

	hello, _, _ := clientcore.BeginClientHandshake(idPub, "") // no token in hosted mode
	reply, rec := gw.HandleDatagram(hello)
	if reply != nil {
		t.Fatal("rejected handshake must not return a ServerHello")
	}
	if rec.Admitted {
		t.Fatal("unentitled client marked admitted")
	}
	if !strings.Contains(gw.Metrics().Render(), `continuity_handshakes_total{result="rejected"} 1`) {
		t.Fatalf("rejected handshake not counted:\n%s", gw.Metrics().Render())
	}
}

func TestDataGatewayRejectsJunkAndUnknownSession(t *testing.T) {
	idPriv, _, _ := clientcore.GenerateIdentity()
	service, _ := hostedService(t)
	gw := NewDataGateway(idPriv, service, 64, nil)

	// Junk.
	if reply, _ := gw.HandleDatagram([]byte("not a frame")); reply != nil {
		t.Fatal("junk produced a reply")
	}
	// Valid-looking data for a session that was never admitted.
	id := sessionID(0x7E)
	client, _ := clientcore.NewSession(id, []byte("0123456789abcdef0123456789abcdef"), clientcore.RoleInitiator, 16)
	wire, _ := client.Outbound([]byte("x"))
	_, drec := gw.HandleDatagram(wire)
	if drec.Deliver {
		t.Fatal("unadmitted-session data delivered")
	}
	if !strings.Contains(gw.Metrics().Render(), `continuity_data_packets_total{decision="rejected"}`) {
		t.Fatalf("rejected data not counted:\n%s", gw.Metrics().Render())
	}
}
