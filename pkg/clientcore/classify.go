package clientcore

// DatagramKind classifies a received datagram so a gateway can route it to the
// handshake or the data plane without decoding it fully.
type DatagramKind int

const (
	KindUnknown DatagramKind = iota
	KindData
	KindClientHello
	KindServerHello
	KindPing
)

func (k DatagramKind) String() string {
	switch k {
	case KindData:
		return "data"
	case KindClientHello:
		return "client-hello"
	case KindServerHello:
		return "server-hello"
	case KindPing:
		return "ping"
	default:
		return "unknown"
	}
}

// ClassifyDatagram inspects the magic (and, for handshakes, the message type) of
// a datagram. It does not validate the rest of the frame.
func ClassifyDatagram(b []byte) DatagramKind {
	if len(b) < 4 {
		return KindUnknown
	}
	magic := [4]byte{b[0], b[1], b[2], b[3]}
	switch magic {
	case dataMagic:
		return KindData
	case pingMagic:
		return KindPing
	case handshakeMagic:
		if len(b) < 6 {
			return KindUnknown
		}
		switch b[5] {
		case msgClientHello:
			return KindClientHello
		case msgServerHello:
			return KindServerHello
		}
	}
	return KindUnknown
}
