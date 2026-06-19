# Decisions

Last updated: 2026-06-19

This file records material project decisions for continuity across Codex sessions. Formal ADRs still need to be created during Stage 0 where required by `docs/product/project-specification.md`.

## 2026-06-17: Treat the Pasted Specification as Authoritative

Decision:

The complete pasted project specification is stored at `docs/product/project-specification.md` and is the authoritative starting point for implementation work.

Alternatives considered:

* Rely on conversation history only.
* Start implementing directly from the prompt.

Rationale:

The project is expected to span multiple sessions. The specification must be available from the repository before any implementation or delegated review begins.

Consequences:

Future sessions must read `docs/product/project-specification.md`, `AGENTS.md`, `PROJECT_STATE.md`, `TASKS.md` and `DECISIONS.md` before changing files.

Conditions for revisiting:

Revisit only if the user replaces the specification or explicitly approves a scope change.

## 2026-06-17: Next Implementation Work Is Stage 0 Only

Decision:

The next implementation session must perform Stage 0 repository bootstrap and source due diligence only. Stage 1 transport work is deferred.

Alternatives considered:

* Begin the dual-path UDP probe immediately.
* Build the macOS Network Extension first.
* Start with payment or entitlement work.

Rationale:

The specification requires legal, architectural and engineering controls before implementation. Stage 0 creates the repo structure, ADRs, provenance controls, manifests, CI and skeletons that prevent unsafe scope expansion.

Consequences:

No VPN transport, packet framing, deduplication, gateway or macOS Packet Tunnel Provider implementation should begin until Stage 0 exit criteria are complete and reviewed.

Conditions for revisiting:

Revisit only if the user explicitly reprioritises the project and accepts the resulting provenance and architecture risk.

## 2026-06-17: Continuity First, Not Aggregation

Decision:

The product scope is continuity and reduced packet loss through duplicate protected delivery over Wi-Fi and Android USB tethering, not aggregate bandwidth bonding.

Alternatives considered:

* Market or design the product as bandwidth aggregation.
* Start with Multipath QUIC, MASQUE or a multi-region VPN service.

Rationale:

The feasibility risk is whether macOS can reliably send independent protected traffic over two simultaneous paths and whether first-copy acceptance preserves user sessions. Aggregation adds scheduler and fairness complexity before the core reliability claim is proven.

Consequences:

The Stage 1 experiment must prove explicit per-interface UDP transmission, gateway deduplication and path-loss survival before the project invests in polished UI, payments or broader control-plane work.

Conditions for revisiting:

Revisit after Stage 1 and real train testing produce measurable evidence, or if the commercial proposition changes explicitly.

## 2026-06-17: Use Manual Xcode Creation Instructions for Stage 0

Decision:

Stage 0 creates macOS source directories and a SwiftPM compile scaffold, but does not generate or commit an Xcode project yet. `apps/macos/README.md` records exact manual Xcode creation steps.

Alternatives considered:

* Generate an Xcode project immediately.
* Skip macOS source scaffolding until Stage 1.

Rationale:

NetworkExtension signing, app group configuration, developer team identifiers and entitlements require owner review. A premature project would create fragile settings and false confidence. SwiftPM gives a lightweight compile check for the placeholder source without deciding signing details.

Consequences:

Swift CI currently builds the SwiftPM scaffold and runs SwiftLint when installed. Real XCTest and UI tests are deferred until the Xcode project and full Apple test toolchain are configured.

Conditions for revisiting:

Revisit when signing details, bundle identifiers, entitlements and the Apple Developer Team ID are known.

## 2026-06-17: Keep Research Sources Outside Product Directories

Decision:

Upstream research repositories are listed in `research/upstream-manifest.yaml` and fetched only into `.research-src/` by `scripts/fetch-research-sources.sh`. `.research-src/` is ignored by Git.

Alternatives considered:

* Vendor upstream repositories into the monorepo.
* Copy selected upstream files directly into product directories during bootstrap.

Rationale:

Stage 0 needs source due diligence without creating provenance or GPL contamination risk. Keeping sources in an ignored research workspace preserves a clear boundary between study material and product code.

Consequences:

Every future source import must update `docs/legal/code-provenance.md`, `docs/legal/upstream-licences.md` and `NOTICE`. Inspiration-only projects require clean-room notes rather than copied implementation.

Conditions for revisiting:

Revisit only if the project intentionally vendors a reviewed permissive dependency or changes distribution/licensing strategy.

## 2026-06-17: Local Stage 0 Swift Validation Uses Build-Only Checks

Decision:

The local `make test-macos` target runs `swift build` with workspace-local cache paths and `--disable-sandbox`.

Alternatives considered:

* Use `swift test` immediately.
* Omit local Swift validation.
* Require unsandboxed Codex execution for SwiftPM.

Rationale:

