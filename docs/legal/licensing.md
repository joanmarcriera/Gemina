# Licensing

From release 0.1.0 the whole repository is under one licence: the
**PolyForm Noncommercial License 1.0.0** (SPDX identifier
`PolyForm-Noncommercial-1.0.0`).

* Licence text: [`LICENSE`](../../LICENSE). It starts with a `Required Notice:`
  line that credits Riera Technologies Ltd.
* Plain-words summary and how to get a commercial licence:
  [`COMMERCIAL.md`](../../COMMERCIAL.md).
* Notices: [`NOTICE`](../../NOTICE).

This is not legal advice. Have a lawyer read the licence and
[`COMMERCIAL.md`](../../COMMERCIAL.md) before you sell a commercial licence.

## What the licence means

* Free for noncommercial use: personal, research, hobby, and use by charities,
  schools, public bodies and the like.
* Anyone who shares the software must keep the licence text (or link) and the
  `Required Notice:` line. That is the attribution.
* Commercial use needs a paid licence from Riera Technologies Ltd.
* The project is source-available. It is not OSI "open source", because it
  limits commercial use. Do not call it open source.

## Earlier releases

Before release 0.1.0 the repository carried a split licence: AGPL-3.0-only for
the gateway and Apache-2.0 for the client and shared code. That source was
already public. People who received it keep the rights those licences gave them
for that source, and nobody can take those rights back. The texts are in
[`LICENSES/pre-0.1.0/`](../../LICENSES/pre-0.1.0/).

The old split was chosen so the client could ship on the Mac App Store, which
cannot carry AGPL code. That reason no longer applies to new work, because the
copyright holder can distribute its own code on the Mac App Store under any
licence it chooses. The code boundary (the client must not import gateway code)
stays, as an architecture rule enforced by CI.

## Who holds the copyright

The `Required Notice:` names Riera Technologies Ltd. Every commit in the
history so far was authored by the project owner. No third-party source is
copied into the tree, and the Go and Swift code have no third-party
dependencies (see [`dependency-inventory.md`](dependency-inventory.md)), so no
dependency licence limits the choice. If the code was written by the owner
personally, a written assignment or licence to the company should exist before
commercial licences are sold.

## What a contributor needs to know

* By sending a contribution you agree that it is licensed under the same terms,
  and that Riera Technologies Ltd may also license it commercially.
* Do not add third-party code without recording it in
  [`code-provenance.md`](code-provenance.md),
  [`upstream-licences.md`](upstream-licences.md) and `NOTICE`. A dependency
  under a licence that forbids this project's terms (strong copyleft such as GPL
  or AGPL) must not be linked into the product.
* Keep the `Required Notice:` line in `LICENSE` and `NOTICE`.

## SPDX identifier

Use `PolyForm-Noncommercial-1.0.0`. Per-file SPDX headers are not present in the
source files. `LICENSE` is authoritative for every file.
