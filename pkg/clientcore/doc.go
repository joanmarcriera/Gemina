// Package clientcore is the client-side dual-path transport core: it frames
// outbound tunnel packets with a per-session identity so identical copies can be
// sent over several paths (Wi-Fi and the Android tether), and deduplicates
// inbound packets so each logical packet is delivered once regardless of how
// many paths carried it. It is transport-agnostic — paths are opaque to it — so
// it can be driven by the macOS NEPacketTunnelProvider over the narrow Swift/Go
// boundary (ADR-0002) and exercised end-to-end in tests without any sockets.
package clientcore