The local CommandLineTools Swift environment does not provide the `Testing` or `XCTest` modules, and SwiftPM's nested `sandbox-exec` is blocked inside the Codex sandbox. A build-only check still validates that the Stage 0 source scaffold and manifest compile without adding unsupported test assumptions.

Consequences:

Swift unit and UI tests remain explicit Stage 0 hardening tasks. CI currently performs Swift build and SwiftLint, not XCTest.

Conditions for revisiting:

Revisit when the Xcode project exists and the environment has a full Apple test toolchain.

## 2026-06-17: Fetch Research Sources by Exact Commit Only

Decision:

`scripts/fetch-research-sources.sh` initialises local repositories under `.research-src/`, fetches only manifest-pinned commits with `--depth 1 --no-tags`, checks out detached `HEAD`s and verifies that each checked-out `HEAD` exactly matches the manifest commit.

Alternatives considered:

* Use `git clone --no-checkout` and then fetch the pinned commit.
* Track upstream branches directly.
* Vendor research repositories into the product tree.

Rationale:

Stage 0 source due diligence needs reproducible research inputs without broad history fetches, branch drift or accidental source import. Exact Git object pins are the integrity anchors for the cloned research repositories.

Consequences:

The manifest must contain full lowercase 40-character Git commit IDs. Pending pins now fail `scripts/licence-check.sh`. Research clones remain ignored and are not product source.

Conditions for revisiting:

Revisit if a future supply-chain process needs signed tags, commit signature verification, source archives with checksums or a different dependency-management system.

## 2026-06-18: Validate Stage 0 from a Clean Workspace Copy

Decision:

Add `scripts/validate-clean-workspace.sh` and expose it as `make clean-workspace-check`. The script copies the repository to a temporary directory while excluding Git metadata, research clones, build caches, local Codex configuration and secret-like local files, then runs Stage 0 structural, licence, test and lint checks from the copy.

Alternatives considered:

* Rely only on checks from the working tree.
* Require a committed clean checkout before any clean validation can run.
* Run `make fetch-research` in the clean validation path.

Rationale:

The sandbox has unstable `.git` write permissions, but Stage 0 still needs evidence that the repository content can validate without local build artefacts or research clones. The clean-copy check gives repeatable pre-commit evidence without network fetches or Stage 1 behaviour.

Consequences:

`make clean-workspace-check` is now part of Stage 0 validation. It does not replace GitHub CI or legal review, and it intentionally avoids network source fetching by default.

Conditions for revisiting:

Revisit after the initial commit is pushed and CI runs from a real clean checkout.

## 2026-06-19: Allow Manual Dispatch for Stage 0 CI

Decision:

Add `workflow_dispatch` to the Stage 0 Go, infrastructure, licence and macOS CI workflows.

Alternatives considered:

* Rely only on path-filtered push and pull-request triggers.
* Remove path filters from every CI workflow.
* Add a separate all-in-one Stage 0 validation workflow.

Rationale:

The first repository push produced a GitHub Actions `startup_failure` run with no jobs or logs, even though the workflows were later visible and active. Manual dispatch keeps path-filtered CI efficient while giving Stage 0 an explicit clean-checkout validation control.

Consequences:

Maintainers can run each Stage 0 CI workflow on demand from GitHub or `gh workflow run`. Push and pull-request triggers remain path-filtered.

Conditions for revisiting:

Revisit if Stage 0 adopts a single required CI gate, if branch protection requires different status checks, or if the workflows become expensive enough to need stricter manual controls.

## 2026-06-19: Open Stage 1 Probe After Stage 0 Reviews

Decision:

Treat the independent Stage 0 engineering review and legal/provenance records review as sufficient to start the Stage 1 dual-path UDP probe, while keeping all source-import and legal-review restrictions in force.

Alternatives considered:

* Block all Stage 1 work until full legal sign-off for future imports.
* Start only after branch protection and PR-trigger checks are configured.
* Start Stage 1 with no reviewer follow-up record.

Rationale:

The reviews confirmed that no upstream implementation source has been imported, the Stage 0 repository controls and CI evidence are adequate, and the Stage 1 backlog is scoped to an evidence-producing probe rather than VPN product behaviour. The legal/provenance approval is explicitly limited to Stage 0 records and does not authorise source import.

Consequences:

Stage 1 may begin with the dual-path UDP probe. Any source import remains blocked until the per-file/subtree licence review, provenance updates and standing conditions in `TASKS.md` are satisfied. PR-trigger confirmation, branch protection and SwiftLint pinning remain hardening follow-ups before Stage 1 merges.

Conditions for revisiting:

Revisit if a reviewer reopens either Stage 0 gate, if Stage 1 needs upstream source import, if CI evidence regresses, or if legal counsel imposes stricter conditions before implementation.

## 2026-06-19: Start Stage 1 Below The Network Layer

