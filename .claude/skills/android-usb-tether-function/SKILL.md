---
name: android-usb-tether-function
description: Switch and inspect an Android phone's USB tether gadget function (RNDIS vs NCM) over adb to control whether macOS can natively claim it as a network interface. Use when working on gemina uplink acquisition, deciding if a given phone tethers over NCM (macOS-native, no kext) or RNDIS (needs a driver), driving `svc usb setFunctions`/`dumpsys tethering`, or building the pre-purchase device compatibility matrix. Reach for this whenever the task involves Android USB tethering, `adb` USB gadget control, NCM/RNDIS, or "make the phone show up as a usable NIC on the Mac".
---

# Controlling Android USB tether function (RNDIS ↔ NCM) over adb

The gemina second WAN needs the phone to appear on macOS as a **usable
network interface with cellular upstream**. Whether that's possible without a
custom macOS driver depends entirely on which **USB gadget function** the phone
tethers over:

- **NCM** (CDC Network Control Model) → macOS has a **native** host driver
  (`AppleUSBNCMControl`/`AppleUSBNCMData`). The phone enumerates as an `enX` NIC
  with **no kext and no SIP changes**. This is the low-friction dream path.
- **RNDIS** → macOS ships **no** host driver. The function sits unclaimed; no
  NIC appears. Needs a userspace RNDIS data plane (hard) to consume.

See [[macos-usb-tether-detection]] for the macOS-side class signatures.

## Prerequisites

- `adb` (`brew install --cask android-platform-tools`).
- Phone: Developer options → **USB debugging** on, and authorise the Mac
  (tap "Allow", tick "Always allow from this computer"). Confirm with
  `adb devices` showing `<serial>  device`.

## Inspect first (non-destructive)

```bash
adb shell svc usb getFunctions            # current function(s), e.g. "rndis"
adb shell getprop sys.usb.state           # what the kernel actually applied
adb shell dumpsys tethering | grep -iE "mUsbTetheringFunction|tetherable.*Regexs"
```

Read these carefully — they tell you the whole story:

- `mUsbTetheringFunction: RNDIS` means the **USB tethering toggle is hard-wired to
  RNDIS** by a resource overlay (`config_tether_usb_functions`). This is **root-
  locked**: you cannot change it with `device_config`, `cmd overlay`, or settings.
- `tetherableUsbRegexs: [usb\d, rndis\d]` vs separate `tetherableNcmRegexs:
  [ncm\d]` — Android has two distinct tether *types*. The everyday USB-tethering
  toggle is `TETHERING_USB` (forces `mUsbTetheringFunction`). `TETHERING_NCM` is a
  separate type with no Settings UI and no shell command, gated behind the
  privileged `TETHER_PRIVILEGED` permission.

## Switch the raw gadget function

```bash
adb shell svc usb setFunctions ncm        # or rndis,adb / ncm,adb to keep adb
```

- Changing functions **resets the USB bus**, so the adb session drops mid-command
  (you'll see `exit 137` or `no devices/emulators found`). Follow every switch
  with `adb wait-for-device`, then re-read `getprop sys.usb.state`.
- Keep `adb` in the set (`ncm,adb`) to retain control after the reset.
- `svc usb getGadgetHalVersion` may return `unknown` even on phones that **do**
  support `ncm` — don't trust it. Verify empirically with `sys.usb.state`.
- `svc usb setScreenUnlockedFunctions <fn>` sets the function applied on unlock;
  the tethering service uses this to pin RNDIS. **Always clear what you set**
  (`setScreenUnlockedFunctions ""`) when done, or the phone keeps defaulting to
  your test function and breaks normal USB use.

## The catch that wastes hours

Setting `svc usb setFunctions ncm` exposes the NCM gadget and **macOS will claim
it** (a new `enX` appears, `ioreg` shows `CDC Network Control Model (NCM) …
matched, active`). But the phone brings the interface up with **only a
link-local address** — macOS gets a `169.254.x.x` (APIPA) and **no internet**,
because **NAT/DHCP is only run by the tethering service**, and the tethering
service only NATs the function it's pinned to (RNDIS → `rndis0`). The raw `ncm`
function is `usb0` on the phone with no IPv4 and no NAT.

So on an RNDIS-pinned phone (OnePlus/OxygenOS confirmed, Android 16) you hit a
wall:

- USB tethering ON → NATs `rndis0` (works: it gets a private RFC1918 `/24` and
  routes to cellular), but macOS can't see RNDIS.
- `svc usb setFunctions ncm` → macOS sees the NIC, but the phone won't NAT it.
- The two cannot be combined without **root** (to flip the overlay to NCM) or a
  **privileged `TETHERING_NCM` trigger**.

## Decision table for the compatibility matrix

| Phone tethers over | macOS driver? | Phone NAT? | Result |
|---|---|---|---|
| NCM by default (Pixel / AOSP 14+, some OEMs) | native | yes | **Works, zero install** — recommend this tier |
| RNDIS, pinned (OnePlus/OxygenOS, many OEMs) | none | only over RNDIS | Needs userspace RNDIS data plane, **or** root to switch to NCM |

This is exactly the input the pre-purchase compatibility check needs: probe the
phone's tether function before purchase and route the user to the supported tier.

## Always restore the phone when finished

```bash
adb shell svc usb setScreenUnlockedFunctions ""
adb shell svc usb setFunctions ""          # back to charging+adb default
```

Tell the user to re-toggle USB tethering in the phone UI if they want their
normal tether back — your experiments turn it off (`tethered_config_state=0`).
