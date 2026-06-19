# AGENTS.md

This repository follows the global Codex execution and delegation policy below, plus the project-specific rules in `docs/product/project-specification.md`.

## Project Identity

Product: commercial macOS continuity VPN.

The first commercial release is a continuity product, not a bandwidth-aggregation product. It duplicates protected traffic over train Wi-Fi and Android USB tethering, sends both copies to a Hetzner gateway, accepts the first valid packet, and discards duplicates.

Authoritative specification:

`docs/product/project-specification.md`

Current stage:

Stage 1, dual-path UDP probe.

Stage 0 exit criteria have been met and reviewed. Stage 1 work must stay within
the evidence-producing probe scope until the UDP path, gateway deduplication and
path-loss survival evidence is recorded. Do not begin production VPN transport,
NetworkExtension packet handling, payment, entitlement or release work from
later stages until their gates are explicitly opened.

## Available Local Inference Resources

Two Ollama servers are available.

### Fast local GPU worker

* Provider name: `ollama_rtx4060`
* Agent name: `ollama_fast`
* Endpoint: `http://192.168.0.5:30068`
* Hardware: NVIDIA RTX 4060 with 8 GB VRAM
* Characteristics: relatively fast, but constrained model size and context
* Intended work: narrow reviews, file summarisation, test-case suggestions, documentation drafts, classification, extraction and other bounded tasks

Do not send this worker an entire large repository or a task requiring broad architectural context. Give it only the necessary files, diff or precise question.

### Larger local Mac worker

* Provider name: `ollama_mac`
* Agent name: `ollama_deep`
* Endpoint: `http://127.0.0.1:11434`
* Hardware: Apple Silicon MacBook Pro M5 with 48 GB unified memory
* Operational memory budget: approximately 35 GB for Ollama
* Characteristics: suitable for larger local models and deeper bounded analysis
* Intended work: multi-file analysis, implementation planning, test design, patch review and medium-complexity delegated reasoning

The 35 GB value is an operational budget, not a guaranteed hard memory limit. Avoid requesting unnecessarily large contexts. Do not load several large models concurrently.

## Remote Docker integration environment

A remote Linux integration server is available through the SSH alias:

* SSH alias: `codex-oracle`
* Remote user: `opc`
* Remote project directory: `/srv/codex/myproject`
* Docker Compose project name: `myproject`
* Environment type: disposable development and integration testing
* This is not a production environment

Use `scripts/remote` for normal remote operations:

* `scripts/remote check`
* `scripts/remote initialise`
* `scripts/remote sync`
* `scripts/remote build`
* `scripts/remote up`
* `scripts/remote status`
* `scripts/remote logs`
* `scripts/remote test`
* `scripts/remote down`

### Remote execution rules

* Use non-interactive sudo in the form `sudo -n`.
* Never use interactive `sudo`, `su`, or `sudo su -`.
* Do not assume that a remote shell persists between tool calls.
* For compound root operations, use `sudo -n bash -lc '...'`.
* Do not modify SSH configuration, users, sudoers, firewalls, cloud-init,
  boot configuration or Oracle Cloud settings unless explicitly instructed.
* Do not access or modify any directory outside `/srv/codex/myproject`
  unless explicitly instructed.
* Do not run `docker system prune`, delete unrelated volumes, or remove
  unrelated containers.
* Use the Compose project name `myproject` for all Compose operations.
* Do not store secrets, private keys or tokens in the repository.
* Do not copy the local `.env` file to the server.
* Bind test services to `127.0.0.1` unless public exposure is explicitly
  required.
* After deployment, verify container status, review logs and test the
  application endpoint.
* When a remote operation fails, collect evidence before changing the
  configuration.
* Before destructive remote operations, explain the exact affected resources
  and request approval.


## Delegation Policy

The primary Codex agent remains the orchestrator and final authority.

Before performing substantial work, identify whether a bounded portion can be delegated to a local Ollama agent without compromising correctness.

Prefer `ollama_fast` for:

* summarising selected files;
* identifying straightforward defects;
* drafting unit-test cases;
* extracting interfaces, dependencies or TODO items;
* reviewing a small diff;
* generating routine documentation;
* comparing a small number of implementation options.

