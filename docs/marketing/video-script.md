# Gemina VPN — Video Walkthrough Script & Recording Guide

> **Honest pre-release framing.** Gemina VPN is a macOS *reliability* tool: it
> sends your traffic over **two uplinks at once** — your Wi-Fi *and* an Android
> phone's cellular link via USB tethering — and the gateway delivers the first
> copy of each packet while discarding the duplicate, so a link dropping
> mid-session does not break it. This is **reliability, not speed** (it does not
> add the two links' bandwidth together). It works with **any Android, no root**,
> and needs **no kernel extension or SIP change on the Mac**. It is **open source
> and self-hostable**, with an optional paid hosted gateway.
>
> **This video must show the REAL, proven things — not a fake app.** Today the
> dual-path transport is proven via command-line tools (`geminactl`, the
> userspace RNDIS spike) and the gateway's redacted logs and Prometheus metrics.
> Encryption and the shipping macOS app UI are still in development. Do **not**
> stage a polished app UI, and do **not** invent numbers — read every figure off
> the live gateway during the take.

---

## 1. Explainer / demo script (2–3 minutes)

Five beats: **Problem → Idea → Live proof → Open-source / self-host or hosted →
Call to action.** Times are cumulative targets; total ~2:40.

Voiceover is in British English. Captions/lower-thirds are short on-screen text.

---

### Scene 1 — The problem (0:00–0:20)

**On-screen action**
- Open on a real video-call or SSH session on the Mac (use a throwaway call or a
  local `ssh` to your own box — see redaction notes).
- Mid-sentence, the Wi-Fi menu-bar icon blips / shows "no connection"; the call
  freezes and the dreaded "Reconnecting…" spinner appears.

**Voiceover**
> "You know the moment. You walk into a lift, the Wi-Fi hiccups for a second —
> and your call drops. The session's gone, and you're dialling back in."

**Captions / lower-thirds**
- Opening title: **Gemina VPN**
- Lower-third: *One Wi-Fi blip. Whole session lost.*

---

### Scene 2 — The idea (0:20–0:45)

**On-screen action**
- Cut to a simple animation or the README's diagram: the Mac, two arrows
  ("copy A" over Wi-Fi, "copy B" over an Android phone's USB tether), both
  meeting one gateway, one clean line out to the internet.
- Highlight the gateway label: *first copy delivered, duplicate discarded.*

**Voiceover**
> "Gemina VPN takes a different approach. Instead of relying on one link, it
> sends every packet **twice** — once over your Wi-Fi, once over an Android
> phone's cellular tether. The gateway keeps the first copy of each packet and
> throws the duplicate away. If one link drops, the other is already carrying the
> same traffic — so the session never notices."

**Captions / lower-thirds**
- *Send it twice. Over two independent links.*
- *Gateway keeps the first copy, drops the duplicate.*
- Footnote caption (small): *Reliability, not extra speed.*

---

### Scene 3 — Live proof (0:45–1:55)

This is the heart of the video. Everything here is real and recordable today.

#### 3a — Preflight says "supported" (0:45–1:05)

**On-screen action**
- Terminal, large readable font. Phone connected, USB tethering on, Wi-Fi up.
- Run:
  ```
  geminactl preflight
  ```
- Show the plain verdict line (for example **supported**) and the one-line
  "what to change" guidance. Optionally flash `geminactl preflight -json`
  to show the machine-readable report the app/website will consume — but prefer
  the human verdict on camera.

**Voiceover**
> "First, does your gear even work? One command — `geminactl preflight` —
> checks this Mac and this Android together and gives a plain verdict. No
> guesswork."

**Captions / lower-thirds**
- *`geminactl preflight` → a plain "supported" verdict.*

> **Honesty note for the editor:** if, on your hardware, preflight reports the
> tether as *present but not yet usable* (macOS ships no RNDIS host driver, so the
> phone is not yet a system NIC), do not hide it — narrate it as the real
> pre-release state and lean on the userspace spike below for the live cellular
> path. Never fake a "supported" verdict.

#### 3b — Both paths delivering, with dedup (1:05–1:35)

**On-screen action**
- Split screen (or cut back and forth):
  - **Left:** the gateway's **redacted JSON logs** scrolling — lines showing
    `decision` = `first-copy` / `duplicate` and `path` = `wi-fi` /
    `android-usb-tether`. (Source address never reaches the handler; these logs
    are redaction-clean by design.)
  - **Right:** the gateway's Prometheus **/metrics** page in a browser or via
    `curl`, showing `gemina_packets_total{decision="first-copy",path="wi-fi"}`
    and `…{path="android-usb-tether"}` counters ticking up, plus the matching
    `duplicate` series. Optionally a Grafana panel of per-path delivery.
- Start the dual-path run (the proven path is the userspace spike):
  ```
  GEMINA_GATEWAY_IP=<gateway-ip> GEMINA_WIFI_IFACE=en0 make run-dualpath
  ```
  (run from `research/usb-rndis-spike/`), or the `geminactl probe` dual-path
  form for the machinery. Let the editor read the **actual** counts off the
  screen — do not pre-write numbers into the script.

**Voiceover**
> "Now watch the gateway. The same packet is arriving over **both** links. The
> log tags each one: first copy or duplicate, Wi-Fi or the phone's cellular path.
> And the metrics show it live — both paths delivering, and the duplicates being
> deduplicated down to a single delivery. That's the redundancy working."

**Captions / lower-thirds**
- *Gateway logs: `first-copy` / `duplicate`, per path.*
- *`/metrics`: `gemina_packets_total{decision,path}` — the failover signal.*
- *Both links delivering. Duplicates dropped.*

#### 3c — Live failover: unplug Wi-Fi, session continues (1:35–1:55)

**On-screen action**
- With the dual-path run still going, **physically disable Wi-Fi** (turn Wi-Fi
  off in the menu bar, or pull the dock Ethernet if that is your Wi-Fi stand-in).
- On the gateway side, show the `wi-fi` first-copy series **flatten to zero**
  while the `android-usb-tether` series **keeps climbing** and the active-session
  count holds. The call / SSH session in Scene 1's style keeps running.
- Then re-enable Wi-Fi and show the `wi-fi` series resume.

**Voiceover**
> "Here's the test that matters. I'll pull the Wi-Fi entirely. Watch — the Wi-Fi
> line goes flat, but the phone's cellular path keeps every packet flowing. The
> session just… continues. No reconnect. Plug Wi-Fi back in and it picks straight
> back up."

**Captions / lower-thirds**
- *Wi-Fi pulled → cellular carries the session. No reconnect.*
- *Survived path-loss, read straight off the gateway.*

> **Editor's note:** this is "the gateway's own view" of survival. Be honest in
> narration that what's proven today is the **transport** — packets surviving a
> link drop, observed at the gateway via the userspace spike and CLI — **not** a
> shipping VPN carrying your encrypted IP traffic through an app yet.

---

### Scene 4 — Open source, self-host or hosted (1:55–2:20)

**On-screen action**
- Show the GitHub repo page (README visible), then the one-liner to run your own
  gateway:
  ```
  docker run --rm --read-only -p 51820:51820/udp \
    -e GEMINA_GATEWAY_ADDR=:51820 \
    ghcr.io/example/gemina-gateway:latest
  ```
- Show a line in the client config pointing at `gateway.example.com:51820` to
  make the "gateway address is always configurable" point.

**Voiceover**
> "The whole thing is open source. The gateway is one small container on one UDP
> port — run it yourself, no database, no accounts. Or, when it's ready, point at
> our optional hosted gateway and skip running a server entirely. Same open-source
> client either way."

**Captions / lower-thirds**
- *Open source. Self-host the gateway — one container, one UDP port.*
- *Optional paid hosted gateway (pricing TBD).*
- Small caption: *Gateway AGPL-3.0 · client & core Apache-2.0.*

---

### Scene 5 — Call to action (2:20–2:40)

**On-screen action**
- End card: product name, the GitHub URL, and a waitlist link.
- Honest status badge on screen: **Pre-release — transport proven, app &
  encryption in progress.**

**Voiceover**
> "Gemina VPN is pre-release: the dual-path transport is proven, and the
> macOS app and encryption are in active development. If you've ever lost a call
> to a flaky network, star the repo and join the waitlist — and follow along as
> we ship."

**Captions / lower-thirds**
- *★ Star on GitHub · Join the waitlist*
- *github.com/joanmarcriera/gemina*
- *Pre-release. Honest about what's proven.*

---

## 2. Social cut (30–45 seconds, X / LinkedIn)

Same story, condensed. Vertical or square crop works; keep captions burned in
(many viewers watch muted).

| Time | On-screen | Voiceover / caption |
|---|---|---|
| 0:00–0:05 | Call freezes as Wi-Fi blips; "Reconnecting…" | "A Wi-Fi blip just killed your call." |
| 0:05–0:13 | Two-arrow diagram: Wi-Fi + phone tether → one gateway | "Gemina VPN sends every packet twice — Wi-Fi *and* your phone's cellular. The gateway keeps the first copy." |
| 0:13–0:28 | Gateway `/metrics` + JSON logs: both paths' `first-copy` counters climbing; then Wi-Fi pulled → `wi-fi` series flatlines, `android-usb-tether` keeps going | "Watch: both links delivering, duplicates dropped. Pull the Wi-Fi — the session keeps running on cellular. No reconnect." |
| 0:28–0:38 | GitHub repo + `docker run` one-liner | "Open source. Self-host the gateway in one container, or use the hosted one." |
| 0:38–0:45 | End card + status badge | "Pre-release: transport proven, app in progress. Star the repo, join the waitlist." |

**Burned-in caption track (short):**
`Wi-Fi blip = dropped call` → `Send every packet twice` → `Gateway keeps the
first copy` → `Pull Wi-Fi → session survives on cellular` → `Open source ·
self-host or hosted` → `Pre-release · join the waitlist`

---

## 3. Shot list / storyboard

| # | Shot | Duration | What's on screen | Capture source |
|---|---|---|---|---|
| 1 | Dropped call | 0:00–0:20 | Real call/SSH session freezes as Wi-Fi blips; "Reconnecting…" | QuickTime/OBS Mac screen capture |
| 2 | The idea | 0:20–0:45 | Two-arrow diagram (Wi-Fi + Android tether → one gateway → internet); "first copy delivered, duplicate discarded" | Animation / README mermaid diagram, screen-recorded or exported still |
| 3a | Preflight | 0:45–1:05 | Terminal: `geminactl preflight` → plain verdict + one fix; optional `-json` flash | OBS terminal capture (large font) |
| 3b | Both paths + dedup | 1:05–1:35 | Split: gateway JSON logs (`first-copy`/`duplicate`, `wi-fi`/`android-usb-tether`) **and** `/metrics` counters climbing; optional Grafana panel | SSH to gateway (`ssh oracle`) in terminal; browser/`curl` for `/metrics`; `make run-dualpath` driving traffic |
| 3c | Live failover | 1:35–1:55 | Wi-Fi turned off → `wi-fi` series flatlines, `android-usb-tether` keeps climbing, sessions hold; Wi-Fi back on → resumes | Same gateway capture + Mac menu-bar Wi-Fi toggle on camera |
| 4 | Open source / deploy | 1:55–2:20 | GitHub repo; `docker run` gateway one-liner; client pointed at `gateway.example.com:51820` | Browser + terminal capture |
| 5 | Call to action | 2:20–2:40 | End card: name, GitHub URL, waitlist, pre-release status badge | Static end card (Keynote/Figma export) |

**Capture sources at a glance**
- **Mac screen / terminal / browser:** QuickTime Player or OBS.
- **Gateway side (logs + `/metrics`):** an SSH session into the deployed gateway
  (`ssh oracle`) shown in a terminal; the `/metrics` page in a browser or `curl`.
- **Phone (optional):** only if you want to show "USB tethering on" — capture the
  Android screen via `scrcpy` or a phone-mirroring tool, or just show the cable
  plugged in. The phone screen is **not required**; the proof lives on the
  gateway.
- **Diagram:** the README mermaid flowchart, re-drawn cleanly or animated.

---

## 4. Recording guide

### Tools
- **QuickTime Player** (File → New Screen Recording) for a quick, clean single
  capture of the Mac screen. Choose "Record Selected Portion" to crop to the
  terminal/window and keep stray menu-bar items out of frame.
- **OBS Studio** when you need scenes (split-screen Mac + gateway), webcam inset,
  or a separate audio track. Add a *Display Capture* or *Window Capture* per
  source and arrange them as scenes matching the shot list.
- **`scrcpy`** (optional) to mirror the Android screen over USB if you show the
  phone.
- **Terminal:** use a large, legible font (16–18pt+), high-contrast theme, and a
  wide window. Consider a fresh profile with a neutral prompt (no hostname/user
  baked into the prompt — see redaction).

### Resolution / format
- Record at **1920×1080 (1080p)** minimum; **2560×1440** if your display allows,
  for crisp terminal text. 30fps is fine for screen content; 60fps if you have
  smooth scrolling/animation.
- Export **H.264 MP4**, ~12–20 Mbps for 1080p.
- **Social cut:** also export a **1080×1080 (square)** or **1080×1920 (vertical
  9:16)** crop. Keep the key action (logs/metrics) centred so it survives the
  crop.

### Audio
- Use an external or headset mic, not the built-in mic; record in a quiet, soft
  room to cut echo.
- Record voiceover **separately** from the screen take if you can — it's far
  easier to re-do a line than to re-run the whole demo.
- Normalise to about **-16 LUFS** (or -14 LUFS for social), add a gentle
  high-pass + light noise reduction. Leave ~0.5s of silence head and tail.
- Burn in captions for the social cut (assume muted playback).

### REDACTION checklist (do this BEFORE you hit record)

Anything that identifies you, your network, your phone or your account must not
appear on screen. Prefer the artefacts that are **redaction-clean by design**:
the gateway's **redacted JSON logs** and the **coarse-token `/metrics`** (every
label is a fixed token like `wi-fi`, `android-usb-tether`, `first-copy` — never
an IP, MAC, port, serial or session id). These are safe to show in full.

Blur, crop, or avoid showing:
- [ ] **Real IP addresses** — your home/ISP public IP, the gateway's real IP, any
      LAN addresses. Pass the gateway IP via env (`GEMINA_GATEWAY_IP=…`) so it
      is never compiled in, and keep it off-screen; show `gateway.example.com` as
      a placeholder in any client config.
- [ ] **The phone's carrier and its cellular public IP** — do not show `tcpdump`
      output or DHCP leases that reveal the carrier-assigned address or APN.
- [ ] **MAC addresses** — the phone's tether MAC, any `ioreg`/`ifconfig` dumps.
      Avoid raw `ioreg`, `ifconfig`, `networksetup`, `arp` output on camera.
- [ ] **Serial numbers / device identifiers** — USB serials, VID/PID specifics
      tied to your unit, IORegistry dumps.
- [ ] **Account / personal info** — email, hosted-tier tokens or entitlement
      keys, SSH known-hosts, the real gateway hostname, Git remote URLs with
      usernames, browser tabs/bookmarks, notification banners.
- [ ] **Shell prompt / window chrome** — hostname and username baked into the
      prompt; set a neutral prompt (e.g. `PS1='$ '`) and hide the Mac username in
      paths where possible.
- [ ] **Notifications** — enable **Do Not Disturb / Focus** so Messages, Mail and
      Calendar pop-ups don't leak content mid-take.
- [ ] **The call demo** — use a throwaway/test call or an `ssh` to a box you own;
      don't show a real contact's name, face or video.
- [ ] **The `-json` preflight output** — it's machine-readable; glance only, and
      confirm it carries no host identifiers before showing it.

**Reset terminal scrollback before recording** (so earlier, un-redacted output
isn't scrollable on camera):
- In the terminal, run **`clear`** to clear the visible screen, or **`reset`** to
  reset the terminal fully.
- To wipe **scrollback** so nothing earlier can be scrolled into view:
  - **macOS Terminal.app:** Edit → Clear Scrollback (⌥⌘K), or `printf '\033[3J'`.
  - **iTerm2:** Edit → Clear Buffer (⌘K), or Clear Scrollback History.
- Best practice: start each recorded segment in a **brand-new terminal window**
  with a fresh, neutral profile, so there is no prior history at all.

**Final pass:** scrub the exported video frame-by-frame over any terminal/metrics
sections and confirm no IP, MAC, serial, carrier name, hostname or personal
detail is legible — including in reflections, blurred edges, and the macOS menu
bar.

---

## 5. Post copy — description, caption & chapters

### Long description (YouTube / blog)

> **Gemina VPN — keep your calls and SSH sessions alive when one network
> blips.**
>
> Gemina VPN is an open-source macOS *reliability* tool. It sends your traffic
> over **two uplinks at once** — your Wi-Fi *and* an Android phone's cellular link
> over USB tethering — to a single gateway. The gateway delivers the first copy of
> each packet and discards the duplicate, so if one link drops mid-session, the
> other is already carrying the same packets and the session doesn't notice. It's
> built for **reliability, not extra speed**.
>
> Works with **any Android, no root**, and needs **no kernel extension or SIP
> change on the Mac**. **Open source and self-hostable** — run the gateway as one
> small container on one UDP port — with an optional paid hosted gateway planned.
>
> In this video we show the **real, proven** transport: `geminactl preflight`
> giving a plain "supported" verdict, the gateway's redacted logs and Prometheus
> `/metrics` showing both paths delivering and duplicates being deduplicated, and
> a live failover — pull the Wi-Fi and the session keeps running on cellular.
>
> **Status: pre-release.** The dual-path transport is proven end-to-end via
> command-line tools and gateway telemetry; encryption and the shipping macOS app
> are in active development. No staged UI, no invented numbers — every figure is
> read live off the gateway.
>
> ★ Star the repo: https://github.com/joanmarcriera/gemina
> Join the waitlist: <waitlist link>

### Short caption (X / LinkedIn)

> A Wi-Fi blip shouldn't kill your call. Gemina VPN sends every packet over
> Wi-Fi **and** your Android's cellular at once; the gateway keeps the first copy,
> drops the duplicate. Pull the Wi-Fi mid-session — it keeps running. Open source,
> self-hostable. Pre-release (transport proven, app in progress). ★ + waitlist 👇

### Chapter markers

| Time | Chapter |
|---|---|
| 0:00 | The problem: one blip, dropped call |
| 0:20 | The idea: send every packet twice |
| 0:45 | Live proof: `geminactl preflight` |
| 1:05 | Live proof: both paths delivering + dedup (logs & metrics) |
| 1:35 | Live proof: pull Wi-Fi, session survives on cellular |
| 1:55 | Open source: self-host or hosted gateway |
| 2:20 | Pre-release status & call to action |

---

### Recording-day checklist (tear-off)

- [ ] Phone connected, USB tethering on; Wi-Fi up; gateway reachable (`ssh oracle`).
- [ ] Gateway running with `/metrics` enabled (`GEMINA_GATEWAY_METRICS_ADDR`).
- [ ] Do Not Disturb on; neutral terminal profile; scrollback cleared; large font.
- [ ] Gateway IP via env, kept off-screen; `gateway.example.com` placeholder in configs.
- [ ] Dry-run the failover (Wi-Fi off/on) once so the metrics move cleanly on the take.
- [ ] Voiceover recorded separately; captions drafted.
- [ ] Final redaction scrub of the export before publishing.
