# Monetisation financial model: revenue, limits, capex/opex, verdict

**Status: research note / decision input. Not legal, tax, or financial advice.**
**Document date: 2026-09-24.** Written in British English.

> **RE-VERIFY BEFORE RELYING.** Pricing figures below are snapshots taken
> **2026-09-24** from public vendor pages (cited inline) and this session's own
> read of the repo/docs. Cloud pricing, Apple's fee schedule, and payment-
> processor rates change; re-check before quoting a number externally.

This document fills the gap the other two monetisation docs deliberately leave
open. It does not re-litigate:

- **[`monetisation.md`](monetisation.md)** — the open-core architecture is
  decided: client + self-hosted gateway free forever; the only sellable thing
  is an optional paid **hosted gateway**, gated by the existing
  `internal/entitlement` scaffold (signed tokens, `PaymentProvider`
  interface, `ModeOpen`/`ModeHosted` gate). No real payment keys wired yet.
- **[`monetisation-apple-study.md`](monetisation-apple-study.md)** — billing
  rail decided: Stripe/web, not App Store IAP, because the product qualifies
  as a stand-alone companion to a paid web service (Guideline 3.1.3(d)), the
  same shape ExpressVPN/NordVPN bill under. Apple's realistic commission, if
  ever used, is **15%** (Small Business Program), not the 30% headline rate.

What follows is the numbers those two docs don't have: a real financial model
with revenue ranking, honest limits, capex, opex, and a breakeven-driven
GO/HOLD/NO call.

**Sequencing constraint that governs everything below:** this task is
downstream of <a href="https://familia.riera.co.uk/tasks/2562">#2562 BLOCKER:
WS-F on-hardware verification</a>, which has been stalled roughly three months
with no recorded progress as of 2026-09-24. There is no sellable hosted-gateway
product until WS-F lands. Every number here models what *should* happen once/if
it ships — it does not mean any of this should be built before then.

---

## 0. A note on what this session could and couldn't verify