Prefer `ollama_deep` for:

* reasoning across several selected files;
* producing an implementation plan;
* reviewing a substantial patch;
* suggesting refactoring steps;
* constructing test matrices;
* analysing logs and failure evidence;
* checking the primary agent's proposed approach.

Retain work in the primary Codex agent for:

* architecture and security decisions;
* destructive or irreversible changes;
* authentication, cryptography and secrets;
* dependency and supply-chain decisions;
* final integration;
* authoritative test execution;
* final code review;
* ambiguous requirements requiring user judgement.

Use delegation only when its expected value exceeds the cost of describing the task and consuming the result. Do not delegate trivial operations.

Do not spawn ordinary OpenAI-backed Codex subagents merely to reduce token usage. Such subagents consume additional Codex usage. Use the named Ollama agents when the purpose is to move suitable work to local inference.

Never delegate the complete project. Delegate small, verifiable work packages.

No more than one active task may use each Ollama endpoint at a time.

## Local-Agent Task Format

Every delegated request must contain:

1. one precise objective;
2. the relevant filenames or diff;
3. explicit exclusions;
4. the required output format;
5. the acceptance criteria;
6. a maximum response size.

Ask local agents for structured summaries, patch proposals or findings rather than long conversational explanations.

Treat local-agent output as untrusted engineering input. Verify important claims against source files, tests and authoritative documentation.

## Multi-Session Project Protocol

Assume substantial projects cannot be completed within one Codex usage window.

At the beginning of every work cycle:

1. read `PROJECT_STATE.md`, `TASKS.md` and `DECISIONS.md` when they exist;
2. inspect Git status and recent commits;
3. identify one bounded objective that can be completed and validated during the current cycle;
4. state the completion criteria before implementation;
5. avoid beginning additional work until the current objective is stable.

During a work cycle:

* keep changes small and reversible;
* run focused tests early;
* record material design decisions;
* use local Ollama delegation where it reduces primary-model work;
* avoid repeatedly reading or reproducing large files;
* refer to filenames, symbols and diffs rather than copying entire files;
* checkpoint progress before broad context or usage is exhausted.

After approximately 50 to 55 minutes of concentrated work:

1. stop starting new implementation work;
2. finish or safely revert incomplete edits;
3. run the relevant tests;
4. inspect the final diff;
5. update `PROJECT_STATE.md`;
6. update `TASKS.md`, including the next exact action;
7. update `DECISIONS.md` when an architectural choice was made;
8. record tests run, results, blockers and unresolved risks;
9. create an atomic commit when repository policy permits it;
10. provide a concise handover suitable for the next resumed session;
11. exit cleanly.

Do not execute a four-hour sleep command and do not remain idle inside the Codex process. An external scheduler is responsible for waiting and resuming.

## Required Project State

Maintain:

* `PROJECT_STATE.md`
* `TASKS.md`
* `DECISIONS.md`

These files are the durable source of truth across sessions. Do not rely solely on conversation history.

## Project Rules

Codex must:

* inspect the repository before editing;
* read this file and the current project state files;
* state which stage and issue it is addressing;
* make small coherent changes;
* run relevant tests;
* update documentation when behaviour changes;
* record architectural decisions;
* preserve licence headers;
* flag security uncertainty;
* use British English in documentation.

Codex must not:

* copy GPL implementation code into proprietary directories;
* translate GPL files line by line;
* ask an LLM to rewrite GPL files into this project;
* design new cryptographic primitives;
* introduce Kubernetes;
* add usernames, passwords or conventional customer accounts;
* add quotas or traffic-based billing;
* add aggregate bonding before continuity is proven;
* add a second client OS;
* hard-code macOS interface names;
* log secrets;
* store raw access keys;
* claim success without tests, packet captures or measurable evidence where applicable.

## Definition of Done

Every implementation issue must include:

* acceptance criteria met;
* unit tests;
* integration tests where applicable;
* race detector considered or run;
* fuzz test considered;
* documentation updated;
* threat model updated where applicable;
* metrics added where applicable;
* logs contain no secrets;
* licence and provenance checked;
* rollback considered;
* reviewer approval.
