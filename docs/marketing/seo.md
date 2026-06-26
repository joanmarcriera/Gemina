# SEO & content strategy — Gemina VPN

> **Status: pre-release.** This document plans the discovery and content work for an
> open-source product that has **proven its dual-path transport** but has **not yet
> shipped** the full app or encryption. Every recommendation below is constrained by
> that fact: no fabricated metrics, no testimonials, no "download now", no speed
> claims. We rank for **reliability and seamless failover**, never for "faster
> internet".

---

## 0. Positioning guardrails (read first)

Everything that follows must stay inside these truths, taken from `README.md`,
`PROJECT_STATE.md` and `website/index.html`:

- **What it is.** A macOS *reliability* tool. It duplicates protected traffic over
  **two uplinks at once** — Wi-Fi **and** an Android phone's USB tether (cellular) —
  sends both copies to one gateway, and the gateway delivers the first copy and
  discards the duplicate. If one link drops mid-session, the other is already
  carrying the same packets.
- **What it is not.** Not bandwidth aggregation. It does **not** combine the speed of
  two links; it spends the second link on certainty. Do not chase "faster", "bond for
  speed", "double your bandwidth" intents except to redirect them honestly.
- **How it runs.** Works on **any Android with USB tethering** via a *userspace*
  driver — no root on the phone, no kernel extension, no DriverKit, no SIP changes on
  the Mac. Some phones (Pixel, AOSP 14+) are also recognised natively by macOS via
  CDC-NCM.
- **Open-core.** Client + core are Apache-2.0; the gateway is AGPL-3.0. Self-hostable
  as a single container; an **optional paid hosted gateway** is planned (pricing TBD).
- **Honesty rules for all copy and schema.** The dual-path transport is proven
  end-to-end (Wi-Fi + real cellular, deduplicated at a deployed gateway). Encryption,
  the shipping `NEPacketTunnelProvider` data path, accounts and payments are **in
  development**. Pricing is **TBD / early-access**. Never imply otherwise.

---

## 1. Audience & search intent

We are targeting people whose *connection reliability* is a recurring, painful problem
— not people shopping for raw speed. Five primary segments, mapped to intent.

| Segment | The pain they feel | What they type | Search intent | Our honest answer |
|---|---|---|---|---|
| **Remote workers on flaky Wi-Fi** | Calls freeze; they get dropped from meetings; home/cafe Wi-Fi wobbles | "stop wifi dropping video calls mac", "zoom keeps disconnecting macbook" | Problem-aware, solution-seeking | Seamless failover keeps the call up; second link is your phone |
| **Developers / sysadmins** | SSH sessions die, long builds and `rsync`/deploys break on a blip | "keep ssh session alive when wifi drops", "internet failover mac", "mosh alternative reliable connection" | Solution-aware, technical | Packet duplication, not reconnection; open source, inspectable |
| **Field / on-the-move users** | Patchy coverage; tethered laptop on site, in vehicles, at events | "reliable internet two connections laptop", "backup internet connection macbook" | Solution-seeking | Independent WANs (Wi-Fi + cellular) carry the same packets |
| **Live streamers / broadcast** | A dropped frame ruins a stream; want link redundancy | "bonded connection for streaming open source", "internet redundancy live stream" | Comparison/solution-aware | We do *redundancy* (failover), not bonded bitrate — be explicit |
| **People who lose calls in lifts / trains / tunnels** | Connection dies exactly when they move between dead zones | "why does my call drop in the lift", "internet drops on the train mac" | Problem-aware, early | Both paths carry it; one surviving link = no interruption |

Cross-cutting **self-hoster / open-source** intent sits over the top of all five:
"open source internet failover", "self-host wireguard failover", "speedify alternative
open source". These users convert on credibility (read-the-code, run-it-yourself), not
on marketing.

