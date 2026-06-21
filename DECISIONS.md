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

## 2026-06-20: Derive Darwin Link Kinds Only From Explicit Evidence

Decision:

Add fixture-driven Darwin evidence rules that derive Wi-Fi from explicit Network
framework or SystemConfiguration interface-type evidence and Android USB
tethering from explicit Android USB IORegistry evidence.

Alternatives considered:

* Infer Wi-Fi or Android USB tethering from BSD interface names or display names.
* Treat any USB network adapter as Android USB tethering.
* Wait for live macOS collectors before defining evidence-to-kind rules.

Rationale:

The Stage 1 classifier needs a tested contract for the evidence that can assign
path roles, but live collection and socket binding are still separate risks.
Explicit evidence rules let the Darwin boundary accept redacted fixtures and
reject generic or conflicting evidence without depending on one machine's
interface names.

Consequences:

`internal/platform/darwin.LinkKindFromEvidence` can classify injected evidence
from future macOS collectors. Generic USB Ethernet, missing evidence and
conflicting Wi-Fi/Android evidence remain unknown, so `internal/paths` reports a
missing candidate instead of guessing. This still does not prove live
SystemConfiguration, Network framework or IORegistry collection, per-interface
UDP egress, gateway reachability or path-loss survival.

Conditions for revisiting:

Revisit if live macOS evidence uses different stable source keys, if Android USB
tethering cannot be distinguished from generic USB adapters without additional
signals, or if packet captures show that evidence-derived roles do not match the
actual egress path.

## 2026-06-20: Use Command-Backed Darwin Evidence Acquisition For The Probe

Decision:

Add a provisional Stage 1 evidence source that combines conservative BSD
interface state with redacted evidence reduced from `networksetup
-listallhardwareports` and `ioreg -r -c IOEthernetInterface -l` output.

Alternatives considered:

* Block live evidence acquisition until direct SystemConfiguration, Network
  framework and IORegistry API bindings are implemented.
* Parse full command output into diagnostics and redact later.
* Move directly to socket binding and packet captures.

Rationale:

The next Stage 1 risk is whether the project can connect real macOS interface
state to the fixture-backed evidence rules without leaking private hardware
metadata. Command-backed collection is a small, reversible probe step. It lets
tests verify the redaction contract before any socket binding or gateway work.

Consequences:

`internal/platform/darwin.LiveEvidenceInterfaceSnapshots` can now merge BSD
interface state with coarse Wi-Fi and Android USB evidence. The implementation
does not store MAC addresses, serial numbers, source IP addresses or raw
IORegistry product strings in `Evidence`. This is not a production API boundary
and does not prove packet egress. A later slice should expose a redacted
diagnostic command, then packet-capture work must still prove actual paths.

Conditions for revisiting:

Revisit when direct macOS API bindings are available, if command output differs
on the target macOS version, if Android tethering evidence is ambiguous, or if
privacy review requires a stricter diagnostic-data model.

## 2026-06-20: Expose Darwin Evidence Through A Redacted Diagnostic Command

Decision:

Add `continuityctl darwin-evidence` to print a redacted JSON Stage 1 evidence
report built from `LiveEvidenceInterfaceSnapshots`.

Alternatives considered:

* Keep evidence capture as library-only until socket binding exists.
* Print raw `networksetup`, `ioreg` or interface address output for manual
  review.
* Proceed directly to UDP socket binding after library tests.

Rationale:

Before socket binding, the project needs a repeatable operator-facing check that
the target Mac can identify one usable Wi-Fi candidate and one usable Android
USB tethering candidate. JSON output is easier to archive and compare than
terminal prose, but it must remain redacted and must not overstate success.

Consequences:

The diagnostic report includes BSD names, coarse interface state, coarse
evidence tokens, candidates and missing/ambiguous classification issues. It
omits display names, source IP addresses, MAC addresses, serial numbers, raw
access keys and raw IORegistry product strings. The report explicitly labels
itself diagnostic-only and not path success. Packet captures and gateway tests
are still required before any dual-path claim.

