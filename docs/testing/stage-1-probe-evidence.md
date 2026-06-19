# Stage 1 Probe Evidence

The first Stage 1 implementation slice has unit tests for packet identity and duplicate suppression only. These tests prove first-copy semantics inside the Go process; they do not prove path binding, gateway reachability or VPN behaviour.

## Unit Evidence

Required local checks for the current slice:

* `go test ./internal/protocol ./internal/dedup`
* `go test -race ./internal/dedup ./internal/protocol`

The dedup tests must cover:

* exact session-ID size validation;
* invalid packet IDs;
* first copy accepted;
* later copies with the same packet ID rejected as duplicates;
* empty path labels rejected;
* bounded-window eviction;
* concurrent duplicate observation with one first-copy result.

## Integration Evidence Not Yet Available

Before Stage 1 can claim dual-path success, collect:

* local socket-binding evidence showing one UDP path leaves through Wi-Fi and one through Android USB tethering;
* packet captures from the Mac and gateway for the same probe session;
* gateway logs showing both copies of the same `PacketID`;
* loss tests where Wi-Fi disappears and the logical probe continues over USB tethering;
* loss tests where USB tethering disappears and the logical probe continues over Wi-Fi.

Do not claim VPN continuity, packet-loss improvement or working path failover until this evidence exists.
