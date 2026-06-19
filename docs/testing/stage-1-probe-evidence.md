# Stage 1 Probe Evidence

The first Stage 1 implementation slices have unit tests for packet identity,
duplicate suppression, path-candidate classification and conservative Darwin
interface-state collection. These tests prove local classification, first-copy
semantics and state mapping inside the Go process; they do not prove path
binding, gateway reachability or VPN behaviour.

## Unit Evidence

Required local checks for the current slice:

* `go test ./internal/protocol ./internal/dedup`
* `go test ./internal/paths`
* `go test ./internal/platform/darwin`
* `go test -race ./internal/dedup ./internal/protocol`
* `go test -race ./internal/paths`
* `go test -race ./internal/platform/darwin`

The dedup tests must cover:

* exact session-ID size validation;
* invalid packet IDs;
* first copy accepted;
* later copies with the same packet ID rejected as duplicates;
* empty path labels rejected;
* bounded-window eviction;
* concurrent duplicate observation with one first-copy result.

The path-classification tests must cover:

* Wi-Fi and Android USB tethering candidates selected from injected observations;
* fake non-macOS-style identifiers to prove classification does not depend on names such as `en0`;
* unusable observations ignored;
* missing candidates reported;
* ambiguous candidates reported;
* unknown link kinds ignored.

The Darwin observation-boundary tests must cover:

* snapshot fields preserved into `paths.Observation`;
* fixture snapshots feeding the path classifier;
* BSD names and display names not assigning link kind by themselves;
* missing BSD names remaining unusable;
* evidence metadata filtered by source;
* conservative live-collector mapping from injected interface records to snapshots;
* collector preservation of explicit link kind evidence without inferring roles from names;
* fixture-backed derivation of Wi-Fi from Network framework and SystemConfiguration evidence;
* fixture-backed derivation of Android USB tethering from IORegistry evidence;
* generic USB network adapters remaining unknown rather than Android USB tethering;
* conflicting Wi-Fi and Android USB evidence remaining unknown;
* command-backed live evidence merging with conservative BSD interface state;
* `networksetup` hardware-port output reduced to Wi-Fi evidence without storing MAC addresses;
* `ioreg` Ethernet-interface output reduced to Android USB evidence without storing raw product names or serial-like values;
* IPv6-only interfaces remaining unusable for the IPv4-only Stage 1 probe;
* source and nil-source error handling;
* evidence-source string values.

## Integration Evidence Not Yet Available

Before Stage 1 can claim dual-path success, collect:

* local socket-binding evidence showing one UDP path leaves through Wi-Fi and one through Android USB tethering;
* live macOS observation evidence showing how Wi-Fi and Android USB tethering link kinds were assigned;
* live SystemConfiguration, Network framework and IORegistry collection evidence
  matching the redacted fixture shape under `internal/platform/darwin/testdata/`;
* a redacted diagnostic command output from `LiveEvidenceInterfaceSnapshots` on
  the target Mac with Wi-Fi and Android USB tethering both connected;
* packet captures from the Mac and gateway for the same probe session;
* gateway logs showing both copies of the same `PacketID`;
* loss tests where Wi-Fi disappears and the logical probe continues over USB tethering;
* loss tests where USB tethering disappears and the logical probe continues over Wi-Fi.

Do not claim VPN continuity, packet-loss improvement or working path failover until this evidence exists.
