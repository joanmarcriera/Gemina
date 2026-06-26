# Compatibility catalogue: data + privacy design

The public catalogue lives in [`COMPATIBILITY.md`](../../COMPATIBILITY.md). This
note records how it works and why it is built the way it is.

## Goal

Let prospective users check "will my Mac + my phone work?" before they invest
time, and give us a real picture of which devices people use — **without**
collecting personal data or running silent telemetry.

## Principles

1. **No silent collection.** The product never phones home with device data.
   Every catalogue entry is something a person chose to submit.
2. **Redacted by construction.** `geminactl preflight -share` prints only
   coarse tokens already in the compatibility report: the verdict, the macOS
   version, and the tether mode (`native-ncm` / `app-driver-rndis` / `none`). It
   contains no IP, MAC, serial, CID/IMEI or bsd interface name. A unit test
   asserts the share output carries none of those.
3. **The user owns the device label.** We do not auto-extract a phone model
   string (those can carry serials/CIDs — e.g. the raw USB product name we see
   is `KALAMA-MTP_CID:…_SN:…`). Instead the share template invites the user to
   type their own model when they submit. They decide what to share.
4. **Public and reviewable.** Reports arrive as PRs/issues against
   `COMPATIBILITY.md`, so the data is open, deduplicated and curated in the open.

## How it could grow (future, opt-in only)

If we ever add a one-tap "submit my report" path, it must stay **opt-in**, show
the exact redacted payload before sending, and post the same coarse tokens — no
new fields without re-reviewing this contract and the redaction tests. The
gateway's metrics (`observability/METRICS.md`) remain aggregate and redacted and
are not tied to a device.

## Relationship to the preflight verdict

The catalogue's "Status" column mirrors the preflight verdict semantics:

- **native (✅):** macOS claims the phone/connection as a NIC with no app driver
  (iPhone Personal Hotspot over USB; Pixel/AOSP NCM; any wired second line).
- **app-driver RNDIS (⚙️):** an Android RNDIS tether function is present; the
  app's bundled userspace driver makes it usable (any Android).
- **none / unsupported (❌):** no usable second connection detected.
