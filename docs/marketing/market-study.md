# Market study: reach expansion + first-quarter revenue

Date: 2026-06-24. Estimates are clearly labelled; re-verify before relying on any
figure. This is an internal strategy note, not a forecast to bank on.

## TL;DR (the honest version)

- **Biggest easy reach win: support the iPhone as the second path.** macOS already
  exposes an iPhone Personal Hotspot over USB as a native network interface — no
  driver, none of the Android RNDIS work. Our transport already binds to any
  interface (`IP_BOUND_IF`), so **Wi-Fi + iPhone-USB should work with the existing
  code**. Most Mac owners carry an iPhone; this turns a niche (Mac + Android +
  USB tether) into "most Mac users".
- **Reframe the pitch from "Wi-Fi + Android tether" to "bond any two
  connections"** (Wi-Fi + iPhone, + Android, + Ethernet, + an LTE/USB-Wi-Fi
  dongle). Free reach, no code.
- **Distribution shortcuts beat marketing spend:** Setapp (existing paying Mac
  audience, no billing to build), Homebrew cask (self-host crowd), one-click
  cloud deploy for the gateway.
- **Revenue in the first 3 months will almost certainly be small — realistically
  £0 to a few hundred.** The first quarter's real return is a shipped app, an
  audience, GitHub credibility and validation, not cash. Plan accordingly.

## What we are actually selling, and the current reality

A macOS reliability tool: duplicate the *protected* traffic over two links at
once so a blip on one never drops the session. Reliability, not speed.

Reality check that shapes every number below:
- **No shipping consumer app yet.** The transport, encryption, handshake, gateway
  and billing primitives are proven in Go and the Swift bridge compiles, but the
  signed macOS app (NEPacketTunnelProvider + Xcode + notarisation) is not built.
  Nothing is installable by a non-technical buyer today.
- **Friction is high:** needs a Mac *and* a second connection *and* (today) an
  Android phone with a USB cable and a tether toggle. Every "and" halves the
  market. The iPhone path removes the worst "and".
- **The free self-host tier cannibalises the most technical buyers.** That is good
  for adoption and bad for near-term cash: the people most able to find and run
  it are the least likely to pay us to host it.

## Who needs "reliable, not faster" (segments)

Ranked by willingness to pay, not size:

1. **People whose income depends on a call not dropping** — remote workers on
   client calls, live streamers, online poker/trading, telehealth, remote IT/SSH,
   sales demos. Small audience, high willingness to pay (£15–30/mo is plausible),
   reliability is a real cost-avoidance.
2. **Mobile / field / transport** — vans, boats, RVs, trains, events, pop-up
   sites. Underserved, loyal, but often want a router appliance, not a Mac app.
3. **Self-hosters / homelab / privacy crowd** — the launch audience and advocates.
   They will mostly **self-host for free** (that is the model). Monetise via
   goodwill, contributions, and word of mouth, not subscriptions.
4. **General prosumers on flaky Wi-Fi** — biggest headcount, lowest willingness to
   pay, hardest to convince that the problem is their setup and not "the café
   Wi-Fi". This is a grudge purchase made *after* being burned.

## Easy reach expansions, ranked by impact ÷ effort

