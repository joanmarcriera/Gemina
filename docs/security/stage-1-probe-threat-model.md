# Stage 1 Probe Threat Model

## Scope

Component: Stage 1 probe core below live path binding.

Stage: first UDP probe slice.

In scope:

* `internal/protocol` packet identifiers.
* `internal/dedup` first-copy acceptance decisions.
* `internal/paths` fixture-driven path-candidate classification.
* `internal/platform/darwin` injected interface snapshots and conservative
  BSD interface-state collection.
* diagnostic decision values: `first-copy`, `duplicate`, `invalid`.

Out of scope:

* cryptography;
* access keys;
* NetworkExtension packet capture;
* gateway deployment;
* payment and entitlement state;
* private user traffic payloads.

## Assets

* Probe packet identity.
* Probe session continuity evidence.
* Future protected traffic metadata.

## Threats

* A malformed packet ID could be accepted and pollute dedup state.
* Duplicate observations could race and deliver more than one first copy.
* Unbounded dedup state could become a memory-exhaustion vector.
* Path classification could select the wrong interface role if it relies on names rather than observed link kind.
* Live interface-state collection could accidentally expose source IP addresses
  or other local network details in evidence records.
* Darwin observation evidence could misclassify a generic USB network adapter as Android USB tethering.
* Diagnostic output could later be expanded to include private traffic or secrets.

## Mitigations

* `PacketID.Valid` rejects zero sessions and zero packet numbers.
* `dedup.Window` rejects invalid packet IDs and empty path labels.
* `dedup.Window` serialises access with a mutex and is covered by a race-detector test.
* `dedup.Window` has an explicit capacity and evicts the oldest packet ID when full.
* `paths.Classify` uses fixture-provided `LinkKind` values and rejects missing or ambiguous candidates instead of guessing from interface names.
* `platform/darwin` tests verify BSD names and display names are preserved as data but do not assign link kind by themselves.
* `platform/darwin` live collector records only flags and IPv4 presence from
  BSD interface state, not source IP addresses.
* `platform/darwin.NetInterfaceSource` sets `LinkKindUnknown`; later macOS
  evidence sources must explicitly assign Wi-Fi or Android USB tethering kinds.
* Current result fields contain identifiers, path labels and decisions only; no payloads, access keys or private keys are present.

## Residual Risk

The current dedup window is in-memory and local to one process. It is enough for the first probe but not for a production gateway, distributed gateway, replay defence or restart survival.

The current path classifier only handles injected observations. It is not live macOS path discovery, socket binding or proof that traffic left through a specific interface.

The current Darwin live collector only records BSD interface state. It cannot
identify Wi-Fi or Android USB tethering by itself. Future live observation code
must treat Android USB tethering as untrusted until USB association evidence and
packet captures confirm the path.

Future gateway work must revisit denial-of-service controls, replay windows, authenticated packet identity and logging redaction before any real traffic is handled.