**Intent ladder we will cover with content:**
1. *Symptom* ("my video call keeps dropping on Mac") → top-of-funnel blog.
2. *Mechanism* ("how does internet failover actually work") → mid-funnel explainer.
3. *Category* ("internet failover for macOS", "bond wifi and cellular mac") → landing
   page + comparison.
4. *Brand/comparison* ("Speedify alternative", "Gemina VPN") → comparison + repo.
5. *Self-host* ("self-host a continuity gateway") → docs-led, high-intent.

---

## 2. Keyword strategy

Cluster by intent. Rank for **reliability/failover**; treat all "speed/faster/bond for
bandwidth" terms as *redirect* terms answered with an honest "that's not what this is".

### 2.1 Primary clusters (category & problem)

- **Failover / reliability (core):**
  `internet failover mac`, `seamless internet failover macos`, `network failover for
  video calls`, `connection failover wifi cellular`, `keep connection alive when wifi
  drops`.
- **Symptom-led (highest volume, top-of-funnel):**
  `stop wifi dropping video calls mac`, `zoom keeps disconnecting macbook`,
  `why does my internet drop during calls`, `wifi keeps dropping mac fix`.
- **Dual-link / two connections:**
  `use two internet connections at once mac`, `bond wifi and cellular mac`,
  `combine wifi and phone internet macbook`, `wifi and cellular at the same time mac`.
  > Honest note: people search "bond" meaning "use both". Our pages must clarify we
  > duplicate for reliability, we do not sum throughput.

### 2.2 Secondary clusters (mechanism, platform, how-to)

- **Android USB tethering on Mac:**
  `android usb tethering mac`, `tether android to macbook usb`, `usb tethering macos
  no driver`, `android tether mac without app store`.
- **SSH / dev reliability:**
  `keep ssh session alive when wifi drops`, `stable ssh on bad wifi`, `ssh survive
  network change mac`.
- **VPN reliability:**
  `vpn keeps disconnecting mac`, `vpn that survives network drop`, `wireguard
  failover`.
- **Packet-level / technical:**
  `packet duplication failover`, `redundant udp transport`, `multipath udp reliability`.

### 2.3 Long-tail (high-intent, low-competition — write content directly at these)

- `stop wifi dropping video calls on macbook`
- `how to keep zoom call from dropping when wifi is bad`
- `why does my vpn drop in the lift`
- `keep internet connection during train journey laptop`
- `android usb tethering mac no root`
- `self host internet failover gateway`
- `open source speedify alternative for mac`
- `dual path vpn that survives a dropped link`

### 2.4 Competitor / comparison keywords (handle truthfully)

These have buyer intent — people already know the category. Address them with honest
comparison pages, never with disparagement or invented benchmarks.

| Keyword | Competitor reality | Our truthful angle |
|---|---|---|
| `speedify alternative`, `speedify open source` | Speedify *bonds* links (can aggregate speed); proprietary; subscription | We are **open source**, **reliability-first** (duplication, not bonding), **self-hostable**. Be explicit: if you want raw aggregate speed, Speedify bonds; if you want a call that never drops and code you can read, that's us. |
| `multiconnect alternative`, `MASV alternative` | MASV/Multiconnect target large-file transfer / acceleration | Different job: we keep interactive sessions (calls, SSH, VPN) alive, not bulk transfer. State the difference plainly. |
| `dispatch / engarde / glorytun failover` | Open-source bonding/redundancy projects | Acknowledge them; our differentiator is the **userspace Android USB tether on macOS with no root/kext** and the macOS-native product framing. |

**Comparison-page rules:** state what the competitor genuinely does well; state our
*current* limitations (pre-release, encryption in development); never publish a metric
we have not measured; never claim "faster".

### 2.5 Keywords to deliberately AVOID ranking for

`fastest vpn mac`, `double internet speed`, `bandwidth aggregation mac`, `combine
internet speeds`. Chasing these brings the wrong audience and forces dishonest copy.
If we mention them, it is only to correct the expectation.

