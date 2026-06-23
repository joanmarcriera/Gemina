# ADR-0005: Dual-Path Data Plane and Swift/Go Bridge

Date: 2026-06-23

## Status

Accepted

## Context

The dual-path transport is proven end-to-end (see `PROJECT_STATE.md`, "Definition
of Dual-Path Success — ACHIEVED"): the same logical packet sent over Wi-Fi and
the phone's cellular link (userspace RNDIS) both reach the gateway, are
deduplicated to one delivery, and either path can drop without ending the
session. That proof lives in the `research/usb-rndis-spike/` drivers.

ADR-0002 set the shape: Swift app + NetworkExtension, Go transport core, a narrow
C boundary. This ADR decides **how** the macOS `NEPacketTunnelProvider` drives
the proven dual-path duplicate/deduplicate behaviour for real tunnelled traffic,
without re-deriving the transport brain in Swift.

## Decision

1. **The transport brain is Go (`pkg/clientcore.Session`).** It frames an
   outbound tunnel packet with a per-session identity (`CVD1` wire format:
   magic, version, flags, 16-byte session, 8-byte number, payload) and
   deduplicates inbound packets by identity, delivering each logical packet once
   regardless of how many paths carried it. It is transport-agnostic — paths are
   opaque strings — so it is fully unit-tested without sockets (including a
   concurrent two-path race test).

2. **The `NEPacketTunnelProvider` (Swift) owns I/O and paths.** It:
   - reads outbound IP packets from the virtual interface (`packetFlow`);
   - calls the core to frame each packet, then sends the **same** framed bytes
     over every active path;
   - receives framed bytes on each path, calls the core to dedup, and writes the
     surviving payloads back to `packetFlow`.
   - Path A (Wi-Fi) is a UDP socket bound to the Wi-Fi interface
     (`IP_BOUND_IF` / `NWParameters.requiredInterface`). Path B (cellular) is the
     proven userspace RNDIS uplink (the spike), exposing a send-frame/recv-frame
     interface.

3. **The Swift↔Go boundary is a narrow C-shared API** (cgo `//export`), not
   gomobile. Roughly:
   - `cc_session_new(session_id[16], capacity) -> handle`
   - `cc_outbound(handle, payload*, len, outbuf*, outcap) -> framed_len`
   - `cc_inbound(handle, wire*, len, path*, outbuf*, outcap) -> {payload_len, deliver}`
   - `cc_session_free(handle)`
   Memory ownership stays on the Swift side: Go writes into caller-provided
   buffers and holds no Swift pointers across calls; no Go pointer is retained by
   Swift (per ADR-0002's memory-ownership condition).

4. **Encryption is layered below framing, separately.** The payload is encrypted
   by a reviewed construction (no invented crypto; tracked in `TASKS.md`) before
   `Outbound` frames it and after `Inbound` delivers it. This ADR does not choose
   the construction.

## Alternatives Considered

* **gomobile bind.** Generates Objective-C bindings; heavier, less control over
  the per-packet hot path and memory ownership than a hand-written C-shared API.
* **Reimplement dedup/framing in Swift.** Loses the reviewed, race-tested Go core
  and the gateway-shared wire understanding; duplicates the brain in two
  languages.
* **One framed-bytes buffer per path with per-path tags.** Rejected for the data
  plane: dedup is by identity, so identical bytes over every path is simpler and
  halves framing work. Path attribution, if needed, is recorded by the receiver.

## Rationale

The transport brain is exactly the part that benefits from Go's testability and
from sharing wire understanding with the gateway; the parts that must be Swift
(NE lifecycle, `packetFlow`, socket/interface binding) stay Swift. A narrow
C-shared boundary keeps memory ownership analysable. Sending identical framed
bytes over every path matches the proven model and the gateway's identity-based
dedup.

## Consequences

* cgo build complexity in the extension; the FFI cost must be amortised by
  batching packets per call rather than one cgo crossing per packet.
* The RNDIS uplink must run inside the extension's sandbox with
  `com.apple.security.device.usb`; re-confirming that claim is a gating task
  (`TASKS.md`) and blocks the App Store route.
* The core's dedup window capacity and the sequence-number rollover behaviour
  (`protocol.PacketNumber == 0` invalid) become production concerns at real
  packet rates.

## Conditions for Revisiting

Revisit if per-packet cgo cost is too high even batched (move dedup into Swift),
if the NetworkExtension sandbox cannot host the userspace USB claim (reconsider
the app/extension split or the App Store route), or if encryption requirements
change the framing boundary.
