# Gemina VPN — press kit

> **Status: pre-release.** The dual-path transport is proven end-to-end; encryption and
> the shipping macOS app are in active development. This kit contains **no invented
> metrics, users, testimonials, benchmarks or pricing**. British English throughout.
> Replace every `<placeholder>` before use. See also
> [`launch-plan.md`](launch-plan.md), [`seo.md`](seo.md), [`video-script.md`](video-script.md).

---

## Short description (≤ 50 words)

> Gemina VPN is an open-source macOS reliability tool. It sends your traffic over
> Wi-Fi **and** your Android phone's cellular (USB tethering) at once, to a gateway that
> keeps the first copy of each packet — so if one link drops mid-session, your call, SSH
> or VPN doesn't notice. Pre-release.

## Long description (≈ 120 words)

> Gemina VPN keeps your Mac's interactive sessions alive when a network blips. Instead
> of reconnecting after a drop, it sends every packet over **two independent links at the
> same time** — your Wi-Fi and an Android phone's cellular link via USB tethering — to a
> single gateway. The gateway delivers the first copy of each packet and discards the
> duplicate, so if one link fails mid-call the other is already carrying the same packets
> and the session simply continues, with no reconnect.
>
> It is built for **reliability, not speed**: it does not combine the two links'
> bandwidth. It works with **any Android, no root**, and needs **no kernel extension or
> SIP change on the Mac**. It is **open source and self-hostable**, with an optional paid
> hosted gateway planned. **Pre-release:** the dual-path transport is proven; encryption
> and the shipping app are in development.

---

## Key facts

| | |
|---|---|
| **Name** | Gemina VPN |
| **Category** | macOS reliability / networking utility (seamless internet failover) |
| **Platform** | macOS (Apple Silicon primary); requires any Android phone with USB tethering |
| **What it does** | Duplicates traffic over Wi-Fi + Android USB-tether cellular; gateway dedups to one delivery; survives either link dropping |
| **Reliability, not speed** | Does **not** aggregate bandwidth; spends the second link on certainty |
| **Privilege model** | Userspace driver — no root on the phone, no kernel extension, no DriverKit, no SIP changes on the Mac |
| **Open source** | Client + shared core: Apache-2.0 · Gateway: AGPL-3.0 (open-core) |
| **Self-host** | Gateway runs as one container on one UDP port (51820) — no database, no accounts |
| **Hosted option** | Optional paid hosted gateway planned; billed via Stripe; **pricing to be announced** |
| **Status** | **Pre-release** — dual-path transport proven end-to-end; encryption + shipping app in development |
| **Repository** | `https://github.com/joanmarcriera/gemina` |
| **Website** | `<site>` |
| **Waitlist** | `<waitlist>` |
| **Licence texts** | `LICENSES/AGPL-3.0.txt`, `LICENSES/Apache-2.0.txt` |

---

## How it works (explainer)

The client sends every packet **twice** — once over each available uplink (Wi-Fi and the
Android phone's USB-tethered cellular link) — tagging both copies with the same identity.
Both copies travel over genuinely independent networks to a single gateway. The gateway
keeps a short window of recently seen identities, forwards the **first** copy of each
packet, and drops any later duplicate. When a link fails, its copies simply stop arriving;
the surviving link's copies keep the session going, with no reconnection.

```
 macOS client ──copy A──▶ Wi-Fi uplink ─────────▶ ┐
              ──copy B──▶ Android USB tether ────▶ ├─▶ Gateway (UDP 51820)
                          (cellular uplink)         ┘     │
                                                          ▼
                                          first copy delivered, duplicate discarded
                                                          │
                                                          ▼
                                                       Internet
```

The cost is roughly double the data on the protected traffic; the benefit is continuity.
Because the two paths are independent WANs, a blip on either — a tunnel, a lift, a flaky
access point — does not interrupt the flow.

**The unusual engineering result:** the Mac drives the phone's USB tether from an
unprivileged userspace process (a userspace RNDIS data plane), so there is no root on the
phone, no kernel extension, no DriverKit and no SIP change on the Mac. This works on any
Android with USB tethering; some phones (Pixel, AOSP 14+) are also recognised natively by
macOS via CDC-NCM.

**What is proven, and what is not.** The dual-path transport is proven end-to-end: the
same logical packet sent over Wi-Fi and over the phone's cellular link both reach a
deployed gateway from two distinct public WAN addresses, are deduplicated to a single
delivery, and either path can drop without ending the session. **In development:** the
shipping macOS app data path (`NEPacketTunnelProvider`), wiring the already-built and
tested encrypted core into the app, and accounts/payments. Today's public proof carries
probe packets, not your encrypted IP traffic.

---

## Founder / maintainer

> **`<maintainer name>`** — `<one-line bio placeholder: who you are, why you built it>`.
>
> Gemina VPN is an open-source project maintained by `<maintainer name>`. It is built
> in the open; the repository, the architectural decisions (`DECISIONS.md`), the current
> state (`PROJECT_STATE.md`) and the task list (`TASKS.md`) are all public.
>
> Contact: `<contact email / preferred channel placeholder>`

---

## FAQ pointer

The canonical, always-current FAQ lives on the website's *Frequently asked questions*
section (see `website/index.html`) and in [`README.md`](../../README.md). Common
questions — *Does it make my connection faster? (No — it's reliability, not speed.) Do I
need root or a kernel extension? (No.) Which Android phones work? (Any with USB
tethering; run `geminactl preflight`.) Can I self-host? (Yes, free.) What does the
hosted tier cost? (To be announced.) Is it released yet? (No — pre-release.)* — are
answered there and must stay consistent with this kit.

---

## Logos, screenshots & diagrams

> **Placeholders — add real assets before distribution.** All visual assets must follow
> the redaction rules in [`video-script.md`](video-script.md) §4 (no IPs, MACs, serials,
> carrier names, hostnames or personal detail).

- [ ] **Logo / wordmark** — `<path: website/og-image.png or a dedicated logo asset>`
      (provide light + dark variants; SVG preferred).
- [ ] **Social / OG image** — 1200×630, the tagline over the link-merge concept
      (`website/og-image.png`).
- [ ] **Architecture diagram** — the "duplicate, race, dedup" flow (the README/site
      diagram, exported cleanly).
- [ ] **Screenshot: `geminactl preflight`** — terminal showing a plain verdict
      (redacted, neutral prompt).
- [ ] **Screenshot: gateway telemetry** — redacted JSON logs (`first-copy`/`duplicate`,
      `wi-fi`/`android-usb-tether`) and the Prometheus `/metrics` page.
- [ ] **Screenshot: live failover** — the `wi-fi` series flatlining while
      `android-usb-tether` keeps climbing (read live off the gateway).
- [ ] **Demo video** — the explainer per [`video-script.md`](video-script.md), plus the
      30–45s social cut.

Do **not** ship a staged/mocked app UI screenshot: the shipping macOS app is in
development.

---

## Boilerplate (copy-paste)

> Gemina VPN is an open-source macOS reliability tool that keeps your calls, SSH and
> VPN sessions alive when one network link drops, by sending your traffic over Wi-Fi and
> an Android phone's cellular link at the same time. It is built for reliability, not
> speed; works with any Android, no root, and no kernel extension or SIP change on the
> Mac; and is self-hostable, with an optional paid hosted gateway planned. Pre-release.
> `<repo>`

---

## Contact

- **General / press:** `<contact email placeholder>`
- **Security:** see `SECURITY.md` in the repository.
- **Code & issues:** `https://github.com/joanmarcriera/gemina`
- **Website / waitlist:** `<site>` · `<waitlist>`
