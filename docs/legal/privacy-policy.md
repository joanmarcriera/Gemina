# Privacy Policy

> **DRAFT — NOT LEGAL ADVICE.** This document is a pre-release draft prepared for
> an open-source, pre-release product. It has **not** been reviewed by a
> qualified lawyer and must not be relied upon as it stands. Have a qualified
> legal professional review and adapt it for your jurisdiction before you publish
> it or rely on it. Placeholders such as `[Company name]` and `[contact email]`
> must be completed before launch.

**Last updated:** [date] · **Status:** pre-release draft

## Who this policy is about

Continuity VPN is a macOS reliability tool. It sends a copy of your protected
traffic over two uplinks at once — your Wi-Fi and an Android phone's cellular
link over USB — to a single gateway, which delivers the first copy and discards
the duplicate. The aim is continuity: if one link blips, the other is already
carrying the same packets.

The **client** and the **gateway** are open source and can be **self-hosted for
free**. We also intend to offer an **optional paid hosted gateway**, operated by
**[Company name]** ("we", "us", "our"), for people who would rather not run their
own server. This policy explains what data we, as the operator of the *hosted*
service, handle and why.

This is a **pre-release** project. Accounts, payments and the hosted entitlement
flow are **not yet live**; the descriptions below state how the hosted service is
designed to behave when it launches. We will update this policy before the hosted
service begins processing real personal data.

## The most important distinction: self-hosted vs hosted

- **If you self-host the gateway, we hold nothing about you.** A self-hosted
  gateway is software you run on your own infrastructure. We do not operate it,
  we receive no data from it, and we have no account, payment or log relating to
  your use of it. Your relationship is with your own server. (If you obtain the
  software through a third party such as a code-hosting platform or package
  registry, that third party's own privacy terms apply to that download.)
- **If you use our paid hosted gateway,** the limited data described below is
  handled by us and our payment processor so that we can bill you and run the
  service.

The rest of this policy concerns the **hosted** service only.

## What the hosted service collects, and why

We have designed the hosted service to handle as little personal data as
possible. Specifically:

### 1. Account and billing data (via Stripe)

To take a subscription payment we use **Stripe** as our third-party payment
processor. When you subscribe, **Stripe** collects and processes your payment
details (for example your card or other payment-method details and associated
billing information) directly. **We do not see or store your full card number or
other raw payment credentials.** Stripe acts as an independent processor /
controller for that payment data under its own terms; see "Third-party
processors" below.

From the billing relationship we hold the minimum needed to operate a
subscription, which may include: a billing/customer reference issued by Stripe,
your subscription status (active, cancelled, expired), and an email address used
for account and billing communication where one is provided. We use this **only**
to provide the service, take payment, prevent fraud and abuse, and contact you
about your subscription.

### 2. The entitlement token (carries no personal data by design)

Access to the hosted gateway is granted by a signed **entitlement token**. By
design this token carries only:

- an **opaque subject identifier** — an internal reference, **not** your name or
  email;
- a **tier** (which level of access the token grants); and
- an **expiry** time.

The token is cryptographically signed and contains **no personally-identifying
data**. The gateway checks the token to admit a paying client; it does not need
to know who you are to do so.

### 3. Minimal, redacted operational metrics from the gateway

The gateway's job is to deduplicate the two copies of your traffic and forward
the survivor. To run the service reliably we keep **coarse, redacted operational
signals** — for example counts of packets by decision (first-copy, duplicate,
rejected) and by path, and aggregate health metrics. These are designed to
contain **no IP addresses and no host identifiers**: the source address of
incoming traffic is deliberately kept out of the logging path, and logs record
only redacted, coarse decisions rather than identifiable detail.

## What the hosted service does **not** collect

By design, we do **not**:

- **Log the content of your traffic.** The gateway forwards traffic; it does not
  record what you send or receive.
- **Store IP addresses or host identifiers in our logs.** The logging and metrics
  paths are built to exclude them.
- **Put personal data in the entitlement token.** It carries only an opaque
  subject id, a tier, and an expiry.
- **Store your raw card details.** Payment credentials are handled by Stripe, not
  by us.
- **Sell your data**, or use it for advertising or profiling.

Once end-to-end encryption ships (it is still in development — see the project
status), the traffic itself is intended to be end-to-end between your client and
the gateway. We will describe the encryption guarantees accurately when they are
in place, and will not overstate them before then.

## Third-party processors

We rely on a small number of third parties to provide the hosted service. Today
the principal one is:

- **Stripe** — payment processing and subscription billing. Stripe collects and
  processes payment and billing data under its own privacy policy and terms. See
  Stripe's privacy documentation (for example
  <https://stripe.com/privacy>) for its role and how it handles that data.

We may also use standard infrastructure providers (for example a hosting/VPS
provider) to run the gateway. We will keep this list accurate and current as the
hosted service is built out. We do not sell personal data to anyone.

## Where data is processed

The hosted service and our processors may process data in one or more countries.
Where personal data is transferred across borders, we will rely on an
appropriate safeguard (for example the relevant standard contractual clauses or
an adequacy decision). The specific locations and safeguards will be stated here
before the hosted service launches. **[Specify hosting region(s) and transfer
mechanism.]**

## How long we keep data

We keep personal data only as long as we need it for the purposes above:

- **Account and billing records** — kept for the life of your subscription and
  afterwards for as long as we are required to keep them for legal, accounting
  and tax purposes, then deleted or anonymised. **[Specify retention period,
  e.g. tax-record retention.]**
- **The entitlement token** — short-lived; it expires by design and is not a
  long-term store of personal data (it contains none).
- **Operational metrics and redacted logs** — kept for a limited operational
  window for reliability, security and capacity purposes, then discarded or
  retained only in aggregate. **[Specify retention window.]**

## Your rights

If you are in the UK or the European Economic Area, the UK GDPR / EU GDPR gives
you rights over your personal data, including the rights to: access a copy of it;
have inaccurate data corrected; have it erased; restrict or object to certain
processing; and data portability. Where we rely on consent, you may withdraw it
at any time. You also have the right to complain to your data-protection
authority (in the UK, the Information Commissioner's Office).

Because the hosted service holds so little — and because the entitlement token
and operational logs are designed to contain no personal identifiers — much of
what we can act on relates to your **account and billing record**. Some payment
data is held by **Stripe** as processor/controller, so a request may need to be
directed to, or coordinated with, Stripe.

To exercise any right, contact us at **[contact email]**. We will respond within
the time required by applicable law.

## Children

The service is not directed at children and is not intended for use by anyone
under the age at which they can enter into a subscription in their jurisdiction.

## Changes to this policy

We will update this policy as the hosted service is built and before it begins
processing real personal data, and will revise the "Last updated" date above.
Material changes will be communicated by an appropriate means.

## Contact

Questions about this policy or your data: **[contact email]**.
Postal address: **[Company name], [registered/contact address]**.

---

*This is a pre-release draft for an open-source project and is not legal advice.
Have a qualified lawyer review it before relying on it.*