Conditions for revisiting:

Revisit if privacy review disallows BSD names in diagnostics, if operators need
a file-output mode with automatic redaction checks, or if the later socket probe
needs a different evidence report schema.

## 2026-06-21: Keep One Task List and a Lean Project State

Decision:

`TASKS.md` is the single ordered source of truth for outstanding work.
`docs/backlog/stage-1.md` keeps only the fixed Stage 1 objective and acceptance
evidence and points at `TASKS.md`. `PROJECT_STATE.md` is a lean cross-session
handover (current state, risks, blockers, last validation), not an append-only
log; full change history stays in Git.

Alternatives considered:

* Keep tracking tasks across `TASKS.md`, `docs/backlog/stage-1.md`,
  `PROJECT_STATE.md` risks/blockers and the specification backlog.
* Keep appending each cycle's blow-by-blow narrative to `PROJECT_STATE.md`.

Rationale:

Tasks had drifted across five surfaces with real duplication (for example the
SwiftLint pinning item appeared twice inside `TASKS.md`), and `PROJECT_STATE.md`
had grown to a 467-line log of CI run IDs and delegation anecdotes that every
session must re-read. AGENTS.md itself warns against repeatedly reproducing large
files. Consolidation lowers per-cycle context cost and removes conflicting
"next action" sources.

Consequences:

Future sessions add and tick tasks only in `TASKS.md`, and trim rather than grow
`PROJECT_STATE.md`. Other documents link to `TASKS.md` instead of re-listing
tasks.

Conditions for revisiting:

Revisit if the project adopts an external issue tracker as the source of truth or
if a different handover format is needed for multiple concurrent workstreams.

## 2026-06-21: Ring-Buffer Dedup Eviction and Shared Darwin Evidence Vocabulary

Decision:

Back `internal/dedup.Window` eviction with a fixed-size ring buffer (O(1) per
observation) and centralise the Darwin evidence key/value vocabulary into shared
constants referenced by both the live command sources and
`LinkKindFromEvidence`.

Alternatives considered:

* Keep the slice-shift eviction (O(n) per insert once the window is full) and the
  string vocabulary duplicated across `live_evidence.go` and `evidence.go`.
* Defer the dedup structure change until the sequence-aware replay window is
  designed.

Rationale:

The slice-shift eviction shifted the whole order slice on every insert at
capacity — the wrong shape for a dedup hot path the spec requires to be
lock-efficient and benchmarked. The evidence vocabulary was an implicit contract
spread across producer and consumer files, where a one-sided typo silently
downgraded a path to `LinkKindUnknown` with no error. Both are small, reversible,
test-backed changes that do not alter the probe's external behaviour.

Consequences:

Eviction is O(1) and benchmarked; a FIFO-order test guards correctness. The
evidence vocabulary has one definition with a test pinning producer↔consumer
agreement. The dedup window remains a feasibility structure, not production
replay protection.

Conditions for revisiting:

Revisit the dedup structure when the sequence-aware sliding replay window is
designed, and the evidence vocabulary when direct macOS API collectors introduce
new stable evidence keys.

## 2026-06-21: Acquire the Android Tether Uplink via Userspace RNDIS, Not a Kernel/DriverKit Driver

Decision:

The second uplink (Android USB tethering) will be brought into macOS by the app
itself, in userspace: claim the phone's RNDIS USB interfaces via IOUSBHost/libusb,
drive the RNDIS control + data planes in-process, and present the result to the
routing layer through an `NEPacketTunnelProvider`. We will not depend on a kernel
extension, a DriverKit (DEXT) networking driver, or any relaxation of System
Integrity Protection.

Alternatives considered:

* DriverKit USB-networking system extension that publishes a real NIC. Requires
  the restricted `com.apple.developer.driverkit.transport.usb` entitlement
  (Apple approval) and there is no broadly-available public DriverKit family for
  third parties to create an Ethernet NIC.
