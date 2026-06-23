# Userspace RNDIS viability spike

**Question this answers:** can the shipping macOS app use an Android phone's USB
tethering as a second uplink *from userspace* — no kernel extension, no
DriverKit entitlement from Apple, no weakening of System Integrity Protection?

**Answer (proven 2026-06-21 on the target Mac): yes.**

## Background

macOS ships no host driver for RNDIS, the USB network function Android exposes
when "USB tethering" is on. So the phone never becomes a `enX` NIC and never
appears in `networksetup` / `darwin-evidence`. That is a *missing host driver*,
not a missing device. `ioreg` confirmed the OnePlus 12R (KALAMA) presents, in
its active configuration, a standard RNDIS function:

| Interface | `bInterfaceClass` | Role |
| --- | --- | --- |
| RNDIS Communications Control | `0xE0` / sub `1` / proto `3` | control channel |
| RNDIS Ethernet Data | `0x0A` (CDC-Data) | bulk data pipe |
| ADB Interface | `0xFF` | ignored |

The interfaces are `registered, matched, active` but **unclaimed** by any
networking driver — therefore openable by an unprivileged userspace process.

## What the spike does

`rndis_probe.c` (libusb, the same IOKit/IOUSBHost path the app would link):

1. Opens the device by VID/PID (`0x22D9:0x2766`).
2. Locates the RNDIS control + data interfaces and bulk endpoints from the
   descriptors (no hard-coded endpoint numbers).
3. Claims both interfaces.
4. Completes the RNDIS `REMOTE_NDIS_INITIALIZE` handshake over the control
   channel and reads back the device's reported medium and max transfer size.

It sends only `INITIALIZE`. It does **not** read the permanent MAC, does **not**
move user traffic, and leaves no persistent state on the phone.

## Result (redacted)

```
PASS open: claimed a handle to OnePlus RNDIS composite device
PASS descriptors: ctrl_if=0 data_if=1 bulk_in=0x81 bulk_out=0x01
PASS claim: userspace owns both RNDIS interfaces (SIP enabled)
PASS init-send: SEND_ENCAPSULATED_COMMAND (24 bytes) accepted
PASS init-resp: REMOTE_NDIS_INITIALIZE_CMPLT status=0x00000000 medium=0(0=802.3) device_max_transfer=23700 bytes
RESULT: userspace RNDIS handshake COMPLETE
```

`status=0x0` = success; `medium=0` = 802.3 Ethernet; the phone advertised a
23700-byte max transfer. SIP was enabled (`csrutil status: enabled`).

## Run it

```bash
brew install libusb        # one-off
make run                   # phone connected, USB tethering enabled
```

If the phone is absent or not in tethering mode the probe prints `FAIL open`.

## What this proves — and what it does not

**Proven:** an ordinary process, under SIP, can take exclusive control of the
Android RNDIS function and drive its control plane. The hard unknown that would
have killed the in-app approach (can we even open the interface?) is retired.

**Still to confirm before shipping:**

- **Data plane — PROVEN 2026-06-23.** `rndis_dataplane.c` brings the link up
  (`SET OID_GEN_CURRENT_PACKET_FILTER`), sends a DHCP DISCOVER framed in
  `REMOTE_NDIS_PACKET_MSG` on bulk OUT, and reads the phone tether's DHCP OFFER
  back on bulk IN — a full L2 round-trip. Verified live against the OnePlus 12R
  RNDIS tether, reliably across repeated runs, redaction-clean. **L2 frames move
  both ways over the cellular tether from an unprivileged process** (no kext, no
  SIP, no root). `make rndis_dataplane && ./rndis_dataplane` with USB tethering
  on. See skill `userspace-rndis-dataplane`.
- **App Sandbox entitlement.** Both probes ran un-sandboxed. A Developer-ID
  notarised app needs the `com.apple.security.device.usb` entitlement; verify
  the claim still succeeds inside the app's sandbox.
- **Egress into the stack.** Feed received frames into an
  `NEPacketTunnelProvider` so the bonding/failover layer can route per-flow.

## Provenance

`rndis_probe.c` is authored clean-room from the public Remote NDIS (MS-RNDIS)
protocol constants. It is **not** derived from Linux's GPL
`drivers/net/usb/rndis_host.c`. Keep it that way: do not paste GPL driver source
into this file or the product RNDIS implementation.