| Lever | Effort | Reach impact | Notes |
|---|---|---|---|
| **iPhone hotspot as a path** | **Low** (test + market; likely already works) | **High** | Removes the Android/RNDIS barrier for the Mac+iPhone majority. Verify `Wi-Fi + iPhone-USB` end-to-end, then lead with it. |
| **Reframe to "bond any two connections"** | Trivial (copy) | High | Wi-Fi + Ethernet/dongle/phone. Widens the "do I qualify?" answer to "yes". |
| **One-click gateway deploy** (Fly.io/Railway/Render/DO 1-click, docker-compose) | Low | Medium-High | Turns "self-host" from a barrier into a button; feeds both adoption and hosted upsell. |
| **Homebrew cask + `continuityctl` polish** | Low | Medium | Free reach into the dev/self-host crowd; the preflight as a hook. |
| **Setapp distribution** | Low-Medium | Medium-High | Existing paying Mac audience, **no billing to build**; usage-based 70/30 + 20% for users you bring, or 85/15 single-app. Sidesteps the cold-funnel problem. |
| **Scope which traffic is duplicated** (only the call/SSH, not everything) | Medium | Medium | Also a *selling point*: protects the critical flow without doubling all cellular data. Already in the footprint ADR. |
| **Windows/Linux client** | High | High (new platforms) | Gateway is already cross-platform; a non-Mac client is a real project (no NE), not "easy". Park it. |

The top three are the cheap wins. **iPhone support is the single highest-leverage
easy change** because it multiplies the addressable base with little code.

## Competitor benchmark

**Speedify** is the incumbent for connection bonding: free 2 GB tier, then ~£/$7.49/mo
annual to $14.99/mo monthly (families ~$22.50, teams from $14.99). Years of
polish, bonds without our second-device NIC tricks, positioned premium. Our
differentiation is **open-source + self-hostable + privacy + price**, not features
— we will not out-feature them quickly. Note they sell *speed + reliability*; we
deliberately sell only reliability, which is a narrower but more honest claim.

## Pricing thoughts

- Hosted tier likely **£5–9/mo** (under Speedify, above hosting cost). A **lifetime
  / one-time** option reduces friction for a utility and suits a solo maker's cash
  needs. Keep self-host free forever (it is the top of the funnel and the moat).
- Consider a **prosumer tier** (£15–20/mo) aimed at segment 1 (income-dependent
  reliability) with priority/region gateways — higher willingness to pay, smaller
  support load.

## First-quarter revenue model (label: estimate, conservative)

Assumes a solo maker, no ad budget, the app ships ~month 2, one decent technical
launch (Show HN + r/selfhosted + Product Hunt + the explainer video).

| Scenario | Site visitors (qtr) | Waitlist | Paying (exit m3) | ARPU | 3-mo revenue | Exit MRR |
|---|---|---|---|---|---|---|
| **Bear** (app slips / launch flat) | ~1k | ~50 | 0–3 | £7 | ~£0–60 | ~£0–20 |
| **Base** (launch lands, mostly self-host) | 2–6k | 150–600 | 5–25 | £7 | ~£50–400 | ~£35–175 |
| **Bull** (video catches / Setapp accepts / a creator features it) | 10k+ | 1k+ | 50–150 *or* Setapp usage | £7 | ~£500–2k | ~£350–1k |

Why so low: most technical interest self-hosts (free); non-technical buyers are
exactly the ones who find the Android setup too fiddly (the iPhone path helps);
and reliability converts slowly (people buy after being burned). **Setapp is the
wildcard** — it removes the billing/funnel problem and taps an existing paying
audience, and is the most realistic path to non-trivial first-quarter cash.

## Recommendation

1. **Before launch:** verify and ship **iPhone-USB as a second path** (likely a
   day of testing + copy), and reframe the site to "bond any two connections".
2. **Treat Q1 as funnel-building, not revenue.** Optimise for: a shipped, signed
   app; GitHub stars/issues; waitlist *quality*; and 1–2 reference users in
   segment 1 (income-dependent reliability) who will give testimonials later.
3. **Apply to Setapp** in parallel with the direct Stripe tier — it is the
   lowest-friction route to actual subscription revenue for a Mac utility.
4. **Aim the paid message at segment 1**, not "everyone on flaky Wi-Fi".

## Sources

- Speedify pricing (2026): security.org, saasworthy.com.
- Setapp developer revenue model: docs.setapp.com, setapp.com/developers.
- Internal: `PROJECT_STATE.md`, `docs/product/monetisation-apple-study.md`,
  `docs/marketing/seo.md`.