* External travel router presenting plain Ethernet (handover Option D). Reliable
  but ships a separate hardware dependency to the user — fails the "inside the
  app, minimal friction" goal.
* Phone Wi-Fi hotspot + a second Mac radio. Needs hardware the Mac does not have
  (one Wi-Fi chip) and reintroduces the Apple-Silicon USB-Wi-Fi driver gap.
* Park the project, treating Android tethering on macOS as unsupported.

Rationale:

macOS ships no RNDIS host driver, so the phone never becomes a `enX` NIC — but
`ioreg` confirmed the OnePlus 12R exposes a standard, unclaimed RNDIS function
(control class `0xE0`, data class `0x0A`). A userspace viability spike
(`research/usb-rndis-spike/`) opened the device, claimed both RNDIS interfaces,
and completed the `REMOTE_NDIS_INITIALIZE` handshake (`status=0`, `medium=802.3`)
from an unprivileged process **with SIP enabled**. The single risk that would
have made the in-app approach impossible — whether userspace can take the
interface at all — is therefore retired with direct evidence. This is the only
option that keeps the uplink acquisition inside a notarisable app with one-time
user consent and no extra hardware.

Consequences:

* Stage 1 evidence acquisition shifts from "find the `android-usb-tether` NIC" to
  "find the RNDIS USB function": `darwin-evidence` cannot currently see this real
  tether because it reads BSD interfaces and its IORegistry matcher also requires
  the literal token `android`, which this device (`OnePlus`/`KALAMA`/`RNDIS …`)
  does not carry. Both gaps are now tracked in `TASKS.md`.
* The product gains an RNDIS host implementation and a NetworkExtension target,
  both previously listed as not implemented.
* Remaining unknowns before shipping: the claim must be re-confirmed inside the
  App Sandbox (`com.apple.security.device.usb`), and the data plane
  (`SET OID_GEN_CURRENT_PACKET_FILTER`, DHCP over the bulk pipe, RNDIS data
  framing) must be built and proven with packet captures — the spike proves only
  the control handshake.
* Provenance constraint: the RNDIS host code is authored clean-room from the
  public MS-RNDIS protocol; Linux's GPL `rndis_host.c` must not be read by its
  author or pasted in.

Conditions for revisiting:

Revisit if the userspace claim fails inside the App Sandbox and the
`com.apple.security.device.usb` entitlement cannot lift it, if Apple withdraws
unprivileged USB interface access, or if the RNDIS data-plane throughput proves
unfit and a DriverKit driver becomes necessary.

## 2026-06-21: Target Mac App Store Packaging for a Bounded, Removable Footprint

Decision:

Distribute the macOS client through the Mac App Store. The NetworkExtension
packet-tunnel ships as a bundled *app extension* inside a sandboxed app; there is
no kernel extension, DriverKit driver or system extension. The app's entire
footprint is enumerated and the uninstall contract is documented in
`docs/product/footprint.md`.

Alternatives considered:

* Developer ID distribution outside the App Store. The NetworkExtension provider
  would then have to be a System Extension that persists in the system, needs
  user approval at install, and requires explicit deactivation at uninstall —
  inherently less clean.
* Defer the channel choice and build channel-agnostic.

Rationale:

The owner's stated priorities are a clean install/uninstall, minimal friction and
a non-intrusive product. App Store packaging gives the smallest footprint by
construction: a mandatory sandbox confines all state to one container, and an
app-extension provider removes with the bundle so there is no system extension to
approve or deactivate. The userspace-RNDIS decision already removed the only
reason we would have needed a kernel/DriverKit driver, so a sandboxed
App Store build is achievable rather than a compromise.

Consequences:

* The app must be sandboxed; USB access to the phone relies on
  `com.apple.security.device.usb`, whose viability under the sandbox is the open
  task gating this choice (`TASKS.md`).
