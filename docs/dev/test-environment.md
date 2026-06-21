# Test environment: bonding the uplinks without losing the Mac

This is how we exercise the dual-path / failover work on a single MacBook Pro
without breaking its network — and specifically without cutting the operator's
own link to the outside world (including the Claude session driving the work).

## Interface roles (fixed for the duration of a test cycle)

| Interface | Role | Rule |
| --- | --- | --- |
| Cabled Ethernet (Thunderbolt preferred over the dock LAN) | **Management / lifeline** | The app never touches it. Pinned first in the network service order so it owns the default route. |
| Wi-Fi (`en0`) | Test uplink A | Bonded, failed over, and deliberately broken during path-loss tests. |
| Phone USB-RNDIS | Test uplink B | Bonded, failed over, and deliberately broken. |

Keep the management cable connected while testing. It is the channel that
survives every path-loss experiment, so the operator stays reachable even when
both test uplinks are intentionally down.

## Why this is safe, not just hopeful

* **Pinned service order.** With the cabled service first
  (`scripts/restore-network.sh`), Wi-Fi or the phone appearing and disappearing
  can never steal the default route.
* **Userspace RNDIS cannot hijack routing.** The phone is driven in-process and
  never registers as a macOS network service, so — unlike a real NIC — it is
  structurally incapable of becoming the default route or changing DNS.
* **Stage 1 changes no global routing.** Per-socket binding sends probes out a
  chosen source interface without touching the routing table, so at this stage
  nothing global is mutated. The cable is belt-and-braces.
* **Tunnel stage stays scoped.** When the NEPacketTunnelProvider arrives, it must
  route only the test destination (the gateway) via included routes — never the
  default route — and must exclude the management subnet. The experiment is
  opt-in per destination; everything else, including the Claude session, stays on
  the cable.

## Procedure

1. **Snapshot** the baseline before touching anything:
   `scripts/snapshot-network.sh` (writes to the git-ignored `.netcheck/`).
2. **Pin** the management cable first: `scripts/restore-network.sh` (auto-detects
   a wired service, or pass the exact service name). Confirm the primary
   interface it prints is the cable.
3. **Run the test** (socket-bind probes today; scoped tunnel later).
4. **Restore** at any sign of trouble, or at the end:
   `scripts/restore-network.sh`. This reasserts the service order and prints the
   resulting default path so you can confirm the lifeline is intact.

## Recovery

If a test leaves connectivity wrong, `scripts/restore-network.sh` is the single
command to reassert a safe state. It is conservative by design: today it only
reorders services and reports the default path (Stage 1 has no global route or
tunnel to tear down). When the tunnel lands, its disable step is added there so
recovery stays one command.
