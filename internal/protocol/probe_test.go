package protocol

import (
	"errors"
	"testing"
)

func testSession(b byte) SessionID {
	var s SessionID
	for i := range s {
		s[i] = b
	}
	return s
}

func TestProbeMarshalUnmarshalRoundTrip(t *testing.T) {
	want := ProbePacket{
		ID:   PacketID{Session: testSession(0x5a), Number: 42},
		Path: PathWiFi,
	}

	wire, err := want.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(wire) != ProbeWireSize {
		t.Fatalf("wire size = %d, want %d", len(wire), ProbeWireSize)
	}

	got, err := UnmarshalProbe(wire)
	if err != nil {
		t.Fatalf("UnmarshalProbe: %v", err)
	}
	if got != want {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}

func TestMarshalRejectsInvalidPacketID(t *testing.T) {
	// Zero session and zero number are both invalid packet identities.
	for _, p := range []ProbePacket{
		{ID: PacketID{Session: SessionID{}, Number: 1}, Path: PathWiFi},
		{ID: PacketID{Session: testSession(1), Number: 0}, Path: PathWiFi},
	} {
		if _, err := p.MarshalBinary(); !errors.Is(err, ErrInvalidProbe) {
			t.Fatalf("MarshalBinary(%+v) error = %v, want ErrInvalidProbe", p, err)
		}
	}
}

func TestUnmarshalProbeRejectsMalformed(t *testing.T) {
	valid, err := ProbePacket{ID: PacketID{Session: testSession(7), Number: 9}, Path: PathAndroidUSBTether}.MarshalBinary()
	if err != nil {
		t.Fatalf("setup marshal: %v", err)
	}

	tests := []struct {
		name string
		in   []byte
		want error
	}{
		{name: "empty", in: nil, want: ErrShortProbe},
		{name: "truncated", in: valid[:ProbeWireSize-1], want: ErrShortProbe},
		{name: "bad magic", in: corrupt(valid, 0, 'X'), want: ErrBadMagic},
		{name: "unsupported version", in: corrupt(valid, 4, 0xFF), want: ErrUnsupportedVersion},
		{name: "invalid id (zero session)", in: zeroSession(valid), want: ErrInvalidProbe},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalProbe(tt.in)
			if !errors.Is(err, tt.want) {
				t.Fatalf("UnmarshalProbe(%s) error = %v, want %v", tt.name, err, tt.want)
			}
		})
	}
}

func TestUnmarshalProbeIgnoresTrailingBytes(t *testing.T) {
	valid, err := ProbePacket{ID: PacketID{Session: testSession(3), Number: 5}, Path: PathWiFi}.MarshalBinary()
	if err != nil {
		t.Fatalf("setup marshal: %v", err)
	}
	padded := append(valid, 0xde, 0xad)

	got, err := UnmarshalProbe(padded)
	if err != nil {
		t.Fatalf("UnmarshalProbe(padded): %v", err)
	}
	if got.ID.Number != 5 {
		t.Fatalf("number = %d, want 5", got.ID.Number)
	}
}

func TestPathTagString(t *testing.T) {
	cases := map[PathTag]string{
		PathUnknown:          "unknown",
		PathWiFi:             "wi-fi",
		PathAndroidUSBTether: "android-usb-tether",
		PathTag(200):         "unknown",
	}
	for tag, want := range cases {
		if got := tag.String(); got != want {
			t.Fatalf("PathTag(%d).String() = %q, want %q", tag, got, want)
		}
	}
}

func corrupt(b []byte, i int, v byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	out[i] = v
	return out
}

func zeroSession(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	for i := ProbeSessionOffset; i < ProbeSessionOffset+SessionIDSize; i++ {
		out[i] = 0
	}
	return out
}