* The packet tunnel must route only the test/gateway destination via included
  routes and never become the system default, both for the test rig and to keep
  the product non-intrusive.
* Uninstall MUST remove the VPN configuration (`removeFromPreferences()`) and
  keychain items, not just the bundle; an orphaned VPN profile is the failure
  mode this decision exists to prevent.
* If App Store review rejects the USB-tethering use case, the fallback is
  Developer ID + System Extension, with the documented sysext uninstall step.

Conditions for revisiting:

Revisit if the USB claim cannot be made to work under the App Sandbox, or if App
Store review rejects the use case, in which case fall back to Developer ID
distribution and update `docs/product/footprint.md` for the system-extension
uninstall path.

## 2026-06-21: Cabled Management Channel for Single-Mac Bonding Tests

Decision:

Test the dual-path / failover work on one MacBook Pro using a fixed interface
split: a cabled Ethernet service is the management lifeline (pinned first in the
network service order, never touched by the app), while Wi-Fi and the phone
USB-RNDIS are the two test uplinks. The protocol and helper scripts
(`scripts/snapshot-network.sh`, `scripts/restore-network.sh`) live in
`docs/dev/test-environment.md`.

Alternatives considered:

* Test over Wi-Fi alone and accept losing connectivity (including the Claude
  session) whenever a path-loss experiment runs.
* Use a second Mac as the test driver.

Rationale:

Path-loss tests deliberately break the test uplinks, so the operator needs an
out-of-band channel that survives. A pinned cabled service guarantees the default
route stays put regardless of which test uplink is up. Stage 1 is socket-bind
only and changes no global routing, so the risk is currently low; the protocol
exists so it stays low when the scoped tunnel is introduced.

Consequences:

Baselines are captured before a cycle and the service order is reasserted on
demand or on trouble with one command. When the NEPacketTunnelProvider lands, its
disable step is added to the restore script and its included routes must exclude
the management subnet.

Conditions for revisiting:

Revisit if testing moves to a dedicated rig or CI hardware where the operator's
own connectivity is not at stake.

## 2026-06-21: Gateway Deployed as a Source-Built Container Under systemd on the Host

Decision:

Deploy the Stage-1 probe gateway to the `oracle` VPS as a container, built
natively on the host from rsynced first-party source (no registry, no
cross-compilation), run under a systemd unit, and shipped by a single repeatable
script (`scripts/deploy-dev-gateway.sh`). The runtime image is distroless,
non-root, read-only rootfs.

Alternatives considered:

* Build the image on the Mac and push to a registry, then pull on the host.
  Rejected: the Mac has no Docker daemon, the host is arm64 (cross-arch friction),
  and a registry adds credentials and another moving part for a dev gateway.
* Run the binary directly under systemd without a container. Workable, but the
  container gives a clean, reproducible footprint and easy teardown.

Rationale:

The host is native arm64 with Docker, passwordless sudo, firewalld and rsync, so
building on the host is the least-moving-parts path and is genuinely repeatable —
re-running the script ships a new release. Building from rsynced source (with
`.research-src` excluded) keeps GPL research material off the server and the
build context first-party. systemd gives restart-on-boot and restart-on-crash.

Consequences:

* The deploy depends on host tooling (docker, systemd, firewalld, passwordless
  sudo); documented in `docs/dev/gateway-deploy.md`.
* The host firewall is opened by the script, but the Oracle Cloud VCN security
  list is a separate cloud-level ingress filter that must be opened from the
  console (no OCI CLI/credentials on the host). Until then the gateway is
  reachable on-host but not from the internet; on-host end-to-end dedup is
  verified.
* No registry means no image provenance trail beyond the source tree; acceptable
  for a dev gateway, revisit for any production/distributed build.

Conditions for revisiting:

Revisit when multiple hosts or reproducible signed images are needed (introduce a
registry and CI build), or if the gateway moves to managed infrastructure defined
in `deploy/tofu`.
