# Licensing

This project uses an **open-core dual licence**. Two licences apply, chosen per
directory:

- The **gateway** (the server) is licensed under the **GNU Affero General Public
  License, version 3 only** — SPDX identifier `AGPL-3.0-only`.
- The **client** and the **shared core** are licensed under the **Apache
  License, version 2.0** — SPDX identifier `Apache-2.0`.

The full, verbatim licence texts live in
[`LICENSES/AGPL-3.0.txt`](../../LICENSES/AGPL-3.0.txt) and
[`LICENSES/Apache-2.0.txt`](../../LICENSES/Apache-2.0.txt). The root
[`LICENSE`](../../LICENSE) file carries a short summary and the directory map.

## Why this split

- **AGPL on the gateway** stops a competitor from taking the server, running it
  as a hosted service, and never contributing their changes back. The AGPL's
  network-use clause means anyone who offers the gateway to users over a network
  must offer those users the corresponding source. That protects the hosted
  service we intend to operate.
- **Apache-2.0 on the client** is required because AGPL software cannot be
  distributed on the Mac App Store; App Store distribution terms are
  incompatible with the AGPL's conditions. Apache-2.0 is App-Store-friendly and,
  unlike a bare permissive licence such as MIT, it carries an **express patent
  grant**, which matters for a product that ships networking and transport code.

## Directory to licence map

This map is authoritative. When in doubt about a file, locate it under one of
these prefixes.

### `AGPL-3.0-only`

- `cmd/gateway/`
- `internal/gateway/`
- Gateway-only assets under `deploy/` that contain gateway **server logic**
  (for example, the gateway service definition and its container/build assets).

### `Apache-2.0`

Everything the client ships or links, plus the shared core and tooling:

- `apps/macos/`
- `pkg/clientcore/`
- `pkg/protocoltypes/`
- `pkg/testkit/`
- `internal/protocol/`
- `internal/dedup/`
- `internal/transport/`
- `internal/paths/`
- `internal/platform/`
- `internal/diagnostics/`
- `internal/bootstrap/`
- `cmd/continuityctl/`
- the rest of the shared core and tooling

## Compatibility: Apache-2.0 into AGPL-3.0

Apache-2.0 is **one-way compatible** into AGPL-3.0. This means:

- The **AGPL gateway may include the Apache-2.0 core.** Combining Apache-2.0
  code into an AGPL-3.0 work is permitted; the combined gateway is then governed
  as a whole by the AGPL.
- The **client must never include AGPL code.** The flow does not run the other
  way: you cannot pull AGPL gateway code into the Apache-2.0 client without
  relicensing the client, which would break App Store distribution.

This holds in practice today because **the client never imports the gateway
packages.** The client depends only on the shared core; it has no build or
import path into `cmd/gateway/` or `internal/gateway/`.

## What a contributor needs to know

- **Know which side you are editing.** Use the directory map above. Code under
  the gateway prefixes is AGPL-3.0-only; everything else the client ships or
  links is Apache-2.0.
- **Never let AGPL code reach the client.** Do not add an import from any
  Apache-2.0 client/core package to a gateway package. The client must stay
  AGPL-free so it remains distributable on the Mac App Store.
- **Apache core can flow into the gateway**, not the reverse. If you need shared
  logic in both, it belongs in the Apache-2.0 core, and the gateway consumes it.
- **By contributing, you license your contribution under the licence that
  applies to the directory you are editing** (`AGPL-3.0-only` for the gateway,
  `Apache-2.0` elsewhere).
- **Third-party code** is governed separately. Nothing third-party is copied
  into product directories today; if that ever changes it must retain its
  original licence and attribution and be recorded in
  [`code-provenance.md`](code-provenance.md),
  [`upstream-licences.md`](upstream-licences.md) and the root `NOTICE`.

## SPDX identifiers

Use these exact SPDX licence identifiers:

- Gateway: `AGPL-3.0-only`
- Client and shared core: `Apache-2.0`

Per-file SPDX headers (for example, `// SPDX-License-Identifier: Apache-2.0`)
are **not yet present** in the source files. They will be added across the tree
in a follow-up change. Until then, this document and the directory map are the
authoritative record of which licence applies to which file.