---

## 3. On-page SEO for the static landing page (`website/`)

The current page (`website/index.html`) is already strong: honest pre-release banner,
reliability-not-speed framing, semantic sections, skip link, ARIA labels. The
following are concrete additions/refinements. The GitHub URL placeholder
`github.com/example/gemina` should be replaced with the real repo
`https://github.com/joanmarcriera/gemina` before launch.

### 3.1 Title tag

Keep it benefit-led and under ~60 characters where possible. Current:
`Gemina VPN — your connection never notices a dropout` (good). Alternative that
captures category + platform for search:

```html
<title>Gemina VPN — seamless internet failover for macOS</title>
```

Pick one; do not stuff. If the homepage targets the symptom audience, the current
"never notices a dropout" line is excellent and should stay.

### 3.2 Meta description

The current description is accurate and benefit-led. Trim to ~155 characters and lead
with the reliability promise:

```html
<meta name="description" content="Gemina VPN keeps your Mac online when one link drops — it sends your traffic over Wi-Fi and your Android phone's cellular at once. Open source, self-hostable.">
```

### 3.3 H1 / H2 structure

One H1 (already correct: the hero "Your connection never notices the dropout"). Ensure
H2s read as a logical, keyword-aware outline. Current section headings map well; keep:

- H1: *Your connection never notices the dropout* (hero)
- H2: *Duplicate, race, dedup.* — mechanism (good; could add a hidden-from-design but
  crawlable nearby phrase like "how internet failover works on macOS" in body copy)
- H2: *Built for staying up, not racing ahead.* — reliability-not-speed
- H2: *Run it yourself, or let us run it.* — open-core / pricing (TBD)
- H2: *A quick compatibility check…* — Android USB tethering
- H2: *Frequently asked questions* — captures question-shaped queries

Within body copy (not headings), naturally include the category phrases users search:
"internet failover", "Android USB tethering on Mac", "seamless failover", "no root, no
kernel extension". The FAQ already does this well.

### 3.4 Open Graph & Twitter cards

Add to `<head>`. Use a purpose-built social image (e.g. the link-monitor scope
graphic with the tagline) at `website/og-image.png`, ideally 1200×630.

```html
<meta property="og:type" content="website">
<meta property="og:site_name" content="Gemina VPN">
<meta property="og:title" content="Gemina VPN — seamless internet failover for macOS">
<meta property="og:description" content="Sends your Mac's traffic over Wi-Fi and your Android phone's cellular at once, so if one link drops your call, SSH or VPN never notices. Open source, self-hostable. Pre-release.">
<meta property="og:url" content="https://geminavpn.example/">
<meta property="og:image" content="https://geminavpn.example/og-image.png">
<meta property="og:image:alt" content="Two input signal traces — Wi-Fi and phone — merging into one unbroken output line.">

<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="Gemina VPN — seamless internet failover for macOS">
<meta name="twitter:description" content="Two links at once so one dropout never ends your call, SSH or VPN. Open source, self-hostable. Pre-release.">
<meta name="twitter:image" content="https://geminavpn.example/og-image.png">
```

Replace `geminavpn.example` with the real domain once chosen. Keep the OG
description honest: it includes "Pre-release."

### 3.5 JSON-LD structured data (`SoftwareApplication`)

Add a script block to `<head>`. Mark pricing as early-access/TBD by using a
`priceSpecification` with a `0` self-host price and **omitting** a concrete hosted
price (do not invent one). Do **not** add `aggregateRating` or `review` — we have no
real reviews and faking them is both dishonest and against schema guidelines.

