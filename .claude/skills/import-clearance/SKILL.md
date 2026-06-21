---
name: import-clearance
description: Run a per-file/subtree licence scan for a specific path BEFORE importing any upstream file into product directories. Use whenever about to vendor or copy third-party source, to satisfy the standing condition that root-file inspection is not import clearance.
disable-model-invocation: true
---

# Import clearance

`TASKS.md` standing condition: "Run a per-file/subtree licence scan of the
specific path before importing ANY upstream file (root-file inspection is not
import clearance)." This skill performs that scan for a given path and produces a
go/no-go record.

## Usage

Provide the upstream path you intend to import (a file or a subtree), e.g.
`import-clearance vendor-candidate/wireguard-go/tun`.

## Steps

1. Confirm the target exists in `.research-src/` (research material lives only
   there; it must never be imported directly into product dirs).

2. Run the project's structural gate:
   ```bash
   sh scripts/licence-check.sh
   ```

3. Per-file scan of the exact path (not just its root licence):
   ```bash
   # licence headers / SPDX identifiers in every file of the path
   grep -rniE 'SPDX-License-Identifier|GPL|AGPL|LGPL|MPL|Apache|BSD|MIT|copyright' <path>
   # files with NO licence header at all (often the riskiest)
   find <path> -type f \( -name '*.go' -o -name '*.c' -o -name '*.h' -o -name '*.swift' \) \
     -exec sh -c 'head -20 "$1" | grep -qiE "licen|copyright|SPDX" || echo "NO-HEADER: $1"' _ {} \;
   ```

4. Classify each licence found and check compatibility with the product (no
   GPL/AGPL into product dirs; MPL-2.0 carries per-file source-disclosure
   obligations; record attribution for permissive licences).

5. Write the outcome to `docs/legal/code-provenance.md` (or a new
   `docs/legal/import-<name>.md`): the path, every licence found, files lacking
   headers, the classification, and an explicit **CLEAR to import** /
   **BLOCKED** verdict with the reason.

## Guardrails

- A clean repository-root LICENSE does **not** clear the subtree. Scan every
  file in the path.
- Files with no licence header are blocking until provenance is established.
- This skill clears *licensing*; clean-room authorship of inspired cores is a
  separate condition — see the `clean-room-note` skill.
- Do not import anything until the verdict is recorded as CLEAR. Full legal
  review still governs any distribution decision.
