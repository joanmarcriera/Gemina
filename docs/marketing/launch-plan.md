# Go-to-market launch plan — Gemina VPN

> **Status: pre-release.** This plan governs the public launch of an open-source
> product whose **dual-path transport is proven end-to-end** but whose **encryption,
> shipping macOS app and paid hosted tier are still in development**. Everything below
> is constrained by that fact. It deliberately contains **no invented metrics, users,
> testimonials, benchmarks or pricing**. British English throughout.
>
> Companion documents: [`press-kit.md`](press-kit.md), [`seo.md`](seo.md),
> [`video-script.md`](video-script.md). Do not duplicate or contradict them; this plan
> sequences and packages what they describe.

---

## 0. The honest baseline (read first)

Drawn from [`README.md`](../../README.md), [`PROJECT_STATE.md`](../../PROJECT_STATE.md),
[`docs/product/monetisation.md`](../product/monetisation.md) and the SEO guardrails.
Every claim in this plan must stay inside these truths:

**What it is.** A macOS *reliability* tool. It duplicates protected traffic over **two
uplinks at once** — Wi-Fi **and** an Android phone's USB tether (cellular) — sends both
copies to one gateway, and the gateway delivers the first copy of each packet and
discards the duplicate. If one link drops mid-session, the other is already carrying the
same packets, so the session does not notice.

**What it is *not*.** Not bandwidth aggregation. It does **not** combine the speed of
two links; it spends the second link on certainty. Never claim "faster", "double your
bandwidth" or "bonded speed".

**How it runs.** Works on **any Android with USB tethering** from a *userspace* driver —
**no root on the phone, no kernel extension, no DriverKit and no SIP changes on the
Mac**. Some phones (Pixel, AOSP 14+) are also recognised natively by macOS via CDC-NCM.

**Open-core.** Client + shared core are Apache-2.0; the gateway is AGPL-3.0.
Self-hostable as a single container; an **optional paid hosted gateway** is planned
(billed via Stripe; **pricing to be announced**).

**Proven vs in development.**
- **Proven end-to-end:** dual-path duplicate transmission over two independent WANs
  (Wi-Fi + real cellular), server-side deduplication to a single delivery, survival of
  either path dropping, and a userspace Android USB tether data plane needing no
  root/kext/SIP. Evidence: the *Definition of Dual-Path Success — ACHIEVED* section of
  `PROJECT_STATE.md`.
- **In development (do not claim as shipping):** the shipping macOS app data path
  (`NEPacketTunnelProvider`), the on-wire handshake message and pinned-key distribution,
  accounts and payments, and the paid hosted tier.
- Encryption note: the encrypted core (`pkg/clientcore`, AES-256-GCM + X25519/HKDF with
  a pinned Ed25519 gateway identity) is **built and unit/race tested**, but it is **not
  yet wired through the shipping app**, so today's public proof carries probe packets,
  not your encrypted IP traffic. Frame encryption as "designed and built into the core,
  integration in progress" — never as a delivered, audited shipping feature.

---

## 1. Positioning & messaging

### 1.1 The one-liner

> **Gemina VPN keeps your Mac's calls, SSH and VPN sessions alive when one link
> blips — by sending your traffic over Wi-Fi and your Android phone's cellular at the
> same time. Open source, self-hostable. Pre-release.**

Shorter, for GitHub description / OG tags:

> *Seamless internet failover for macOS — Wi-Fi and an Android USB tether at once. Open
> source, self-hostable. Pre-release.*

### 1.2 The three core messages

1. **Reliability you can feel, not speed you can't.** We duplicate every packet over two
   independent links and keep the first copy of each. One link can drop mid-call and the
   session simply continues — no reconnect. We are explicit that this is *redundancy*,
   not bonding: it does not add the two links' bandwidth together.
2. **Any Android, no root, no kernel hacks.** The Mac drives the phone's USB tether from
   an unprivileged userspace process — no root on the phone, no kernel extension, no
   DriverKit, no SIP changes. No second SIM, no special hardware. This is the unusual
   engineering result that earns technical trust.
3. **Open source, yours to run.** Client and core are Apache-2.0, the gateway AGPL-3.0.
   Read the code, self-host the gateway as one container on one UDP port — no database,
   no accounts — or, later, pay for a hosted gateway and skip the operations. Same
   open-source client either way.

