# Terms of Service

> **DRAFT — NOT LEGAL ADVICE.** This document is a pre-release draft prepared for
> an open-source, pre-release product. It has **not** been reviewed by a
> qualified lawyer and must not be relied upon as it stands. Have a qualified
> legal professional review and adapt it for your jurisdiction before you publish
> it or rely on it. Placeholders such as `[Company name]` and `[contact email]`
> must be completed before launch.

**Last updated:** [date] · **Status:** pre-release draft

## 1. About these terms

Gemina VPN is a macOS reliability tool that sends a copy of your protected
traffic over two uplinks at once to a single gateway, which delivers the first
copy and discards the duplicate. The **client** and **gateway** are open source
and can be **self-hosted for free**. **[Company name]** ("we", "us", "our") also
intends to offer an **optional paid hosted gateway**.

These terms have two parts:

- **The open-source software** is governed by its open-source licences (Section
  2). These terms do not restrict the rights those licences grant you.
- **The paid hosted gateway service**, where and when we offer it, is governed by
  the additional terms in Sections 4 onward.

This is a **pre-release** project. The hosted service is **not yet available**;
the subscription terms below describe how it is intended to operate.

## 2. The open-source software and its licences

The software is open source under an **open-core dual licence**:

- The **gateway / server** is licensed under **AGPL-3.0-only**.
- The **client and shared core** are licensed under **Apache-2.0**.

Your rights to use, study, modify, self-host and redistribute the software are
defined by those licences, not by these terms. In particular, **you may
self-host the gateway for free** under the AGPL, with no account, payment or
entitlement token required. For the directory-to-licence map and the rationale,
see [`docs/legal/licensing.md`](./licensing.md) and the full licence texts in the
`LICENSES/` directory of the repository.

If anything in these terms appears to conflict with the open-source licence that
applies to a piece of software, **the open-source licence governs** for that
software.

## 3. Pre-release status and no warranty

This is **pre-release software**. Key parts — including encryption, the shipping
macOS app data path, and the hosted accounts/payment flow — are still in
development and **not yet complete**. The dual-path transport has been
demonstrated, but the product is not finished.

To the maximum extent permitted by law, the software and any hosted service are
provided **"as is" and "as available", without warranties of any kind**, whether
express or implied, including any implied warranties of merchantability, fitness
for a particular purpose, non-infringement, or uninterrupted or error-free
operation. We do not warrant that the service will keep any particular
connection alive, prevent any dropout, or be available at any particular time.
Nothing in this Section limits the warranty terms of the open-source licences as
they apply to the open-source software.

## 4. Acceptable use

Whether you self-host or use our hosted gateway, you agree not to:

- use the software or service for any unlawful purpose, or to send, carry or
  store unlawful content;
- attempt to gain unauthorised access to, disrupt, overload, or interfere with
  the service, its infrastructure, or other users;
- circumvent or tamper with the entitlement mechanism, share or forge entitlement
  tokens, or otherwise obtain paid access without paying for it;
- use the service to harm, harass, defraud or infringe the rights of others; or
- use the service in breach of any applicable law, export control, or sanctions
  regime.

We may suspend or terminate access to the **hosted** service for a serious or
repeated breach of this Section, as described in Section 7. (We cannot suspend a
gateway you self-host — that is yours to operate.)

## 5. The hosted subscription (when available)

The following apply to the **paid hosted gateway** once it is offered:

- **Early access.** The hosted service will launch as an **early-access**
  offering. It may change, be limited, or be withdrawn while in early access, and
  features may differ from those described on the website.
- **Pricing — to be determined.** Pricing is **not yet finalised**. We will tell
  you the price before any charge is ever made. Subscriptions, where offered,
  renew for successive periods until cancelled, and applicable taxes may be
  added.
- **Billing via Stripe.** Payments are processed by **Stripe**, our third-party
  payment processor. By subscribing you also agree to Stripe's applicable terms
  for the payment you make. We do not store your raw card details.
- **Entitlement.** Paid access is granted by a signed entitlement token that
  carries only an opaque subject id, a tier and an expiry — no personal data. The
  gateway admits a paying client by checking that token.
- **Cancellation.** You may cancel at any time; cancellation stops future
  renewals and takes effect at the end of the current paid period unless stated
  otherwise. Refunds, where offered, will follow the refund terms stated at
  purchase and any non-waivable rights you have under applicable consumer law.
  **[Specify refund/cancellation policy.]**

Because pricing and the payment flow are not yet live, the specifics in this
Section will be confirmed and may change before the hosted service launches.

## 6. Your responsibilities

You are responsible for: your own use of the software and service; keeping your
account credentials and any entitlement token secure; the lawfulness of the
traffic you send; and your own devices and connections (including any mobile-data
charges your carrier applies — the tool roughly doubles the data on protected
traffic, since it sends each packet over two links). If you self-host, you are
solely responsible for operating, securing and complying with the law in respect
of your own gateway.

## 7. Suspension and termination

You may stop using the software at any time, and may cancel a hosted subscription
as described above. We may suspend or terminate your access to the **hosted**
service if you breach these terms, if required by law, or if necessary to protect
the service or other users. We will give reasonable notice where practicable.
Termination of the hosted service does not affect your rights to the open-source
software under its licences.

## 8. Limitation of liability

To the maximum extent permitted by law, and except for liability that cannot
lawfully be excluded (such as for death or personal injury caused by negligence,
or for fraud):

- we will not be liable for any indirect, incidental, special, consequential or
  punitive damages, or for any loss of profits, revenue, data, or goodwill,
  arising out of or relating to the software or the hosted service; and
- our total aggregate liability arising out of or relating to the **hosted
  service** will not exceed the amount you paid us for that service in the **[12]
  months** before the event giving rise to the liability (and, where you have
  paid us nothing — for example because you self-host — our aggregate liability
  will not exceed **[a nominal amount, e.g. the greater of the fees paid or a
  small fixed sum]**).

Some jurisdictions do not allow certain limitations, so some of the above may not
apply to you; in that case our liability is limited to the smallest extent
permitted by law. This Section does not limit any rights granted to you under the
open-source licences.

## 9. Changes to these terms

We may update these terms as the product and the hosted service develop. We will
revise the "Last updated" date and, for material changes affecting the hosted
service, give notice by an appropriate means. Continued use of the hosted service
after a change takes effect constitutes acceptance of the updated terms.

## 10. Governing law

These terms, and any dispute arising out of or in connection with them or the
hosted service, are governed by the laws of **[governing-law jurisdiction]**, and
the courts of **[jurisdiction/venue]** will have **[exclusive/non-exclusive]**
jurisdiction, without prejudice to any non-waivable rights you have under the law
of your own country of residence. **[Confirm with counsel.]**

## 11. Contact

Questions about these terms: **[contact email]**.
Postal address: **[Company name], [registered/contact address]**.

---

*This is a pre-release draft for an open-source project and is not legal advice.
Have a qualified lawyer review it before relying on it.*
