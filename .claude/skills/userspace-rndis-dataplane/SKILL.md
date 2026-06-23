---
name: userspace-rndis-dataplane
description: Build, run, and extend the continuity-vpn userspace RNDIS data plane that drives an Android phone's USB tether from an unprivileged macOS process (no kext, no SIP, no DriverKit) — libusb/IOUSBHost, RNDIS message framing, packet-filter OIDs, DHCP-over-RNDIS. Use for any work in research/usb-rndis-spike, moving L2 frames over the phone tether, NEPacketTunnelProvider integration of the RNDIS link, or debugging "phone tether won't pass packets on macOS". Reach for this whenever the task touches RNDIS, libusb bulk endpoints, REMOTE_NDIS_PACKET_MSG, or feeding the Android uplink into the bonding stack.
---

# Userspace RNDIS data plane (continuity-vpn)

macOS ships no RNDIS host driver, but the Android RNDIS function is left
**unclaimed** on the bus — so an ordinary process can claim it and speak RNDIS
itself, giving the phone's cellular tether to the app **without a kext, without
DriverKit, without disabling SIP**. This is the Route-B uplink: it works on
*every* Android (RNDIS is universal), unlike NCM which is device-dependent (see
[[ncm-tether-lower-friction-than-rndis]] and skill `android-usb-tether-function`).

Code lives in `research/usb-rndis-spike/`, layered so the pure logic is testable:
- `rndis_lib.{c,h}` — **pure** framing: RNDIS wrap/unwrap, DHCP build/parse,
  ARP, UDP/IP, the CVP1 probe, checksums. Hardware-independent.
- `rndis_lib_test.c` — unit tests for all of the above (`make test`, no phone).
- `rndis_usb.{c,h}` — libusb I/O: claim by RNDIS class, INITIALIZE, packet
  filter, bulk send/recv. Shared by every driver.
- `rndis_probe.c` — control plane (claim + INITIALIZE; proven).
- `rndis_dataplane.c` — data plane (DHCP DISCOVER/OFFER round-trip; proven).
- `rndis_egress.c` — real UDP egress to the deployed gateway over cellular
  (lease + ARP + CVP1 probes; proven end-to-end 2026-06-23).

**Keep new logic testable**: put any new pure framing/parsing in `rndis_lib.c`
with a test in `rndis_lib_test.c`; put USB I/O in `rndis_usb.c`; keep the driver
`main()`s thin. Tests use synthetic, non-identifying data (locally-administered
test MACs, TEST-NET/RFC1918 addresses as C byte arrays, never dotted-quad
strings — the redaction hook blocks those) to stay within the invariant.

## Proving egress end-to-end (the verification recipe)

`rndis_egress.c` takes the gateway address from the environment so no server IP
is ever compiled in:

```bash
CONTINUITY_GATEWAY_IP=<ip> CONTINUITY_GATEWAY_PORT=51820 make run-egress
```

To confirm arrival (the bytes leave only via the USB bulk pipe, so any arrival
proves cellular egress): on the gateway host, watch the container logs for
`first-copy`/`duplicate` decisions tagged `android-usb-tether`, and run
`sudo tcpdump -ni any 'udp and port 51820'` — the source will be a **cellular
carrier public IP**, not the Mac's LAN. Mask the source octets when reporting and
delete any capture file afterwards (it holds the carrier IP). The gateway runs in
Docker, so its own logs see a SNAT'd source — use the host tcpdump for the source
proof and the container logs for the decode/dedup proof.

## Build & run

```bash
brew install libusb                      # one-off
cd research/usb-rndis-spike
make rndis_dataplane && ./rndis_dataplane   # or: make run-dataplane
```

The phone must have **USB tethering ON** (so the RNDIS function *and* the phone's
tether DHCP/NAT server are running). Confirm the function is present first:

```bash
adb shell svc usb getFunctions           # expect: rndis
ioreg -r -c IOUSBHostInterface -l -w 0 | grep -E '^\+-o RNDIS'
```

Discover the device by **RNDIS interface class (0xE0)**, never a fixed VID/PID —
the product id changes when adb is in the composite (seen: 0x2766 vs 0x276A).

## Hard rules (do not violate)

- **Provenance / clean-room.** All RNDIS code is authored from the public
  MS-RNDIS layout and the DHCP/BOOTP RFCs. **Never** read or paste from Linux's
  GPL `drivers/net/usb/rndis_host.c` or any GPL DHCP client into product or spike
  dirs. Run the `provenance-licence-reviewer` agent before merging.
- **Redaction.** Never print or store a MAC, IP, or serial — not even in a
  comment (the `redaction-guard.sh` hook blocks dotted-quads in source, and
  `smoke.sh` greps runtime output). Use a locally-administered random host MAC;
  report success as "lease offered (address redacted)".
- **Safety.** The process never registers a macOS network service, so it cannot
  steal the default route or DNS. Keep it that way. Use the cabled management
  lifeline and `scripts/snapshot-network.sh` / `restore-network.sh` during tests
  (`docs/dev/test-environment.md`). Phone changes (USB tethering, `svc usb`) are
  reversible UI/adb toggles — restore them when done.

## RNDIS protocol reference (little-endian on the wire)

Control messages go over EP0: host→device `bmRequestType=0x21`
`SEND_ENCAPSULATED_COMMAND (0x00)`; device→host `0xA1`
`GET_ENCAPSULATED_RESPONSE (0x01)`, both to the **control** interface number.

| Message | Type | Completion | Notes |
|---|---|---|---|
| INITIALIZE | `0x00000002` | `0x80000002` | 24-byte msg; status at resp+12, medium at +28, max-xfer at +36 |
| SET (OID) | `0x00000005` | `0x80000005` | 28-byte header + info buffer; status at resp+12 |
| QUERY (OID) | `0x00000004` | `0x80000004` | for non-identifying caps only |
| PACKET (data) | `0x00000001` | — | over **bulk** endpoints, not EP0 |

Bring the link up after INITIALIZE: `SET OID_GEN_CURRENT_PACKET_FILTER`
(`0x0001010E`) with `DIRECTED(0x1)|MULTICAST(0x2)|BROADCAST(0x10)`. Without a
non-zero filter the device drops all RX.

`REMOTE_NDIS_PACKET_MSG` (data, on bulk IN/OUT), 44-byte header then the raw
Ethernet frame:

```
+0  MessageType   = 0x00000001
+4  MessageLength = 44 + frameLen
+8  DataOffset    = 36   (offset from byte 8 to the frame; 44-8)
+12 DataLength    = frameLen
+16..+43 OOB / per-packet-info / VcHandle / reserved = 0
+44 <Ethernet frame>
```

On RX, validate `MessageType==0x1`, then read the frame at `8 + DataOffset` for
`DataLength` bytes. Other message types on bulk IN are ignored. Watch for
bulk-OUT ZLP termination if a frame ever lands on a wMaxPacketSize multiple.

## The roadmap this unblocks

1. ✅ Control: claim + INITIALIZE (`rndis_probe.c`).
2. ✅ Data plane: packet filter + DHCP round-trip (`rndis_dataplane.c`).
3. ✅ Hold the lease + ARP the phone gateway; send a real CVP1 UDP probe to the
   oracle gateway over cellular, verified in gateway logs (`rndis_egress.c`).
4. ▢ Feed RX frames into `NEPacketTunnelProvider` and TX from it, so the
   bonding/failover layer routes per-flow.
5. ▢ Re-confirm the USB claim inside the App Sandbox with
   `com.apple.security.device.usb`.
6. ▢ Simultaneous Wi-Fi + cellular dual-path through the gateway (Stage-1 gate).