### 1.3 Audience segments (from [`seo.md`](seo.md) §1)

| Segment | The pain | Our honest answer | Primary channels |
|---|---|---|---|
| **Remote workers on flaky Wi-Fi** | Calls freeze; dropped from meetings | Seamless failover keeps the call up; the second link is your phone | Product Hunt, X/LinkedIn, symptom blog |
| **Developers / sysadmins** | SSH sessions die; long builds/deploys break on a blip | Packet duplication, not reconnection; open source, inspectable | Show HN, Lobsters, r/selfhosted, dev blog |
| **Field / on-the-move users** | Patchy coverage on site, in vehicles, at events | Independent WANs (Wi-Fi + cellular) carry the same packets | X/LinkedIn, scenario blog |
| **Live streamers / broadcast** | A dropped frame ruins a stream; want link redundancy | We do *redundancy* (failover), not bonded bitrate — stated plainly | Niche forums, comparison page |
| **People who lose calls in lifts / trains / tunnels** | Connection dies when they move between dead zones | Both paths carry it; one surviving link = no interruption | Symptom blog, social cut |

Cross-cutting **self-hoster / open-source** intent sits over all five and converts on
credibility (read-the-code, run-it-yourself), not marketing. This is the launch's
primary early audience.

### 1.4 Honesty guardrails — what NOT to claim

These are non-negotiable. Any asset that breaks one is blocked from publishing.

- **Do not claim speed.** No "faster", "double your bandwidth", "bonded throughput",
  "aggregate your connections". If a speed framing is raised, correct it.
- **Do not imply it ships today.** No "download now", no "install the app", no app-store
  badge. The macOS app is in development.
- **Do not present encryption as a delivered, audited shipping feature.** It is built
  into the core and tested, but not yet integrated into the shipping app; today's public
  proof carries probe packets. Say so.
- **No invented numbers.** No user counts, star counts, ratings, testimonials,
  latency/throughput benchmarks, or uptime figures we have not measured. Every figure in
  the demo is read live off the gateway during the take.
- **No pricing.** The hosted tier is "planned, pricing to be announced". Never quote a
  price, a discount, or a fake "launch deal".
- **No dark patterns.** No countdowns, fake scarcity, pre-ticked boxes. Keep the site's
  promise: *we'll tell you the price before anything is ever charged.*
- **No disparagement / no fake comparisons.** Comparison pages state what competitors do
  well and our *current* pre-release limitations; never publish an unmeasured benchmark.
- **No misuse of "VPN".** Today it is a reliability transport; encryption and the VPN
  data path are in development. Don't imply a privacy/anonymity product it isn't.

---

## 2. Phased launch plan

Three phases, gated on readiness rather than dates. Each lists goals, channels, the
assets it needs, and **success signals chosen to be real, not vanity** — we measure
qualified intent (self-host installs, waitlist quality, substantive issues), not raw
counts displayed as proof.

### Phase 0 — Pre-launch (foundations & funnel)

**Goal.** Be submission-ready and credible *before* any post goes out, with the
compatibility preflight as the top of the funnel. Nothing in Phase 1/2 should ship until
Phase 0's checklist (§4) is green.

**Channels.** None public yet. This phase is internal preparation plus a quiet,
shareable-by-link state (repo public-readiness, site live, waitlist open).

**Assets needed.**
- Public repo, audited and ready (see §4; run `scripts/prepare-public.sh`).
- Static site live (the `website/` page) with the honest pre-release banner, real repo
  URL, OG/Twitter cards, JSON-LD, `robots.txt`/`sitemap.xml` (per `seo.md` §3–4).
- Waitlist endpoint working, privacy-respecting, with the "price before charge" promise.
- `geminactl preflight` working as the funnel: a visitor runs it (or uses the
  web-consumed `-json` report) to learn, honestly, whether their Mac + Android combo is
  supported, present-but-not-yet-usable, or needs a change. This sets correct
  expectations *before* anyone joins the waitlist.
