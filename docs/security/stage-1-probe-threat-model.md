# Stage 1 Probe Threat Model

## Scope

Component: Stage 1 packet identity and duplicate-suppression core.

Stage: first UDP probe slice.

In scope:

* `internal/protocol` packet identifiers.
* `internal/dedup` first-copy acceptance decisions.
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
* Diagnostic output could later be expanded to include private traffic or secrets.

## Mitigations

* `PacketID.Valid` rejects zero sessions and zero packet numbers.
* `dedup.Window` rejects invalid packet IDs and empty path labels.
* `dedup.Window` serialises access with a mutex and is covered by a race-detector test.
* `dedup.Window` has an explicit capacity and evicts the oldest packet ID when full.
* Current result fields contain identifiers, path labels and decisions only; no payloads, access keys or private keys are present.

## Residual Risk

The current dedup window is in-memory and local to one process. It is enough for the first probe but not for a production gateway, distributed gateway, replay defence or restart survival.

Future gateway work must revisit denial-of-service controls, replay windows, authenticated packet identity and logging redaction before any real traffic is handled.
