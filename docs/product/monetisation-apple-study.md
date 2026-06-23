# Monetisation strategy study: Apple-ecosystem options vs the current model

**Status: research note / decision input. Not legal advice.**
**Document date: 2026-06-24.** Written in British English.

> **RE-VERIFY BEFORE RELYING.** Every figure, rule and rate in this document is a
> snapshot taken on **2026-06-24** from the sources cited at the foot of each
> section. Apple's commercial terms, the App Review Guidelines, and the
> US/EU legal position **change frequently and are actively in litigation**
> (the *Epic v. Apple* anti-steering case was back in the US district court in
> April 2026 and Apple has signalled a Supreme Court appeal). Treat the
> percentages below as indicative, not contractual. Before you make a pricing,
> contractual or engineering commitment on the strength of this study,
> **re-read the primary Apple pages and the latest ruling** and confirm the
> numbers still hold. Do not quote these figures to customers or investors
> without that re-check.

---

## 1. What we are deciding

The project is open-core (see [`docs/product/monetisation.md`](monetisation.md)):

- **Client:** FOSS, Apache-2.0, intended for the **Mac App Store**.
- **Gateway:** FOSS, AGPL-3.0, self-hostable for free.
- **Revenue:** an **optional paid hosted gateway** subscription. The
  `internal/entitlement` scaffold already models this: signed, expiring,
  opaque entitlement tokens minted on a payment event, with a
  `PaymentProvider` interface designed for **Stripe (web)** and/or App
  Store / Play IAP. The hosted gateway enforces the token; the self-hosted
  path stays ungated and free.

The product is a **macOS** reliability/networking utility whose paid value is a
**hosted server subscription** — a thing that runs on our infrastructure and is
useful independently of the app, not a digital good "consumed in-app". That
distinction (Section 5) is the crux of whether we owe Apple anything at all.

The question this study answers: **for the paid hosted-gateway subscription,
should we bill via Stripe on the web, via App Store In-App Purchase (IAP), via an
external-purchase link, or some mix — and what does each cost us in fees,
control, friction, compliance risk and effort?**

---

## 2. Option A — App Store In-App Purchase / auto-renewable subscriptions (StoreKit 2)

**Mechanics.** Auto-renewable subscriptions are configured in App Store Connect
and sold in-app through **StoreKit 2** (the modern Swift API: `Product`,
`Product.purchase()`, `Transaction.currentEntitlements`, `Transaction.updates`).
Apple hosts the paywall transaction, takes payment with the user's Apple Account
on file, renews automatically, and exposes signed transactions the app verifies
on-device or via the App Store Server API. Subscriptions live in a *subscription
group*; a user holds at most one active level per group and can up/downgrade
within it.

**What Apple requires.** Under **Guideline 3.1.1**, if a digital service is
"unlocked or consumed within the app", payment **must** go through IAP — you may
not show your own credit-card form or (historically) steer to the web. For our
architecture, IAP would mean: on purchase, our backend maps the App Store
transaction (via App Store Server Notifications v2 / the Server API) to an opaque
subject and mints the same entitlement token the gateway already checks. The
`PaymentProvider` abstraction was built precisely so a StoreKit provider can sit
beside the Stripe one.

**User experience.** Lowest friction *for an in-app buyer*: native sheet, Apple
Account billing, trusted "Manage Subscriptions" UI, Ask-to-Buy/Family controls,
no card entry. The cost is the commission (Section 3) and that the customer
relationship (email, dunning, refunds, churn signals) is largely Apple's, not
ours.

