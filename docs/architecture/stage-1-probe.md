# Stage 1 Probe Architecture

## Objective

Prove the smallest useful dual-path UDP probe before building VPN behaviour:

* UDP socket A explicitly leaves through Wi-Fi.
* UDP socket B explicitly leaves through Android USB tethering.
* Both copies reach one gateway process.
* One logical packet is delivered once.
* Either path can disappear without ending the logical probe session.

## Current Slice

This repository currently implements only a unit-testable core below the live network layer:

* `internal/protocol` defines stable probe packet identity:
  * `SessionID`
  * `PacketNumber`
  * `PacketID`
* `internal/dedup` defines a bounded in-memory first-copy acceptance window:
  * valid first copy -> `first-copy`
  * later copy with the same `PacketID` -> `duplicate`
  * invalid packet ID or empty path label -> `invalid`
* `internal/paths` classifies injected path observations:
  * usable Wi-Fi observation -> Wi-Fi candidate
  * usable Android USB tethering observation -> Android USB tethering candidate
  * missing or multiple usable observations -> structured issue

The window records path labels such as `wifi` and `usb-tether`, but those labels are not proof of macOS interface binding. They are placeholders for future observations from real sockets and packet captures.

The path classifier consumes enum-like link kinds from an injected observation source. It does not infer roles from BSD interface names such as `en0`.

## Explicitly Out Of Scope For This Slice

This slice does not implement:

* live macOS interface enumeration;
* source-address or socket binding;
* gateway networking;
* packet serialisation;
* encryption or WireGuard integration;
* replay protection beyond the bounded duplicate window;
* NetworkExtension packet handling;
* packet captures;
* claims that Wi-Fi or Android USB tethering were used.

## Package Boundaries

`internal/protocol` owns identity validation and string formatting for diagnostic output.

`internal/dedup` owns duplicate-suppression state for probe packets. It must not log private traffic or access keys. It accepts caller-provided path labels only as observations.

`internal/paths` will later own macOS path discovery and interface selection. It must avoid hard-coded interface names.

The current `internal/paths` implementation only classifies fixture data. A future platform adapter must populate observations from macOS APIs and still prove egress with packet captures.

`internal/gateway` will later connect UDP receive loops to the dedup window and evidence logs.

## Evidence Required Before Claiming Path Success

The Stage 1 probe is not successful until the project has:

* packet captures showing distinct Wi-Fi and Android USB tethering egress;
* gateway logs showing both copies of the same logical packet reaching one process;
* test logs showing one first-copy decision and one duplicate decision for the same `PacketID`;
* failure evidence showing the logical probe session survives loss of either path;
* documentation of the exact macOS interface identifiers observed during the test run, without hard-coding them into product code.
