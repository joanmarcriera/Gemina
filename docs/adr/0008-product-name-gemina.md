# ADR-0008: Product name — Gemina

Date: 2026-06-26

## Status

Accepted

## Context

The product was provisionally called **Continuity**. That name collides directly
with Apple's own cross-device brand (Handoff / Continuity Camera / "Continuity"
features), which is a real App-review, trademark and brand-dilution risk for a
macOS networking product that ships outside, and may later inside, Apple's
ecosystem. A name had to be chosen before any Apple App ID, provisioning profile,
signing certificate or marketing existed — renaming is cheapest before those
artifacts are created, and the Network Extension's bundle ids / app group are
exactly what a rename changes.

The product's mechanism is specific: it sends every packet down two independent
paths at once and the gateway keeps the first copy to arrive, discarding the
duplicate. The name should carry that idea.

## Decision

The product is named **Gemina**.

*Gemina* is Latin for "twinned". A *Legio Gemina* was a Roman twin legion formed
by merging two understrength legions into one full-strength unit (e.g. Caesar's
*Legio X Gemina*, and *Legio XIII / XIV Gemina*, reconstituted after Actium) so
the force still arrived at full strength. It is the exact metaphor for sending a
packet down two paths so the message arrives even if one path is lost.

The rationale is surfaced to users in the app's *About Gemina* dialog and in the
README "The name" section.

## Alternatives Considered

* **Keep "Continuity"** — rejected: Apple-brand collision (the reason for the ADR).
* **Angaros / Angareion** — the Persian royal relay courier that always got
  through ("nothing mortal travels faster… neither snow nor rain"); a strong
  guaranteed-delivery story, but an obscure, hard-to-pronounce brand.
* **Diaulos** — Greek "double course / two channels"; the most literal, but it is
  an athletic/musical term, not a redundancy-of-forces one, and reads awkwardly.
* **Tessera** — Polybius's watchword token, relayed and returned to confirm
  delivery (a built-in ACK); a great networking analogy, but it collides with
  existing "Tessera" networking/LED software, Web3 and semiconductor trademarks.
* Earlier shortlist (Twinpath, Bonded, DualLink, Splice, Throughline, Conflux,
  Holdfast) — descriptive but generic; none carried a story to tell.

## Rationale

Gemina has no software/trademark collision found, yields clean identifiers
(`com.joanmarcriera.gemina*`, module `github.com/joanmarcriera/gemina`, CLI
`geminactl`), and ships with a true, short historical story that maps precisely
onto the dual-path / first-copy-wins design — useful for both brand and docs.

## Consequences

* Full rename across the brand/Apple surface (Swift, `PRODUCT_NAME`, bundle ids,
  app group, NE principal class, entitlements) and the internals (Go module path
  and ~86 imports, `GEMINA_*` env vars, `geminactl` CLI, the `geminacore` cgo
  bridge, `gemina_*` Prometheus metrics, deploy paths, the GitHub repo).
* Two things are deliberately kept: the opaque `cc_*` C-ABI function prefix (an
  internal contract, not brand), and the two clientcore crypto domain-separation
  constants (`"continuity-vpn session key v1"`, `"continuity-vpn handshake v1"`) —
  protocol tags kept brand-independent and stable to avoid wire-compat churn.
* The lowercase word "continuity" is retained where it names the *design concept*
  (connection continuity), e.g. ADR-0001 "continuity-first".

## Conditions for Revisiting

A trademark conflict surfacing for "Gemina" in the relevant class/territory, or a
strategic rebrand of the product line.