```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  "name": "Gemina VPN",
  "applicationCategory": "UtilitiesApplication",
  "operatingSystem": "macOS",
  "description": "Open-source macOS reliability tool that sends your traffic over Wi-Fi and an Android phone's USB-tethered cellular link at the same time, delivering seamless failover so a dropped link never interrupts a call, SSH session or VPN. Pre-release: dual-path transport proven; encryption and the full app in development.",
  "url": "https://geminavpn.example/",
  "license": "https://www.gnu.org/licenses/agpl-3.0.html",
  "isAccessibleForFree": true,
  "softwareVersion": "pre-release",
  "releaseNotes": "Pre-release. Dual-path transport proven end-to-end; encryption and the shipping app are in development.",
  "offers": {
    "@type": "Offer",
    "price": "0",
    "priceCurrency": "USD",
    "description": "Self-host the gateway free and open source. An optional paid hosted gateway is planned (pricing to be announced; early access)."
  },
  "codeRepository": "https://github.com/joanmarcriera/gemina"
}
</script>
```

Optionally add a `FAQPage` JSON-LD block mirroring the on-page FAQ (the six existing
questions). Only include answers that are already on the page, verbatim in substance,
so the structured data and visible content match.

### 3.6 Image alt text

The inline SVGs already carry `<title>`/`aria-labelledby` and the brand mark is
`aria-hidden` — good. Rules going forward:

- Every meaningful diagram/illustration gets descriptive alt text built around the
  *concept*, not the filename: e.g. *"Diagram: Wi-Fi and phone each send a duplicate
  packet to the gateway, which forwards the first to arrive."* (the flow diagram's
  `<title>` already does this).
- Decorative marks stay `aria-hidden="true"` / empty alt.
- The OG image needs `og:image:alt` (added above).

### 3.7 Accessibility-as-SEO

Accessibility and SEO overlap heavily; the page already does most of this. Maintain:

- Single H1, logical heading order (no skipped levels).
- Skip link (present), keyboard-navigable nav, visible focus states.
- `lang="en-GB"` (present) — keep British English everywhere.
- `prefers-reduced-motion` respected for the animated scope (verify in `styles.css`).
- Colour contrast meeting WCAG AA on the dark theme.
- Form labels (`<label class="sr-only" for="wl-email">` present) and
  `aria-live` status (present) for the waitlist.

These improve crawlability, dwell time and Core Web Vitals (no layout shift, fast
render), all of which feed ranking.

---

## 4. Technical SEO for the static site

The site is framework-free static HTML/CSS/JS — already excellent for performance.
Concrete deliverables:

### 4.1 `robots.txt`

Place at site root (`website/robots.txt`):

```
User-agent: *
Allow: /
Sitemap: https://geminavpn.example/sitemap.xml
```

Do not block the blog or docs once published. Do not add fake disallows.

### 4.2 `sitemap.xml`

Root-level (`website/sitemap.xml`). Start minimal and grow as content ships:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemap.org/schemas/sitemap/0.9">
  <url>
    <loc>https://geminavpn.example/</loc>
    <changefreq>weekly</changefreq>
    <priority>1.0</priority>
  </url>
  <!-- add /blog/<slug>/ entries as articles publish -->