- The "what's proven / what isn't" transparency article (`seo.md` content #11) drafted.
- Press kit ([`press-kit.md`](press-kit.md)) finalised; logo + screenshots/diagram
  placeholders filled.
- Demo video recorded per [`video-script.md`](video-script.md) (real gateway telemetry,
  redaction scrub done).

**Success signals (real, not vanity).**
- `prepare-public.sh` returns **GO**; git history rewritten to drop the old LAN address.
- Preflight runs cleanly on at least the maintainer's own Mac + Android combinations and
  returns an honest verdict (including the present-but-not-usable case).
- Waitlist captures *qualified* sign-ups (people who ran preflight first) rather than
  raw email volume — track "ran preflight → joined" as the quality signal.
- Site passes Core Web Vitals and structured-data validation; no honesty-guardrail
  breaches in any asset.

### Phase 1 — Developer / early-adopter launch

**Goal.** Earn technical credibility with the people who convert on *reading the code and
running it themselves*. Get the gateway self-hosted by real users and surface substantive
issues. This is where reputation is made or lost; honesty is the strategy.

**Channels.**
- **Show HN** — lead with the proven engineering result (userspace Android USB tether on
  macOS, no root/kext, proven dual-path failover), clearly labelled pre-release.
- **r/selfhosted** — the AGPL gateway as a single self-hosted container; lead with
  `docker run`, no accounts, no database.
- **r/macapps / r/MacOS** — the symptom ("calls drop on bad Wi-Fi") and the any-Android,
  no-root setup; respect each subreddit's self-promotion rules; pre-release framing.
- **Lobsters** — the userspace-RNDIS engineering write-up.
- **Relevant Discords/forums** — self-hosting and macOS-developer communities; share as a
  participant, not a billboard; link the repo and the transparency article.

**Assets needed.**
- The ready-to-post drafts in §3 (Show HN, r/selfhosted).
- The self-host walkthrough (`seo.md` content #6) and the dual-path / userspace-RNDIS
  engineering posts (#9, #10) published, so links point somewhere substantial.
- A populated `CONTRIBUTING`, `SECURITY.md`, issue templates, and a labelled "good first
  issue"/roadmap view so incoming developers can engage.
- The demo video available but **not** the centrepiece yet (Phase 1 leads with code).

**Success signals (real, not vanity).**
- **Self-host installs:** people actually running `docker run` against their own gateway
  and pointing a client at it — measured via voluntary feedback, issues, and (privacy-
  respecting) image-pull/referrer signals, not as a headline number.
- **Issue quality:** substantive bug reports, compatibility reports for specific Android
  models, and architecture questions — evidence people read the code.
- **Waitlist quality:** sign-ups that arrive *after* reading the transparency article.
- GitHub stars/forks treated as a community-health indicator, **not** displayed on-site
  as proof.
- Sentiment: discussion engages with the real trade-off (reliability not speed,
  pre-release) rather than expecting a finished product — a sign the framing landed.

### Phase 2 — Broader launch

**Goal.** Reach beyond the technical core to the symptom audience (people who lose calls
on flaky networks) once the product story is robust and the explainer video is polished.
Only enter Phase 2 after Phase 1 feedback is incorporated and claims are still fully
honest.

**Channels.**
- **Product Hunt** — tagline + description + maker comment from §3; the explainer video
  as the hero asset; honest pre-release status pinned in the maker comment.
- **The explainer video** ([`video-script.md`](video-script.md)) — published on
  YouTube/site and cut for X/LinkedIn (the 30–45s social cut).
- **X / LinkedIn announcement thread** (§3), timed with Product Hunt.
- **Comparison pages** — honest "vs Speedify" and "open-source connection-reliability
  tools compared" (`seo.md` §6.4), capturing high-intent comparison searches.
- **Awesome-lists & directories** — submit to `awesome-selfhosted`, `awesome-macos`,
  `awesome-go` etc. **only once there is a usable release**, to avoid burning goodwill
  (per `seo.md` §6.3).

**Assets needed.**
- Product Hunt assets (gallery images, the explainer video, maker comment).
- Comparison pages published and validated against the honesty guardrails.
- The symptom-led blog posts (`seo.md` content #1, #4) live as landing destinations.
- Updated press kit reflecting any milestone reached (e.g. app integration progress).

**Success signals (real, not vanity).**
- Quality of inbound from the symptom audience: do they understand it's reliability, not
  speed, and pre-release? (Measured via question themes, not upvote count.)
- Search Console: appearing for *failover/reliability* terms rather than *speed* terms —
  the right audience finding us (per `seo.md` §7).
- Comparison-page traffic converting to repo visits and self-host docs, not bounce.
- Continued substantive issues and compatibility reports; sustained, honest engagement.
- Product Hunt: treat ranking/upvotes as secondary; the real signal is qualified
  waitlist sign-ups and self-host attempts attributable to the launch.

---

## 3. Ready-to-post drafts

> **All copy below is a DRAFT for review.** Replace every `<placeholder>` before posting.
> Do not post any of it until §4's checklist is green and a human has re-checked it
> against the §1.4 honesty guardrails. `<repo>` =
> `https://github.com/joanmarcriera/gemina`; `<waitlist>` = the live waitlist URL;
> `<site>` = the live site URL.

### 3.1 Show HN — DRAFT

**Title:**

> Show HN: Gemina VPN – userspace Android USB tether on macOS, no root or kext (pre-release)

**Body:**

> Gemina VPN is an open-source macOS *reliability* tool. It sends your traffic over
> two uplinks at once — your Wi-Fi **and** an Android phone's cellular link over USB
> tethering — to a single gateway. The gateway delivers the first copy of each packet and
> discards the duplicate, so if one link drops mid-session, the other is already carrying
> the same packets and the session doesn't notice. It's reliability, not speed: it does
> **not** add the two links' bandwidth together.
>
> The part I think is interesting to this crowd: the Mac drives the phone's USB tether
> from an **unprivileged userspace process** — no root on the phone, no kernel extension,
> no DriverKit, no SIP changes. A userspace RNDIS data plane claims the phone's USB
> interfaces, completes the RNDIS init, sets the packet filter, holds a DHCP lease,
> ARP-resolves the phone's gateway, and pushes real UDP/IP frames out over cellular. It
> works on any Android with USB tethering (RNDIS is universal); some phones (Pixel/AOSP
> 14+) also work via macOS's native CDC-NCM.
>
> **What's proven (end-to-end):** the same logical packet sent over Wi-Fi *and* over the
> phone's cellular link both reach a deployed gateway from two distinct public WAN IPs,
> get deduplicated to one delivery, and either path can drop without ending the session.
>
> **What's NOT done yet (so I'm not claiming it):** the shipping macOS app data path
> (`NEPacketTunnelProvider`), wiring the encrypted core (it's built and tested —
> AES-256-GCM + X25519 with a pinned gateway identity — but not yet integrated into the
> app), and accounts/payments. Today's public proof carries probe packets, not your
> encrypted IP traffic.
>
> Open-core: client + core Apache-2.0, gateway AGPL-3.0. Self-host the gateway as one
> container on one UDP port (no DB, no accounts); an optional paid hosted gateway is
> planned later (pricing TBD).
>
> Repo (with the dual-path evidence write-up): `<repo>`
> What's proven vs not: `<site>`/blog/what-pre-release-means
> Happy to go deep on the RNDIS/userspace bits.

### 3.2 Product Hunt — DRAFT

**Tagline (≤60 chars):**

> Keep your Mac online when one link drops — open source

**Description:**

> Gemina VPN is an open-source macOS reliability tool. It sends your traffic over
> Wi-Fi **and** your Android phone's cellular (USB tethering) at the same time, to one
> gateway that keeps the first copy of each packet and drops the duplicate. If one link
> blips mid-call, the other is already carrying it — so your call, SSH or VPN session
> doesn't notice. It's built for reliability, not extra speed (it doesn't bond bandwidth).
> Works with any Android, no root, and needs no kernel extension or SIP change on the Mac.
> Self-host the gateway as one container, or use the planned hosted option.
>
> **Pre-release:** the dual-path transport is proven end-to-end; encryption and the
> shipping app are in active development.

**Maker comment (pin this):**

> Hi PH 👋 I built this because a one-second Wi-Fi blip — a lift, a train, a flaky café —
> kept killing my calls and SSH sessions. Instead of reconnecting faster, Gemina VPN
> sends every packet twice over two independent links and keeps whichever copy arrives
> first, so a dropped link never ends the session.
>
> Being upfront because it matters here: **this is pre-release.** What's *proven today* is
> the transport — packets surviving a real link drop, observed live at a deployed gateway,
> with the phone's cellular driven from userspace (no root, no kernel extension, no SIP
> changes). What's still **in development**: the shipping macOS app, wiring the
> already-built encrypted core into it, and accounts/payments. So there's no "download the
> app" button yet — there's a repo to read and a gateway you can self-host, plus a
> waitlist. No invented benchmarks, no fake numbers; every figure in the demo is read live
> off the gateway. It's reliability, not speed — it won't make you faster. Pricing for the
> optional hosted tier is TBD and we'll tell you before anything is ever charged.
>
> Code: `<repo>` · Waitlist: `<waitlist>` · Happy to answer anything.

### 3.3 r/selfhosted — DRAFT

**Title:**

> Self-hostable internet-failover gateway for macOS clients — one container, one UDP port, no DB (open source, pre-release)

**Body:**

> I'm building Gemina VPN, an open-source macOS reliability tool, and the server half
> is squarely a r/selfhosted thing, so I wanted to share it here honestly.
>
> The client sends your traffic over two links at once — Wi-Fi and an Android phone's USB
> tether (cellular) — to a **gateway you run yourself**. The gateway deduplicates the two
> copies down to one delivery, so if a link drops mid-session the surviving link keeps it
> going. Reliability, not speed (it doesn't bond bandwidth).
>
> The gateway is AGPL-3.0 and deliberately boring to operate: **one container, one UDP
> port, no database, no accounts.**
>
> ```
> docker run --rm --read-only -p 51820:51820/udp \
>   -e GEMINA_GATEWAY_ADDR=:51820 \
>   <your-image>
> ```
>
> The client points at your gateway by hostname (`gateway.example.com:51820`) — the
> address is always configurable, never hard-coded, so self-hosting and the (planned,
> optional, paid) hosted option use the same open-source client. The gate that exists in
> the code is hosted-tier-only; **self-hosting is free and ungated.**
>
> **Honest status — pre-release.** The dual-path transport is proven end-to-end (same
> packet over Wi-Fi + real cellular, deduplicated at a deployed gateway, either path can
> drop). Encryption (built into the core, tested) and the shipping macOS app are still in
> development, so today the proof carries probe packets, not your encrypted traffic. No
> fake numbers — happy to show the gateway's redacted logs and Prometheus `/metrics`.
>
> Repo + self-host docs: `<repo>`

### 3.4 X / LinkedIn announcement thread — DRAFT

> **1/** A one-second Wi-Fi blip shouldn't kill your call. Gemina VPN is an
> open-source macOS reliability tool that sends your traffic over Wi-Fi **and** your
> Android's cellular at the same time — so when one link drops, your session doesn't.
> 🧵 Pre-release. `<repo>`
>
> **2/** How it works: every packet goes out twice, over two independent links, to one
> gateway. The gateway keeps the first copy and drops the duplicate. One link blips? The
> other is already carrying the same packets. No reconnect.
>
> **3/** Important: this is **reliability, not speed.** It does *not* add the two links'
> bandwidth together. It spends the second link on certainty — a call/SSH/VPN session that
> survives a dropped network.
>
> **4/** The unusual bit: the Mac drives the phone's USB tether from **userspace** — no
> root on the phone, no kernel extension, no SIP changes. Any Android with USB tethering
> works. No second SIM, no special hardware.
>
> **5/** It's open source. Client + core are Apache-2.0; the gateway is AGPL-3.0 and
> self-hostable as one small container on one UDP port — no DB, no accounts. An optional
> paid hosted gateway is planned later (pricing TBD; we'll tell you before charging).
>
> **6/** Honest status: the dual-path transport is **proven end-to-end** (Wi-Fi + real
> cellular, deduplicated at a deployed gateway, either path can drop). Encryption and the
> shipping app are **in development** — today's proof carries probe packets, not your
> encrypted traffic. No fake numbers.
>
> **7/** If you've ever lost a call to a flaky network: ★ the repo and join the waitlist,
> and follow along as we ship. Repo: `<repo>` · Waitlist: `<waitlist>`

---

## 4. Launch checklist (pre-publish)

Complete and verify **before any Phase 1 post**. The repository must go public from a
clean, audited state. Cross-references: [`scripts/prepare-public.sh`](../../scripts/prepare-public.sh)
(read-only GO/NO-GO audit) and [`docs/dev/repository-strategy.md`](../dev/repository-strategy.md)
(release runbook).

### 4.1 Repository public-readiness
- [ ] Run `scripts/prepare-public.sh` from the repo root; it returns **GO** (exit 0). It
      audits tool/scratch dirs, lockfiles/built binaries, licence/notice files, obvious
      secrets, and raw IPv4 addresses.
- [ ] **Rewrite git history to drop the old real LAN address** (flagged in
      `PROJECT_STATE.md` and the release audit). Removing the file at HEAD is **not
      enough** — purge it from history; if any real secret ever appeared, rotate it too.
- [ ] Confirm no `.research-src/` or upstream implementation source is tracked; no GPL
      implementation code in product directories.
- [ ] Replace placeholder URLs: `github.com/example/...` and
      `ghcr.io/example/gemina-gateway` in README/site/docs with the real repo/image;
      `geminavpn.example` with the real domain.
- [ ] Verify the licence invariant in CI: **the client never imports gateway packages**
      (per `repository-strategy.md`).
- [ ] Repo hygiene: README's honest first paragraph intact; topics/description set
      (`seo.md` §6.1); `CONTRIBUTING`, `SECURITY.md`, `CODE_OF_CONDUCT`, issue templates,
      and a roadmap/"good first issue" view present.

### 4.2 Licence finalised
- [ ] AGPL-3.0 gateway + Apache-2.0 client/core applied with per-directory headers;
      `LICENSE`, `NOTICE`, `docs/legal/licensing.md`, `LICENSES/*.txt` present and
      consistent.
- [ ] `make licence-check` passes; any third-party material retains its original licence
      and attribution.

### 4.3 Privacy / terms live
- [ ] Privacy policy and terms published at stable URLs and linked from the site/waitlist,
      consistent with the data actually collected.
- [ ] Waitlist data handling documented; the "we'll tell you the price before anything is
      ever charged" promise stated and honoured.
- [ ] No payment is taken at launch (hosted tier not yet available); the entitlement gate
      stays hosted-only and self-host stays free/ungated.

### 4.4 The preflight working
- [ ] `geminactl preflight` returns honest verdicts on the maintainer's Mac + Android
      combinations, **including** the *present-but-not-yet-usable* case (no faked
      "supported").
- [ ] `geminactl preflight -json` produces a clean, host-identifier-free report for
      the app/site to consume.
- [ ] The site's compatibility section funnels visitors through preflight *before* the
      waitlist, setting correct expectations.

### 4.5 Demo video recorded
- [ ] Recorded per [`video-script.md`](video-script.md): real gateway telemetry (redacted
      JSON logs + Prometheus `/metrics`), live failover (pull Wi-Fi, session survives on
      cellular), **no staged app UI, no invented numbers**.
- [ ] Redaction scrub complete: no IPs, MACs, serials, carrier names, hostnames, account
      data, or personal detail anywhere in the export (including reflections and the menu
      bar).
- [ ] An honest pre-release status badge is on screen.

### 4.6 Analytics without dark patterns
- [ ] Privacy-respecting analytics only (e.g. self-hosted Plausible/Umami); Google/Bing
      Search Console + sitemap submitted (per `seo.md` §7).
- [ ] No countdowns, fake scarcity, pre-ticked boxes, fabricated counts/ratings/
      testimonials/benchmarks anywhere (site, schema, social, video).
- [ ] JSON-LD `offers` honest (self-host `0`, no invented hosted price); **no**
      `aggregateRating`/`review`. Structured data validated.

### 4.7 Final honesty pass
- [ ] A human re-reads every launch asset (drafts in §3, site, video, press kit) against
      the §1.4 guardrails. Any breach blocks publish.

---

## 5. Sequencing summary

1. **Phase 0** — finish §4; site + waitlist + preflight live; press kit + video ready;
   `prepare-public.sh` = GO; history rewritten. *Gate: no honesty breaches.*
2. **Phase 1** — repo public; Show HN + r/selfhosted + r/macapps + Lobsters + Discords,
   leading with the proven engineering result. *Gate: substantive issues handled, claims
   still honest before going broader.*
3. **Phase 2** — Product Hunt + explainer video + X/LinkedIn thread + comparison pages;
   awesome-lists only once there's a usable release. *Gate: milestones reached are
   reflected truthfully; pre-release framing maintained until the app actually ships.*

Re-baseline messaging and update claims **only** when a real milestone lands (encryption
wired into the app, app shipped, hosted tier priced) — then, and only then, retire the
relevant pre-release caveats.