> Sources: [Auto-renewable Subscriptions — Apple](https://developer.apple.com/app-store/subscriptions/);
> [App Review Guidelines — Apple](https://developer.apple.com/app-store/review/guidelines/);
> [What is StoreKit 2 — Qonversion](https://qonversion.io/blog/what-is-storekit-2-and-what-are-its-new-features).

---

## 3. Commission tiers (figures as of 2026-06-24 — re-verify)

| Tier | Rate | Applies to |
|---|---|---|
| Standard | **30%** | Year-1 auto-renewable subscriptions; standard IAP, large developers |
| Reduced (loyalty) | **15%** | Subscriptions **after 1 continuous paid year** for the same subscriber |
| **Small Business Program (SBP)** | **15%** | All IAP/paid-app proceeds for developers under the threshold |
| EU alternative-terms SBP, post year-1 subs | **10%** | Only under EU alternative business terms |

**Small Business Program detail.** Flat **15%** commission on paid apps and
IAP for developers (and their associated accounts) whose **proceeds were ≤ US$1
million in the prior calendar year**; you must also stay ≤ US$1M in the current
year or the standard rate resumes for future sales. You can re-qualify in a
later year if you drop back under. Enrolment is via App Store Connect (accept the
latest Paid Apps agreement / Schedule 2); adjusted proceeds take effect ~15 days
after the fiscal month in which enrolment is approved.

**Practical read for this project:** as a new, small project we would almost
certainly qualify for the **15% SBP rate** from day one — not 30%. The headline
"Apple takes 30%" is the *wrong* number for us; **15%** is the realistic IAP
cost, and it drops the loyalty/SBP question to a second-order concern.

> Sources: [App Store Small Business Program — Apple](https://developer.apple.com/app-store/small-business-program/);
> [The 15% App Store Fee (2026) — RevenueCat](https://www.revenuecat.com/blog/engineering/small-business-program/);
> [App Store Small Business Program guide 2026 — Adapty](https://adapty.io/blog/app-store-small-business-program/).

---

## 4. External purchase / link entitlements and the 2024–2026 rulings

This is the area in most flux. Snapshot as of **2026-06-24**:

**United States.** Following the **30 April 2025** ruling in *Epic v. Apple*
(Judge Gonzalez Rogers), Apple may **no longer charge a commission on purchases
completed on the developer's own website** reached via an in-app link, and may
**not** impose "anti-steering" friction (scare-screen interstitials, forced
formatting, extra logins) on those links. In the **US storefront** the
**StoreKit External Purchase Link Entitlement is not required** to include such
buttons/links — apps can link out to web checkout, and Apple currently takes
**0%** of those web purchases (the developer keeps the full amount, less their
own payment processor's fee).

**Caveat — actively litigated.** A federal appeals step held Apple should be able
to charge a "reasonable" fee but did **not** define one; the matter was back in
the district court around **April 2026**, with Apple pursuing a Supreme Court
appeal. **The "0% in the US" position could change.** Re-verify before relying.

**European Union (DMA regime).** Different and fee-laden. Under EU alternative
business terms, external/linked purchases can attract a **stacked** set of fees,
for example (2026): an **Initial Acquisition Fee ~2%** (new users, first 6
months), a **Store Services Fee** (Tier 1 ~5% / Tier 2 ~13%, or ~10% under SBP),
and a **Core Technology Commission ~5%** on digital-goods revenue (this 5%
replaced the old €0.50-per-install Core Technology Fee from **1 January 2026**).
Reports put the maximally-stacked EU take at **~20%**. So "link out and pay
nothing" is a **US** phenomenon, **not** an EU one.

**Reader-app rules.** A narrow class ("reader" apps — magazines, newspapers,
books, audio, music, video) may apply for the **External Link Account
Entitlement** to link to web account management. **Our service is not a reader
app** (a VPN/networking utility is not "previously purchased content"), so the
reader carve-out does **not** apply to us. We do **not** need it — the
US anti-steering position and the stand-alone-companion route (Section 5) are the
relevant levers, not reader status.

**What we may legally do to steer to Stripe and avoid the cut:** on the **US
storefront**, include a clearly-labelled in-app link/button to our Stripe web
checkout; on a **Mac** app this is even easier than iOS (see Section 5/7). The EU
storefront is where steering is taxed. Note these App-Store-Connect storefront
rules are written for iOS/iPadOS; the **Mac** App Store has historically been
more permissive about web sign-up, which materially helps us.

> Sources: [Apple anti-steering ruling / monetisation strategy — RevenueCat](https://www.revenuecat.com/blog/growth/apple-anti-steering-ruling-monetization-strategy/);
> [App Store fees 2026, EU/DMA — FunnelFox](https://blog.funnelfox.com/apple-app-store-fees-2026-eu-dma/);
> [External Purchase Link Entitlement — Apple docs](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.developer.storekit.external-purchase-link);
> [Distributing reader apps with a link — Apple](https://developer.apple.com/support/reader-apps/);
> [Daring Fireball: post-SCOTUS external-link guidelines](https://daringfireball.net/linked/2024/01/16/apple-guidelines-external-purchase-links).

---

## 5. The decisive question: "digital service consumed in-app" vs "service usable outside the app"

This determines whether IAP rules bite us **at all**.

- **Guideline 3.1.1** forces IAP when a digital service is unlocked/consumed
  *inside* the app.
- **Guideline 3.1.3(b)** ("multiplatform services") lets users *access* in the
  app what they bought elsewhere — **but only if the same items are also offered
  as IAP** in the app. So 3.1.3(b) alone does **not** let you avoid IAP; if you
  show a paywall/unlock in-app you generally still owe an IAP option.
- **Guideline 3.1.3(d)** ("stand-alone apps") is the one that matters for us:
  **free apps acting as a stand-alone companion to a paid web-based tool** (the
  guideline's own examples are VPN, cloud storage, email, web hosting, VOIP) **do
  not need to use IAP**, *provided there is no purchasing inside the app and no
  in-app call-to-action to buy outside it.*

**Why this fits us.** Our paid value is a **hosted gateway** — a server that runs
on our infrastructure and is configured by hostname; the client is a free,
open-source tool that merely *uses* it (and works fully against a self-hosted or
third-party gateway with no account at all). This is the textbook 3.1.3(d)
shape, and it is exactly how the VPN industry bills:

- **ExpressVPN** lets users be **billed by ExpressVPN (web account), not the App
  Store**; its 30-day money-back guarantee doesn't apply to App-Store-billed
  subscriptions — i.e. web billing is the primary path.
- **NordVPN** distinguishes web-account purchases from App-Store/Play purchases
  (e.g. referral perks only apply to direct purchases), again indicating web
  billing is first-class and IAP is an optional convenience layer.

The trade-off in the **strict** 3.1.3(d) reading is real: to skip IAP entirely,
the macOS app must contain **no in-app purchasing and no call-to-action** to buy
on the web. A user installs the free app, and a *separate* web flow (which the
app may not advertise inside itself) sells the hosted subscription. On the **US**
storefront, the 2025 anti-steering ruling additionally lets us include an
explicit in-app **link** to checkout (the stricter "no CTA" constraint is the EU/
non-US worry, and historically the Mac App Store is laxer than iOS here). The
risk is App Review's discretion: reviewers sometimes read an in-app paywall as
3.1.1 regardless of where the entitlement was granted. The mitigation is to keep
the app genuinely free and account-optional, with purchasing on the web.

> Sources: [App Review Guidelines — Apple](https://developer.apple.com/app-store/review/guidelines/) (3.1.1, 3.1.3(b), 3.1.3(d));
> [Apple Developer Forums: 3.1.3(b) multiplatform / stand-alone discussion](https://developer.apple.com/forums/thread/117077);
> [ExpressVPN — iOS in-app purchases](https://www.expressvpn.com/support/troubleshooting/ios-in-app-purchases/);
> [NordVPN billing & subscriptions](https://support.nordvpn.com/hc/en-us/categories/24552621609873-Billing-and-subscriptions);
> [Insights about Apple App Store rules for VPN apps — IVPN](https://www.ivpn.net/blog/insights-apple-app-store-rules-vpn-apps/).

---

## 6. TestFlight, trials/intro offers, family sharing

- **TestFlight** distributes pre-release builds to internal/external testers and
  is the standard beta channel before App Store review. It is **not** a sales
  channel: IAPs run in the **sandbox** (no real money), so TestFlight tests the
  StoreKit integration but doesn't monetise. It's an engineering/QA tool here.
- **Free trials / introductory offers.** If we *do* ship IAP, StoreKit supports
  **introductory offers** per subscription group: `freeTrial` (e.g. 1 month
  free), `payAsYouGo`, `payUpFront`. **One introductory offer per subscriber per
  subscription group.** Promotional offers can be layered for retention after the
  trial. If we bill via **Stripe** instead, we run trials ourselves (Stripe has
  native trial support), keeping full control of trial length, eligibility and
  win-back — without Apple's one-intro-offer constraint.
- **Family Sharing.** Auto-renewable subscriptions *can* be shared with a family
  group, but **only the family organiser may redeem offers**; other members get
  access through the shared subscription but can't independently redeem an intro
  offer. If our hosted gateway is *per-account/per-device-capacity*, Family
  Sharing of an IAP subscription could complicate entitlement accounting (one
  Apple subscription, several humans). A Stripe/web subscription sidesteps this:
  we define seats/sharing in our own terms.

> Sources: [Implementing introductory offers — Apple docs](https://developer.apple.com/documentation/storekit/implementing-introductory-offers-in-your-app);
> [Apple subscription offers guide 2026 — Adapty](https://adapty.io/blog/apple-subscription-offers-guide/);
> [What's new in StoreKit (WWDC25) — DEV](https://dev.to/arshtechpro/wwdc-2025-whats-new-in-storekit-and-in-app-purchase-31if).

---

## 7. AGPL / App Store interplay (confirmation)

No conflict, and nothing changes the licence split already recorded in
[`docs/legal/licensing.md`](../legal/licensing.md):

- **The client is Apache-2.0**, which is App-Store-compatible. Apache-2.0 carries
  an express patent grant (better than bare MIT for networking/transport code).
- **The gateway is AGPL-3.0 but is never shipped inside the app.** The App Store
  incompatibility is with **distributing AGPL software through the store**; the
  AGPL's network-use clause is satisfied by us offering the gateway's source to
  its network users, which we do. Since the client **never imports the gateway
  packages** (confirmed in `licensing.md` — the client depends only on the
  Apache-2.0 shared core), no AGPL code reaches the App Store binary. The gateway
  being AGPL is **irrelevant** to App Store distribution of the client.

One useful corollary: because this is the **Mac** App Store, we are **not locked
into it**. macOS apps can also be distributed **directly, notarised, outside the
store** (Developer ID), with **zero Apple commission** and full Stripe billing.
That is a strong fallback/parallel channel that iOS-only products don't have.

> Sources: in-repo [`docs/legal/licensing.md`](../legal/licensing.md) and
> [`README.md`](../../README.md) (licence split, client never imports gateway).

---

## 8. Comparison table

Figures as of **2026-06-24 — re-verify before relying.** Assumes we qualify for
the **Small Business Program (15%)**, which a new small project should.

| Dimension | **Stripe (web)** | **App Store IAP (StoreKit 2)** | **External-link → Stripe (US storefront)** |
|---|---|---|---|
| **Fee %** | ~2.9% + 30¢ (Stripe processing); **0% to Apple** | **15%** SBP (else 30% yr-1 / 15% yr-2) + Apple handles processing | **0% to Apple in US today** (litigated) + ~2.9% Stripe. **EU: stacked ~20%** |
| **Control over customer** | Full — we own email, dunning, churn, pricing, trials, refunds | Low — Apple owns billing relationship, refunds, much of the data | Full once on web; Apple owns the tap that leaves the app |
| **UX friction** | Higher in-app (leave app, enter card) — but Mac users are used to web sign-up | Lowest for in-app buyers (native sheet, Apple Account) | Medium (in-app link, then web checkout); smooth on Mac/US |
| **Compliance risk** | Low **if** app is a true free stand-alone companion (3.1.3(d)); risk is reviewer reading an in-app paywall as 3.1.1 | Lowest (Apple's own rail) but you accept the fee and lock-in | **Medium–high & volatile** — depends on the live anti-steering ruling, which is in active litigation; EU rules differ |
| **Effort** | Low–medium (Stripe + the existing entitlement webhook scaffold already targets this) | Medium–high (StoreKit 2, Server Notifications v2 mapping to entitlement tokens, sandbox/TestFlight, review) | Medium (Stripe **plus** entitlement checks **plus** storefront-conditional link logic + tracking the ruling) |
| **Self-host parity** | Identical — gate is hosted-only; self-host stays free/ungated | Same gate, but now two payment providers to reconcile | Same gate; link only shown where lawful |

---

## 9. Recommendation

**Keep Stripe as the primary and default billing rail for the hosted gateway, on
the web and for self-host/most signups. Do not build App Store IAP for v1.**

Rationale, on the evidence:

1. **We likely owe Apple nothing for the core model.** The product is a
   **stand-alone macOS companion to a paid web-based service** (hosted gateway) —
   the exact 3.1.3(d) shape Apple names VPNs under, and exactly how
   ExpressVPN/NordVPN bill. A free, account-optional client + web-sold
   subscription keeps us out of IAP's scope **legitimately**, not by a loophole.
2. **The fee we'd avoid is 15%, not 30%** (SBP), but 15% on a small subscription
   business is still material, and IAP also cedes the customer relationship,
   trial control and Family-Sharing accounting. Stripe keeps all of that.
3. **The existing scaffold already points at Stripe.** `internal/entitlement`'s
   `PaymentProvider` + webhook → token flow is built for this; Stripe is the
   shortest path to revenue with the least new surface.
4. **Mac-specific upside:** we can also distribute the notarised app **outside**
   the Mac App Store (Developer ID) with **0% commission**, so the store is a
   discovery channel, not a billing chokepoint.

**When to add an Apple path (revisit later, evidence-gated):**

- **App Store IAP** — only if analytics show meaningful **in-app conversion
  intent** that web checkout is losing, i.e. users who would convert in a native
  sheet but abandon a web hand-off. Then the **15% SBP** cost may be worth the
  conversion lift. Offer it *alongside* Stripe, not instead.
- **US external-purchase link** — a low-cost middle option: an in-app link to
  Stripe checkout on the US storefront at **0% Apple commission today**. Worth it
  **only** once the *Epic* litigation settles into a stable rule; until then it's
  volatile, and the EU stacking (~20%) means it must be **storefront-conditional**.

**Net:** Stripe-web for everyone now; treat IAP/external-link as a later,
data-driven optimisation for the specific in-app-conversion segment, gated on the
conversion lift exceeding the 15% (US) cost and on the litigation stabilising.

---

## 10. Concrete next steps

1. **Confirm the 3.1.3(d) posture in writing** before the app is built: the
   shipping Mac client must be **free**, fully usable against self-hosted/third-
   party gateways with **no account**, and contain **no in-app purchase and no
   in-app CTA to buy** (US storefront may add a neutral link post-ruling). Record
   this as a product constraint next to the entitlement design.
2. **Implement the Stripe `PaymentProvider`** against the existing
   `internal/entitlement` interface: real checkout-session creation, webhook
   signature verification, subject↔customer mapping, token minting/renewal,
   revocation on refund/chargeback (the gaps already listed in
   [`monetisation.md`](monetisation.md) §"What remains").
3. **Build the web purchase + token-delivery flow** (account/subscription
   storage, token fetch/refresh) — the parts `monetisation.md` flags as missing.
4. **Stand up a notarised Developer-ID build** as a parallel distribution channel
   (0% commission) alongside the Mac App Store listing.
5. **Instrument conversion** (web-checkout completion vs abandonment from the app)
   so the later IAP/external-link decision is **data-driven**, per Section 9.
6. **Schedule a terms re-verification** (owner action) before any public pricing:
   re-read the Apple primary sources below and the latest *Epic v. Apple* status;
   update this doc's figures and date.
7. **Keep IAP/external-link as a documented backlog item**, not v1 scope, with the
   trigger conditions from Section 9.

---

## 11. Re-verify-before-relying caveat and source list

**All figures captured 2026-06-24. Apple's terms and the US/EU legal position
change frequently and are in active litigation; re-verify against the primary
sources below before relying on any number here. This is a research note, not
legal advice — obtain qualified counsel before committing.**

Primary (Apple):
- App Store Small Business Program — https://developer.apple.com/app-store/small-business-program/
- App Review Guidelines (3.1.1, 3.1.3(b)/(d)) — https://developer.apple.com/app-store/review/guidelines/
- Auto-renewable Subscriptions — https://developer.apple.com/app-store/subscriptions/
- External Purchase Link Entitlement (docs) — https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.developer.storekit.external-purchase-link
- Implementing introductory offers — https://developer.apple.com/documentation/storekit/implementing-introductory-offers-in-your-app
- Distributing reader apps with a link — https://developer.apple.com/support/reader-apps/

Secondary / analysis (treat as commentary, verify against primaries):
- The 15% App Store Fee (2026), RevenueCat — https://www.revenuecat.com/blog/engineering/small-business-program/
- Small Business Program guide 2026, Adapty — https://adapty.io/blog/app-store-small-business-program/
- Apple anti-steering ruling / monetisation strategy, RevenueCat — https://www.revenuecat.com/blog/growth/apple-anti-steering-ruling-monetization-strategy/
- App Store fees 2026 EU/DMA, FunnelFox — https://blog.funnelfox.com/apple-app-store-fees-2026-eu-dma/
- Post-SCOTUS external-link guidelines, Daring Fireball — https://daringfireball.net/linked/2024/01/16/apple-guidelines-external-purchase-links
- 3.1.3(b)/stand-alone discussion, Apple Developer Forums — https://developer.apple.com/forums/thread/117077
- ExpressVPN iOS in-app purchases — https://www.expressvpn.com/support/troubleshooting/ios-in-app-purchases/
- NordVPN billing & subscriptions — https://support.nordvpn.com/hc/en-us/categories/24552621609873-Billing-and-subscriptions
- Apple App Store rules for VPN apps, IVPN — https://www.ivpn.net/blog/insights-apple-app-store-rules-vpn-apps/
- What's new in StoreKit (WWDC25), DEV — https://dev.to/arshtechpro/wwdc-2025-whats-new-in-storekit-and-in-app-purchase-31if

In-repo context:
- [`docs/product/monetisation.md`](monetisation.md), [`docs/legal/licensing.md`](../legal/licensing.md), [`README.md`](../../README.md), [`PROJECT_STATE.md`](../../PROJECT_STATE.md)