Decision:

Implement the first Stage 1 slice as pure Go packet identity and first-copy duplicate suppression before adding macOS interface discovery, UDP socket binding or gateway networking.

Alternatives considered:

* Begin with live macOS interface enumeration and socket binding.
* Begin with a gateway UDP server.
* Combine identity, deduplication, path binding and gateway receipt in one larger probe.

Rationale:

The Stage 1 proof ultimately depends on packet captures and live path evidence, but the deduplication rule is a smaller invariant that can be unit-tested and race-tested without special network conditions. Building it first creates a stable core for later gateway receive loops while avoiding premature claims about Wi-Fi, Android USB tethering or VPN continuity.

Consequences:

`internal/protocol` owns probe packet identity. `internal/dedup` owns bounded in-memory first-copy acceptance. The current code can prove duplicate-suppression semantics only; it does not prove interface binding, gateway reachability, failover or encryption. Later Stage 1 work must connect real path observations to these types and collect packet-capture evidence before claiming success.

Conditions for revisiting:

Revisit if the live probe needs a different packet identity format, if authenticated replay protection changes the deduplication boundary, or if gateway tests show the bounded in-memory window is unsuitable even for the feasibility probe.

## 2026-06-19: Classify Paths From Observed Link Kinds

Decision:

Model Stage 1 path candidates from typed observations with explicit link kinds, rather than deriving Wi-Fi or Android USB tethering roles from BSD interface names such as `en0`.

Alternatives considered:

* Match known macOS interface names directly.
* Parse display names inside the core classifier.
* Wait for live macOS APIs before modelling path candidates.

Rationale:

macOS interface names vary by hardware, adapter order and tethering state. Treating names as identifiers rather than classification rules keeps the core testable and avoids baking one machine's naming into the product. A later Darwin adapter can populate `LinkKind` from appropriate system evidence while the pure Go classifier stays fixture-driven.

Consequences:

`internal/paths` can select Wi-Fi and Android USB tethering candidates from injected observations and can reject missing or ambiguous candidates. It cannot yet discover live macOS paths or prove egress. Future Darwin adapter work must define the system evidence used to assign link kinds and must still collect packet captures before any dual-path success claim.

Conditions for revisiting:

Revisit if live macOS evidence cannot reliably distinguish Android USB tethering from other USB network devices, or if the product needs to support multiple simultaneous Wi-Fi or tethering candidates.

## 2026-06-19: Use A Darwin Evidence Boundary Before Live Collection

Decision:

Add a Darwin-specific fixture boundary that maps injected `InterfaceSnapshot` records into `internal/paths.Observation` values before implementing live macOS collection.

Alternatives considered:

* Call live macOS APIs immediately.
* Parse command output directly inside `internal/paths`.
* Classify Android USB tethering from BSD interface names or display names.

Rationale:

The project needs a place to preserve macOS-specific evidence without letting the core classifier depend on one machine's interface names. The fixture boundary lets tests prove that BSD names are data only, that link kind must be explicit, and that Android USB tethering should require USB association evidence before a live collector claims it.

Consequences:

`internal/platform/darwin` owns fixture snapshots and evidence metadata. This
decision established the boundary before live collection. The later conservative
BSD interface-state collector is recorded in the decision below; richer
SystemConfiguration, Network framework and IORegistry collection still remains
future work.

Conditions for revisiting:

Revisit if live macOS collection requires a materially different observation shape, if Android USB tethering cannot be distinguished from generic USB network devices, or if packet-capture evidence shows the selected interface role is wrong.

## 2026-06-19: Collect BSD Interface State Conservatively

Decision:

Add a conservative Darwin live collector using Go's standard `net.Interfaces`
API to populate BSD interface names, interface flags and IPv4 presence behind the
existing `InterfaceSnapshot` boundary.

Alternatives considered:

* Wait until SystemConfiguration, Network framework and IORegistry collection can
  be implemented together.
* Infer Wi-Fi and Android USB tethering from BSD names or display names.
* Start socket binding before live state collection exists.

Rationale:

The Stage 1 probe needs live interface state, but BSD names and display names are
not reliable role evidence. Collecting only flags and IPv4 presence creates a
useful live boundary while keeping role assignment dependent on stronger future
macOS evidence sources.

Consequences:

`internal/platform/darwin` can now list conservative interface snapshots from the
host, but `NetInterfaceSource` sets `LinkKindUnknown`. The collector records
state evidence rather than source IP addresses and still cannot prove Wi-Fi,
Android USB tethering, egress, packet capture evidence or path-loss survival.

Conditions for revisiting:

Revisit if standard-library interface state is insufficient for socket binding,
if live macOS evidence requires source-address metadata in a redacted form, or if
SystemConfiguration/Network framework/IORegistry integration changes the snapshot
shape.
