---
name: clean-room-note
description: Scaffold a clean-room provenance note BEFORE writing dedup/transport/RNDIS code informed by GPL sources (Engarde, OpenMPTCProuter, Linux rndis_host). Use when about to implement such logic, to satisfy the project's standing legal condition that clean-room notes are authored before the code.
disable-model-invocation: true
---

# Clean-room note

`TASKS.md` requires: "Author clean-room notes **before** writing any
Engarde/OpenMPTCProuter (GPL) inspired dedup/transport code; the reader of GPL
source must not author the corresponding core." This skill creates that note so
the discipline is recorded before a line of the core is written.

## Steps

1. Confirm the component (e.g. `dedup replay window`, `RNDIS host data plane`)
   and the public specification(s) it will be built from (e.g. MS-RNDIS, the
   project specification §8.2). The note must cite a spec, not a GPL codebase.

2. Create `docs/legal/clean-room-<component>.md` with this structure:

   ```markdown
   # Clean-room note: <component>

   Date: <YYYY-MM-DD>
   Author: <name>

   ## What is being implemented
   <one paragraph: the component and where it lives in the tree>

   ## Authorised sources (public specs only)
   - <spec name + URL/section>

   ## Prohibited sources (must NOT be read by the author above)
   - Engarde (GPL)
   - OpenMPTCProuter (GPL)
   - Linux drivers/net/usb/rndis_host.c (GPL)
   - <any other copyleft implementation of this component>

   ## Attestation
   I, the author named above, will implement <component> solely from the
   authorised public specifications listed here and have not read the
   prohibited sources. If that changes, I will stop and hand authorship to
   someone who has not read them.
   ```

3. Add a one-line pointer in `docs/legal/code-provenance.md` if that index
   exists. Do **not** paste any GPL source into the note.

4. Only after the note is committed should the corresponding core be written,
   with a provenance header in the source file (see
   `research/usb-rndis-spike/rndis_probe.c` for the pattern).

## Guardrails

- The note precedes the code. A note written after the fact does not satisfy the
  condition.
- One note per component; do not bundle unrelated cores under one attestation.
- British English in the prose (project docs convention).
