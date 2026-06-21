# Stage 1 Backlog

The active, ordered Stage 1 task list now lives in the root `TASKS.md` (single
source of truth). This file keeps only the fixed objective and the acceptance
evidence that defines "done"; do not duplicate the task checklist here.

## Objective

Prove that UDP socket A explicitly leaves through Wi-Fi, UDP socket B explicitly
leaves through Android USB tethering, both reach the same Hetzner process, one
logical packet is delivered once, and either path can disappear without ending
the logical session.

## Stage 1 acceptance evidence

* Packet captures showing each path independently reaches the gateway.
* Test logs showing duplicate suppression.
* Failure test showing either path can disappear without ending the logical
  session.
* Updated threat model.
* Updated ADRs if transport assumptions change.