Per the task brief, this session attempted to SSH directly into `oracle`
(the deployed Oracle Cloud gateway host, arm64, `uk-london-1`, per the
`gemina-gateway-ops` skill and `docs/dev/gateway-deploy.md`) to read live
instance shape, resource usage, and bandwidth off the box. **That SSH
connection timed out on port 22 from this environment** (confirmed with a
direct TCP check, not just an SSH auth failure — general internet egress from
this environment works fine, so the failure is specific to reaching that host/
port, most likely the Oracle Cloud VCN security list not permitting this
session's egress path, or the host being unreachable for another reason). This
was not chased further, per the working convention of stopping after 2–3 failed
attempts rather than thrashing.

**Consequence:** the capex/opex figures below use Oracle's *published* Ampere
A1 pricing and the specs already documented in this repo (an arm64 instance
running the probe-mode container today), not a live `df`/`free`/bandwidth
readout. If the real instance shape or usage differs materially from Oracle's
Always Free envelope (2 OCPUs / 12 GB RAM as of the June 2026 tier cut), the
hosting-cost line changes — **revalidate with a live SSH session before
treating the "hosting is ~free" conclusion below as fact**, since it is the
single most load-bearing assumption in the opex model.

---

## 1. Revenue paths — ranked, one decisive pick

| Path | Verdict |
|---|---|
| **Hosted-gateway subscription (Stripe/Lemon Squeezy)** | **GO — the primary path.** Only path the codebase/docs are already built for. |
| Direct Stripe/Lemon Squeezy checkout | Not a separate path — it's the *billing mechanism* for the path above. |
| Self-host support licence | **HOLD.** Selects an even smaller, already-payment-averse slice of an already-tiny population. Handle ad hoc via GitHub issues, don't productise. |
| App Store IAP subscription | **NO** — already correctly ruled out in `monetisation-apple-study.md`. Legally unnecessary under 3.1.3(d); would only hand Apple ~15% for nothing. |
| GitHub Sponsors | **Goodwill line, not revenue.** Already wired (`FUNDING.yml`) at zero marginal cost. Model it as a $0 assumption, not a forecast line. |
| Enterprise/ops-team tier | **HOLD.** Real pain point (SRE/field-ops teams on flaky links) but needs a B2B sales motion Marc doesn't have bandwidth for pre-launch. Revisit only if a team self-discovers and asks inbound. |

**Decisive recommendation: build the hosted-gateway subscription, billed
through Lemon Squeezy (not Stripe directly), wired to the existing
`internal/entitlement` scaffold — and stop there.**

Why Lemon Squeezy over raw Stripe for this specific launch: Marc's own
payments doctrine is that a Merchant of Record handles checkout + UK/EU/US
VAT/tax, and Stripe is used directly only where specifically needed — a
one-person project selling an international subscription is exactly the case
that doctrine exists for (no VAT-MOSS registration hassle, one line of
tax/compliance surface instead of fifty jurisdictions). `internal/entitlement`
already models `PaymentProvider` as an interface with a Stripe implementation
as one concrete instance; a Lemon Squeezy provider is the same shape of work,
and is cheaper in engineering-and-compliance-time even though its per-
transaction fee (Section 4) is nominally higher than Stripe's raw processing
fee.

Why this path and no other: (a) it is the only one the codebase and docs are
already architected for — the entitlement token, `PaymentProvider` interface,
and `ModeHosted` gate exist and only need real keys, account storage, and a
checkout page, which is *finish-the-job* work, not a new product direction;
(b) it fits solo-maintainer bandwidth — once wired it is "flip a switch after
WS-F ships," not an ongoing sales or support motion, unlike the enterprise or
support-licence paths; (c) given the narrow addressable market (Section 2),
no path here scales past niche/portfolio-income territory, so the right
question isn't "which one scales" but "which one is cheapest to switch on and
matches what the few real buyers will actually pay for" — and that is
convenience-hosting, not IAP, consulting, or enterprise sales.

---

## 2. The limits

### 2.1 Technical — how narrow is the addressable market, really

The chain of filters that bounds a *paying hosted-tier subscriber*:

1. **Owns a Mac.** macOS holds roughly 14–15% global desktop share (~30% in
   the US) as of mid-2026 ([Statcounter](https://gs.statcounter.com/os-version-market-share/macos/desktop/worldwide)).
   Apple last disclosed ~100M active Mac users in 2020; a reasonable current
   estimate is 100–200M active Macs worldwide.
2. **Also carries an Android phone specifically for USB tethering** — the
   *only* supported second-path mechanism today (no iPhone reverse-tether, no
   Bluetooth/hotspot fallback). No direct stat exists for "Mac owner who
   carries an Android," but ecosystem lock-in data (Mac buyers skew heavily
   iPhone via Continuity/Apple Watch/Handoff) implies Android-carrying Mac
   owners are a minority slice — plausibly 10–15% of the Mac base, i.e.
   ~15–25M people worldwide in the best case
   ([SQ Magazine iPhone vs Android](https://sqmagazine.co.uk/iphone-vs-android-statistics/)).
3. **Has the real pain point** — SSH/call continuity on flaky links
   (sysadmins, DevOps/SRE, digital nomads, maritime/rural workers). A thin
   professional sub-slice: generously 1–3% of (2).
4. **Willing to install a NetworkExtension tunnel and pay a subscription**
   rather than tolerate drops or use a free workaround (manual hotspot swap,
   a Tailscale exit node): a typical freemium/OSS paid-conversion rate,
   1–3% of (3).

Multiplying through (≈20M × 2% × 2%) lands at **low thousands of realistic
prospects globally, with actual paying subscribers plausibly in the low
hundreds to low thousands** — a real niche, not a rounding error, but nowhere
near a mass-market wedge. This ceiling does not move without the second-path
mechanism generalising beyond Android-USB-tether (a generic hotspot/Bluetooth
fallback, or an iPhone reverse-tether route) to capture the much larger
iPhone-owning Mac population — which is explicitly out of scope today.

### 2.2 Regulatory/legal

Once the gateway runs in **data+exit** mode (carrying real user tunnel
traffic, not just probe packets — the still-unvalidated Stage-2 mode gated
behind WS-F), two regimes apply, both **standard-SaaS-level, not a special
blocker**:

- **UK Investigatory Powers Act 2016.** The "telecommunications operator"
  definition is broad enough to arguably reach a small relay operator, with
  technical-capability-notice powers (s.253) theoretically in scope. In
  practice these notices target providers at national scale; a niche hosted
  relay is an unlikely enforcement target, but the classification risk is
  real enough to name explicitly in the ToS/privacy review.
- **UK GDPR.** The gateway is a processor of connection metadata (source/dest
  endpoints, timestamps) even without payload inspection — needs a documented
  retention policy, a DPA clause, and a lawful-basis note. Routine additions
  to a privacy policy, not a new compliance programme. Because Gemina markets
  on reliability, not anonymity, it avoids the sharpest VPN-specific legal
  exposure that has burned privacy-marketed VPNs: false "no-logs" claims.
- **Payments.** Lemon Squeezy as Merchant of Record absorbs VAT/tax
  registration; card-data PCI scope stays SAQ-A (the processor holds card
  data, not Gemina).

**Verdict: fold this into the legal review spend already planned
(<a href="https://familia.riera.co.uk/tasks/2568">#2568</a>) rather than
commissioning separate spend** — extend that review's brief by a few
paragraphs to cover "we run a UK-hosted relay touching connection metadata,"
rather than treating it as a new line item.

### 2.3 Platform — the real Apple number for the recommended path

With billing via Lemon Squeezy/web and no App Store IAP, **Apple's commission
on subscription revenue is ~0%.** The only fixed Apple cost is the
**$99/yr Developer Program fee** (already paid, recurring — see Section 4).

Residual risk: an App Review reviewer reads a web-billed subscription
mentioned anywhere in-app as requiring IAP anyway, despite the 3.1.3(d)
carve-out already argued in `monetisation-apple-study.md`. Rate this **medium**
— the guideline text is a genuine, citable exemption, not a stretch, but App
Review is known to be inconsistent on exactly this pattern, and apps that
surface a "subscription" or "hosted service" concept commonly draw an initial
3.1.1 bounce requiring one appeal citing 3.1.3(d). Budget review-cycle time
(one extra appeal round before approval), not extra dollars.

### 2.4 Market — honest demand read

Comparable products validate that *some* market exists for connection
continuity, but not cleanly at this exact positioning:

- **Speedify** (channel-bonding VPN, ~$14.99/mo) has survived roughly a
  decade as a real, if niche, paid product — proof of durable, if modest,
  willingness to pay for this category
  ([Security.org](https://www.security.org/vpn/speedify-vpn/)).
- **Tailscale** monetises teams/organisations, not individuals — its personal
  tier is free for up to 6 users, which suggests individual users default to
  free failover options rather than paying alone
  ([Tailscale pricing](https://tailscale.com/blog/pricing-v4)).
- **Peplink/Cradlepoint**-style hardware bonding routers sell into maritime/
  RV/enterprise verticals at hundreds-to-thousands of dollars — real budget
  exists there, but for a different buyer (appliance/enterprise procurement)
  than a $5–15/mo Mac consumer subscription.

**Blunt verdict:** this reads as a genuine but small niche SaaS opportunity —
plausible modest side income, closer to "meaningfully beats GitHub Sponsors"
than "replaces a salary" — bounded by the market-sizing ceiling in Section
2.1. It is **not** a "this doesn't clear the bar, HOLD everything beyond
Sponsors" conclusion (Speedify's decade of survival is real evidence people
pay for this category), but the honest framing is **side income, capped low,
not a business** — see Section 5 for the number.

---

## 3. Capex — one-off costs to stand up a real production hosted-gateway tier

| Item | Estimate | Notes |
|---|---|---|
| Hosting: bring the Oracle instance to data+exit mode | **~$0** | Same instance, no new resource purchase — `scripts/setup-exit-host.sh` + the data+exit systemd unit run on the box already provisioned. The actual blocking cost here is **engineering time to validate WS-F**, already tracked as <a href="https://familia.riera.co.uk/tasks/2562">#2562</a>, not new cash capex. |
| Payment integration dev time | **~15–25 hours of Marc's own time** | The scaffold (`internal/entitlement`, `StripeProvider`) is ~70% of the work already merged. Remaining: swap/add a Lemon Squeezy `PaymentProvider`, real keys, account/subscription storage, a web checkout + token-delivery page, webhook endpoint hosting, and end-to-end testing. Tracked as <a href="https://familia.riera.co.uk/tasks/2565">#2565</a>. Opportunity-cost only — no external contractor budgeted. |
| Legal review | **Rough order of magnitude: a few hundred pounds** for a solicitor or online legal service to review (not draft from scratch) the existing `docs/legal/privacy-policy.md` / `terms-of-service.md` drafts, extended per Section 2.2 to cover relay/metadata handling. Already flagged, not new: <a href="https://familia.riera.co.uk/tasks/2568">#2568</a>. **Get an actual quote before committing spend** — this is an estimate, not a quote. |
| Apple Developer Program enrolment | **$0 incremental** | Already paid, $99/yr — this is a **recurring opex** line (Section 4), not one-off capex, per the task's own framing. |
| Marketing spend | **$0 budgeted** | Go-to-market groundwork (`docs/marketing/`) is an organic/OSS-goodwill launch plan (Show HN, Product Hunt, social) — time cost, not cash spend. |
| Domain | **~$10–20/yr if a distinct commercial domain is registered** | No dedicated commercial domain found registered for this project yet (the `website/` static site references only `github.com`); if Marc wants a branded checkout domain separate from a GitHub Pages/existing domain, budget one registrar year. Recurring, folded into Section 4 for clarity. |

**Total one-off cash capex: roughly £300–800** (legal review, the only line
with real external spend), **plus ~15–25 hours of Marc's own engineering
time** valued as opportunity cost, not cash. There is no hosting capex to
speak of — the existing Oracle box, already paid for and already running
probe mode, absorbs the data+exit upgrade with a config change, not a
purchase.

---

## 4. Opex — recurring costs vs recurring revenue

### 4.1 Fixed recurring costs (independent of subscriber count)

| Item | Monthly cost | Source |
|---|---|---|
| Apple Developer Program | $99/yr ≈ **$8.25/mo** | [developer.apple.com/programs](https://developer.apple.com/programs/) |
| Domain (if a new one is registered) | ~$10–20/yr ≈ **$1–2/mo** | typical registrar pricing |
| Gateway hosting (base instance) | **~$0/mo**, if usage stays inside Oracle's Always Free envelope | See 4.2 |
| Monitoring (Prometheus/Grafana) | **$0/mo** | Already self-hosted on the same box (`scripts/deploy-monitoring.sh`), bound to localhost only, no extra resource purchase |
| **Fixed total** | **~$10–15/mo** | |

### 4.2 Variable cost per active hosted-tier user

**Compute.** Oracle's Always Free tier (cut 2026-06-15) gives 2 OCPUs / 12 GB
RAM total, 1,500 OCPU-hours and 9,000 GB-hours per month
([Oracle docs](https://docs.oracle.com/en-us/iaas/Content/FreeTier/freetier_topic-Always_Free_Resources.htm)).
A single Ampere A1 instance running 24/7 at 2 OCPU/12 GB consumes ≈1,460
OCPU-hours and ≈8,760 GB-hours/month — **just inside** the free allowance.
Growing past that (more subscribers needing more headroom) costs **$0.01 per
OCPU-hour + $0.0015 per GB-hour**: e.g. doubling to 4 OCPU/24 GB running 24/7
costs ≈$29/mo (OCPU) + ≈$26/mo (RAM) ≈ **$55/mo** for meaningfully more
headroom than the current probe-mode box needs. Cheap in absolute terms.

**Bandwidth.** Oracle made egress **free ($0/GB)** as of February 2026
([InfoQ](https://www.infoq.com/news/2026/07/oracle-cloud-free-tier-limits/)),
which — if it holds — makes the marginal bandwidth cost of relaying paying
users' tunnel traffic **effectively $0 at the infrastructure level today**.
This is the single biggest swing factor in this whole model and the most
important thing to re-verify before pricing a plan: Oracle's free-egress
policy is new (seven months old at the time of writing) and could change.
**Do not price a subscription assuming free egress is permanent — treat it as
a current subsidy, not a guaranteed unit economic.**

**Net read:** at the market ceiling estimated in Section 2.1 (low hundreds to
low thousands of subscribers), gateway hosting cost realistically stays
**under $100/mo** even after outgrowing the free tier, and could plausibly
stay at **$0/mo** for a long time if Oracle's free-egress policy holds and
usage stays modest.

### 4.3 Payment processor fees (Lemon Squeezy, recommended rail)

**5% + $0.50 per transaction**, all-in (includes VAT/tax handling globally)
([lemonsqueezy.com/pricing](https://www.lemonsqueezy.com/pricing)). For
comparison, Stripe direct is cheaper on paper — UK cards 1.5%+£0.20, EU/EEA
2.5%+£0.20, plus 0.7% of billing volume for recurring subscriptions
([stripe.com/pricing](https://stripe.com/pricing)) — but Stripe alone doesn't
absorb VAT/tax-registration obligations across jurisdictions the way a
Merchant of Record does, which is precisely why Marc's standing doctrine
routes checkout through Lemon Squeezy. Either way, this fee is **far below**
the 15% Apple IAP would cost for the same revenue (Section 2.3) — the
Stripe-vs-IAP-vs-Lemon-Squeezy comparison table in `monetisation-apple-study.md`
§8 still holds; this section just prices the actual chosen rail.

### 4.4 Marc's own time — the opportunity cost, stated explicitly

Per Marc's own portfolio-monetisation framing (side income, not salary
replacement), ongoing maintenance/support time for this tier has a real
opportunity cost even though no invoice arrives for it. Assume a
conservative **2–4 hours/month** of support/ops time once the tier is live
(entitlement issues, refunds, the occasional gateway restart, App Review
correspondence) at a nominal contractor-equivalent rate of **$50/hr** — that
is **$100–200/month** of opportunity cost, which dwarfs every cash opex line
above. This is the number the breakeven calculation in Section 5 has to clear
— not the hosting bill.

---

## 5. The verdict

**Assume a $8/mo price point** (mid-range of the $5–15/mo comparable-product
band from Section 2.4), billed via Lemon Squeezy.

- Lemon Squeezy fee: 5% × $8 + $0.50 = $0.90 → **net $7.10/subscriber/month**.
- Fixed cash opex: ~$10–15/mo (Section 4.1) — **breakeven on cash alone is
  ≈2 subscribers.** Trivial, and not the real bar.
- Marc's time opportunity cost: $100–200/mo (Section 4.4) — **breakeven
  including time is ≈15–28 paying subscribers.** This is the bar that
  actually matters.

Against the Section 2.1 market ceiling (low hundreds to low thousands of
realistic subscribers globally, no marketing spend assumed), **15–28
subscribers to clear time-opportunity-cost breakeven is plausible to reach**,
though not guaranteed, especially pre-marketing and pre-any-track-record.
Optimistic annual revenue at the top of the realistic subscriber range (low
thousands × $7.10 net, generously) tops out in the **$15k–$150k/yr gross**
range before churn and any marketing spend — consistent with the limits
agent's independent estimate.

**Capex payback:** the one-off cash spend (£300–800 legal review) pays back
at the ≈2-subscriber cash-breakeven point trivially fast — capex is not a
meaningful gate here. The real gate is the ~15–25 hours of dev time and,
above all, **the still-unresolved WS-F blocker**, which is not a financial
question at all.

### GO / HOLD / NO

**GO — but scoped and sequenced, not urgent.**

- **Build it:** the hosted-gateway subscription via Lemon Squeezy, wired to
  the existing `internal/entitlement` scaffold. The unit economics are
  genuinely favourable (net margin per subscriber is high; fixed costs are
  low-to-zero; capex is small) and there is real, if modest, comparable-market
  evidence (Speedify) that people pay for this category.
- **Don't build:** the self-host support licence or the enterprise tier
  (Section 1) — neither clears the bandwidth-vs-payoff bar for a solo
  maintainer at this market size.
- **Set expectations honestly:** this is a **side-income line capped in the
  low hundreds to low thousands of dollars per year in a good year**, not a
  business that replaces employment income. Frame it that way from the start
  so a modest outcome doesn't read as failure.
- **Sequencing is the actual constraint, not economics:** none of this
  matters until <a href="https://familia.riera.co.uk/tasks/2562">#2562
  (WS-F)</a> unblocks — do not spend the 15–25 hours of payment-integration
  time (<a href="https://familia.riera.co.uk/tasks/2565">#2565</a>) before
  the product exists to sell. The three-month stall on WS-F is the real
  bottleneck to first revenue, not anything in this document.
- **Re-verify before pricing publicly:** Oracle's free-egress policy
  (Section 4.2) is the single most load-bearing and most fragile assumption
  in this model — confirm it still holds at launch time, and confirm the
  live instance shape via a working SSH session (Section 0), before setting
  a price or promising a margin to anyone.

---

## Sources

- Oracle Always Free tier limits (2026-06-15 cut) — https://docs.oracle.com/en-us/iaas/Content/FreeTier/freetier_topic-Always_Free_Resources.htm
- Oracle free-tier egress change — https://www.infoq.com/news/2026/07/oracle-cloud-free-tier-limits/
- Hetzner Cloud pricing (reference only — not the gateway host, see Section 0) — https://betterstack.com/community/guides/web-servers/hetzner-cloud-review/
- Stripe pricing — https://stripe.com/pricing
- Lemon Squeezy pricing — https://www.lemonsqueezy.com/pricing
- Apple Developer Program — https://developer.apple.com/programs/
- Statcounter macOS market share — https://gs.statcounter.com/os-version-market-share/macos/desktop/worldwide
- iPhone vs Android ownership stats — https://sqmagazine.co.uk/iphone-vs-android-statistics/
- UK Investigatory Powers Act 2016, s.261/253 — https://www.legislation.gov.uk/ukpga/2016/25/section/261
- VPNs and GDPR — https://www.infosecurity-magazine.com/opinions/vpns-gdpr-compliant/
- Speedify pricing/positioning — https://www.security.org/vpn/speedify-vpn/
- Tailscale pricing — https://tailscale.com/blog/pricing-v4

In-repo context: [`docs/product/monetisation.md`](monetisation.md),
[`docs/product/monetisation-apple-study.md`](monetisation-apple-study.md),
[`PROJECT_STATE.md`](../../PROJECT_STATE.md), [`TASKS.md`](../../TASKS.md).
