---
name: macos-usb-tether-detection
description: Detect USB tether/network functions on macOS (Android RNDIS, CDC-NCM/ECM, plain USB NICs) by querying IORegistry and keying on USB interface class — not vendor strings. Use whenever working on the gemina darwin-evidence diagnostic, parsing `ioreg` output, deciding whether a phone's USB tether is usable on this Mac, distinguishing RNDIS from NCM, or adding device-level evidence. Reach for this any time the task involves USB networking, tethering, ioreg parsing, or "why doesn't macOS see my phone as a network interface".
---

# Detecting USB tether functions on macOS

A USB-tethered phone or a USB-Ethernet dock exposes one or more **USB
interfaces**, each tagged with a numeric class/subclass/protocol triple. macOS
only turns that into a usable `enX` NIC if it has a **host driver** that claims
the function. The whole gemina uplink problem turns on this gap:

- **Android RNDIS tethering → no macOS NIC.** macOS ships no RNDIS host driver,
  so the function sits unclaimed on the bus. The phone is connected, tethering is
  on, yet `ifconfig`/the BSD interface list show nothing.
- **CDC-NCM and CDC-ECM → usable NIC for free.** macOS *does* have native host
  drivers (`AppleUSBNCMControl`/`AppleUSBNCMData`, `AppleUserECM…`). A function in
  one of these classes is auto-claimed into an `enX` with an IP.

So the right question is never "what vendor is this?" but "**what USB interface
class is the function, and does macOS have a driver for it?**"

## The signatures (key on these, never on product/vendor strings)

| Function | bInterfaceClass | Subclass | Protocol | macOS host driver? |
|---|---|---|---|---|
| Android **RNDIS** control | `224` (0xE0) | `1` | `3` | **No** — never becomes a NIC |
| RNDIS data | `10` (0x0A) | `0` | `0` | (data half of the pair) |
| **CDC-NCM** control | `2` (0x02) | `13` (0x0D) | `0` | **Yes** (`AppleUSBNCMControl`) |
| **CDC-ECM** control | `2` (0x02) | `6` (0x06) | `0` | **Yes** |
| CDC data | `10` (0x0A) | `0` | varies | (data half) |
| Vendor USB-Ethernet (e.g. Realtek dock) | `255`/`2` | — | — | Yes (vendor/CDC driver) |
| USB hub | `9` (0x09) | — | — | n/a |

Why class beats vendor string: the OnePlus on the dev rig reports its product
name as `KALAMA-MTP_CID:…_SN:…` and vendor `OnePlus` — no "android" token at all.
A vendor-string matcher misses it; the `224/1/3` control signature does not, and
it works for any Android phone. Class `2` (Communications) cleanly separates a
real USB NIC (NCM/ECM) from an RNDIS tether (class `224`, "Wireless Controller").

## How to query the USB layer

An unclaimed RNDIS function never publishes an `IOEthernetInterface`, so the
ethernet-oriented query misses it. Query the **USB layer** instead:

```bash
ioreg -r -c IOUSBHostInterface -l
```

Each interface node carries `"bInterfaceClass" = N`, `"bInterfaceSubClass" = N`,
`"bInterfaceProtocol" = N` (unquoted integers) plus inherited
`"USB Product Name"`/`"USB Vendor Name"` strings (which you must NOT propagate —
they contain serials). Match on the integer class triple only.

For interfaces macOS *has* claimed (so you want the BSD name too), the
ethernet-layer query still applies:

```bash
ioreg -r -c IOEthernetInterface -l   # has "BSD Name" = "enX"
```

## Pitfall that will silently eat your data: bufio.Scanner

`bufio.Scanner` has a default 64 KiB line limit and, on exceeding it, **stops
without returning an error** (`Scan()` just returns false). A full `ioreg -l` USB
dump contains single property lines ~90 KiB long (IORegistry personality blobs).
A scanner-based block splitter therefore silently truncates the tree before later
nodes — you get an empty/partial result and no error. This bit the gemina
parser: it worked on the small `IOEthernetInterface` output and broke on the
large `IOUSBHostInterface` output.

**Fix:** split on raw newlines (`strings.Split(output, "\n")` in Go, or read the
whole stream) instead of `bufio.Scanner`. If you must use a scanner, raise the
buffer with `scanner.Buffer(buf, maxSize)` AND check `scanner.Err()`.

Always verify a USB-layer parser against **live hardware** with the real, full
`ioreg` output — a hand-trimmed fixture won't have the giant lines that expose
this bug. In gemina there is an env-gated live test
(`GEMINA_LIVE_USB=1`) for exactly this.

## Honesty rules (gemina never-fake-path-success)

A detected function is **not** a usable path. Present it truthfully:

- Present-on-USB is a **device-level** signal. With no host driver there is no
  NIC, no IP, no link — so it must NOT be promoted to a usable path `Candidate`.
- The `darwin-evidence` report carries it in a separate `device_functions`
  channel with `usable:false`/`host_driver_claimed:false`, plus a
  `tether-present-not-usable` issue that explains a missing candidate. This is
  the signal a pre-purchase compatibility check consumes.
- **Redaction is an invariant.** Emit coarse tokens only (e.g. `android-rndis`).
  Never let a serial, MAC, IPv4, product or vendor string reach the output; the
  smoke gate greps for MAC/IPv4 and fails the build if either appears.

## Where this lives in gemina

- `internal/platform/darwin/usb_functions.go` — `USBFunctionDeviceSource`, the
  class-keyed detector.
- `internal/platform/darwin/live_evidence.go` — `splitIORegistryBlocks` (the
  newline-split fix), interface-level sources, the shared evidence vocabulary.
- `internal/platform/darwin/evidence.go` — `EvidenceValue*` tokens; keep
  producer and consumer on the same constants.
- `internal/diagnostics/darwin_evidence.go` — the `device_functions` channel and
  `IssueCodeTetherPresentNotUsable`.

When adding a new function kind (e.g. NCM), extend the class table above, add the
token to the shared vocabulary, detect by class in `usb_functions.go`, and add a
live-hardware check before trusting it.
