# Stage 1 Backlog

Stage 1 must not begin until Stage 0 exit criteria have been reviewed.

## Objective

Prove that UDP socket A explicitly leaves through Wi-Fi, UDP socket B explicitly leaves through Android USB tethering, both reach the same Hetzner process, one logical packet is delivered once, and either path can disappear without ending the logical session.

## Candidate Tasks

* Enumerate available macOS interfaces without hard-coding interface names.
* Identify Wi-Fi and Android USB tethering candidates using system APIs and observable properties.
* Bind source addresses or otherwise prove per-path UDP egress.
* Send duplicated probes to one gateway process.
* Deduplicate server-side probe identifiers.
* Capture packet evidence for both paths.
* Simulate loss of each path and measure logical-session survival.
* Record metrics and logs without secrets.

## Stage 1 Acceptance Evidence

* Packet captures showing each path independently reaches the gateway.
* Test logs showing duplicate suppression.
* Failure test showing either path can disappear without ending the logical session.
* Updated threat model.
* Updated ADRs if transport assumptions change.