</urlset>
```

Automate generation if the blog is built by a static generator; otherwise update by
hand on each publish.

### 4.3 Canonical URLs

Add a self-referential canonical to every page to avoid duplicate-URL dilution (e.g.
trailing-slash vs not, query strings):

```html
<link rel="canonical" href="https://geminavpn.example/">
```

Pick one canonical host (www vs apex) and 301-redirect the other.

### 4.4 Page speed

Already framework-free, so the baseline is strong. Keep it that way:

- Inline-critical or single small CSS file (current: one `styles.css`); defer JS
  (current `app.js` is `defer` — good).
- Self-host any fonts; avoid render-blocking third-party requests.
- Compress and correctly size the OG/social image; use SVG for diagrams (already
  done).
- Target green Core Web Vitals (LCP < 2.5s, CLS ~0, INP low). No heavy hero media.
- Serve with HTTP caching headers and compression (gzip/brotli) at the host.

### 4.5 Mobile

`<meta name="viewport">` is present. Verify the scope/flow SVGs and plan cards reflow
on small screens; ensure tap targets ≥44px. The symptom audience ("call dropped on the
train") is heavily mobile, so mobile rendering is a ranking and conversion priority.

### 4.6 HTTPS

Serve everything over HTTPS with HSTS. Redirect HTTP→HTTPS. This is table stakes for a
*VPN/networking* product where trust is the entire pitch.

### 4.7 Structured data validation

Validate the JSON-LD with Google's Rich Results Test and Schema.org validator before
launch. Keep `offers` honest (no invented hosted price) and omit ratings/reviews until
real ones exist.

---

## 5. Content plan (8–12 articles)

Technical, credible, open-source-first. Each leans into the reliability/failover and
self-host angles, matches a real search intent, and stays honest about pre-release
status. Suggested slugs under `/blog/`.

| # | Title | Angle (one line) | Target keyword |
|---|---|---|---|
| 1 | **Why your video call drops on a MacBook — and how to stop it** | Top-of-funnel symptom piece that explains link dropouts and introduces failover. | `stop wifi dropping video calls mac` |
| 2 | **How internet failover actually works (duplicate, race, dedup)** | Mechanism explainer: packet duplication beats reconnection; diagrams from the site. | `how internet failover works` |
| 3 | **How USB tethering really works on macOS (RNDIS vs CDC-NCM)** | Deep, credible technical post on Android tether on Mac, no root, no kext. | `android usb tethering mac` |
| 4 | **Why your VPN drops in the lift — and how packet duplication fixes it** | Relatable scenario tied to the core mechanism; honest "encryption in development" note. | `why does my vpn drop in the lift` |
| 5 | **Keeping an SSH session alive across a network blip** | Developer-focused: why duplication survives drops that kill TCP/SSH reconnects. | `keep ssh session alive when wifi drops` |
| 6 | **Self-hosting a continuity gateway in one container** | High-intent, docs-adjacent walkthrough of the AGPL gateway; `docker run` from README. | `self host internet failover gateway` |
| 7 | **Reliability vs speed: why we duplicate instead of bonding links** | Sets honest expectations; pre-empts "will it make me faster?"; differentiates from bonding tools. | `bond wifi and cellular mac` |
| 8 | **Open-source alternatives for connection reliability on macOS** | Honest landscape piece (Speedify, Dispatch, engarde, glorytun) and where we fit. | `speedify alternative open source` |
| 9 | **Two independent WANs: proving a packet arrived over both Wi-Fi and cellular** | Engineering write-up of the proven dual-path result; build-in-public credibility. | `multipath udp reliability` |
| 10 | **No root, no kernel extension, no SIP changes: driving a phone tether from userspace** | The unusual technical achievement; appeals to macOS/security-minded developers. | `android tether mac no root` |
| 11 | **What "pre-release" means for an open-source VPN — what's proven, what isn't** | Transparency post that builds trust and ranks for brand + due-diligence queries. | `continuity vpn open source` |
| 12 | **A compatibility check before you commit: which Android + Mac combos work** | Practical guide around `geminactl preflight`; captures device-compat searches. | `is my android phone supported tethering mac` |

Editorial rules for every post: link to the GitHub repo and relevant `docs/`; show
real code/commands (from the repo, not invented); include the pre-release caveat where
relevant; never quote a benchmark we have not run; British English throughout.

---

## 6. Open-source-specific growth

The repo is itself a discovery surface and a trust signal. Treat GitHub SEO and
community launches as first-class channels.

### 6.1 GitHub topics & README SEO

- **Topics** (repo settings) — pick discoverable, accurate tags:
  `macos`, `vpn`, `failover`, `reliability`, `wireguard`, `android`, `usb-tethering`,
  `multipath`, `udp`, `self-hosted`, `open-source`, `golang`, `swift`.
- **Repo description** (the one-liner GitHub indexes): e.g. *"Seamless internet
  failover for macOS — sends your traffic over Wi-Fi and an Android USB tether at
  once. Open source, self-hostable. Pre-release."*
- **README** already has strong, honest first-paragraph framing — keep the bold
  one-line value prop at the very top (GitHub and search engines weight it). Add the
  website link near the top. Consider a short "Who is this for?" line mirroring the
  audience segments in §1.
- Add a `topics`-aligned social preview image in repo settings (same OG image).

### 6.2 Launch / community angles (honest, pre-release)

- **Show HN** — angle: *"Show HN: Gemina VPN — userspace Android USB tether on
  macOS, no root or kext, with proven dual-path failover (pre-release)."* HN respects
  technical honesty; lead with the proven engineering result and clearly label what is
  not done. Link the repo and the dual-path write-up (article #9).
- **r/selfhosted** — angle: the AGPL gateway as a single container you self-host;
  no accounts, no database. Lead with `docker run` and the self-host article (#6).
- **r/macapps / r/MacOS** — angle: the symptom ("calls drop on bad Wi-Fi") and the
  any-Android, no-root setup. Respect subreddit self-promo rules; pre-release framing.
- Secondary: **r/networking** / **r/homelab** for the multipath/UDP write-up,
  **Lobsters** for the userspace-RNDIS engineering post.
- Every launch must say "pre-release", point to the honest "what's proven / what
  isn't" article (#11), and avoid a "download now" call to action.

### 6.3 Awesome-lists & directories

Once there is a usable release (not before, to avoid burning goodwill), submit to:
`awesome-selfhosted`, `awesome-macos`, `awesome-go`, and relevant
networking/VPN/failover lists. Each listing is a quality backlink and a steady
discovery source. Until then, keep the repo and site polished so they are
submission-ready.

### 6.4 Comparison pages

Build honest comparison pages (also serve as articles #7/#8): "Gemina VPN vs
Speedify", "open-source connection-reliability tools compared". State what each
alternative does well, state our pre-release limitations, never publish unmeasured
numbers. These pages capture high-intent comparison searches and reinforce the
open-source trust story.

---

## 7. Measurement (no dark patterns)

Track discovery honestly; optimise for the right audience reaching the right
expectation, not for vanity numbers.

**Tools**

- **Google Search Console** (and **Bing Webmaster Tools**): submit the sitemap; watch
  impressions, clicks, average position and CTR per query. This is the primary,
  first-party signal.
- A privacy-respecting analytics option (e.g. self-hosted Plausible/Umami) rather than
  invasive tracking — consistent with the product's values.

**Metrics that matter**

- **Query coverage & rank** for the §2 clusters — are we appearing for *failover /
  reliability* terms (good) rather than *speed* terms (wrong audience)?
- **CTR per query** — honest title/description should earn clicks without clickbait.
- **Top landing pages and assisted paths** — which articles bring qualified visitors
  to the repo / self-host docs.
- **GitHub signals** — stars, repo traffic (referrers/clones), topic-driven discovery;
  treat as community-health indicators, not as marketing proof to display on-site.
- **Core Web Vitals** (Search Console) — keep LCP/CLS/INP green.
- **Conversion to genuine intent** — waitlist sign-ups and self-host doc visits,
  measured without countdowns, fake scarcity, or pre-ticked boxes.

**Explicitly avoid**

- Fabricated user counts, ratings, testimonials or benchmarks anywhere (site, schema,
  social).
- Dark-pattern waitlist tactics. The site already promises "we'll tell you the price
  before anything is ever charged" — keep that promise in every funnel.
- Ranking for speed/bandwidth terms by writing copy that implies aggregation.

**Review cadence:** check Search Console monthly; re-baseline the keyword clusters and
content plan each time a real milestone lands (encryption shipped, app shipped, hosted
tier priced) — and only then update claims, schema and the pre-release banners.
