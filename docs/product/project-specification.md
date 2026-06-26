The specification below assumes the first commercial release is a macOS continuity VPN, not a bandwidth-aggregation product:

* It uses train Wi-Fi and Android USB tethering simultaneously.
* It sends protected traffic over both paths.
* The VPS accepts the first valid packet and discards duplicates.
* The objective is continuity and reduced packet loss, not combined throughput.
* One prepaid £5 access key authorises one concurrent device.
* The first deployment uses one Hetzner VPS.
* macOS is the only client platform.
* The application is distributed directly, outside the Mac App Store.

The recommended implementation stack is:

Layer	Technology
macOS application	Swift 6, SwiftUI
macOS VPN integration	NetworkExtension, NEPacketTunnelProvider
macOS transport core	Go library compiled for Apple platforms, bridged into Swift
Cryptographic tunnel	WireGuard protocol and reusable WireGuardKit/wireguard-go components
Multipath continuity layer	New Go implementation inspired by Engarde
VPS gateway	Go on Debian
Control API	Go
Storage	SQLite initially
Infrastructure	OpenTofu/Terraform, cloud-init, Ansible or shell
Payments	Hosted checkout and webhooks
Metrics	Prometheus-compatible exporters and structured logs

Apple explicitly positions NEPacketTunnelProvider as the API for custom IP-level VPN clients and exposes the virtual interface through packetFlow.   WireGuard’s Apple client and Go implementation are both MIT-licensed and therefore suitable as reusable foundations for a commercial application, subject to retaining their notices.

Project Specification: Dual-Path Gemina VPN for macOS

1. Document purpose

This specification is the authoritative starting document for a new Codex coding session.

Codex must use it to:

1. establish the repository;
2. create the architectural documentation;
3. conduct open-source due diligence;
4. implement the transport prototype;
5. implement the macOS proof of concept;
6. deploy the first gateway on Hetzner;
7. progressively convert the prototype into a signed commercial macOS product.

Codex is not authorised to silently expand the scope. Any proposed scope change must be recorded as an Architecture Decision Record before implementation.

⸻

2. Product definition

2.1 Working product description

Build a macOS application that maintains Internet connectivity over two simultaneous upstream connections:

* public or train Wi-Fi;
* Android USB tethering.

The application captures the Mac’s IP traffic, encrypts it and transmits protected copies over both available paths to a remote gateway.

The remote gateway:

1. authenticates the client;
2. decrypts the protected transport;
3. identifies duplicate packets;
4. accepts the first valid copy;
5. discards later duplicates;
6. routes the original IP traffic to the Internet;
7. sends return traffic back using the same continuity strategy.

The first release optimises for:

* stable Teams, Zoom and Google Meet calls;
* persistent SSH connections;
* stable RDP and remote-working sessions;
* reduced disruption when one access network fails;
* simple operation by non-networking specialists.

It does not initially optimise for:

* aggregate download speed;
* torrenting;
* bulk transfer;
* gaming latency optimisation;
* anonymity against a sophisticated adversary;
* censorship circumvention;
* multi-hop VPN routing.

2.2 Product proposition

A continuity VPN that keeps calls and remote-working sessions alive by using Wi-Fi and mobile tethering together.

The product must not initially claim:

* “double your speed”;
* “combine all available bandwidth”;
* “zero packet loss”;
* “unbreakable connection”;
* “anonymous VPN”.

2.3 Commercial model

* Price: £5 for 30 days.
* No metered traffic allowance.
* No conventional user account.
* No username.
* No password.
* No email address required for technical access.
* Customer receives an opaque random access key.
* One key permits one active device lease.
* A second device may explicitly take over the lease.
* Device replacement does not require a customer portal.
* Lost keys are not recoverable unless a separate payment-recovery mechanism is introduced later.

2.4 Initial infrastructure

* One Hetzner VPS.
* One gateway location.
* IPv4 client traffic only.
* One fixed gateway embedded in the client configuration.
* SQLite for entitlement and session state.
* No Kubernetes.
* No service mesh.
* No separate message broker.
* No autoscaling.
* No multi-region orchestration.
* No customer dashboard.

⸻

3. Scope

3.1 Version 0: feasibility prototype

Version 0 proves that:

1. a Mac can identify both train Wi-Fi and Android USB tethering;
2. the client can send UDP traffic over each interface independently;
3. both paths reach the same VPS;
4. the VPS can deduplicate packets;
5. an active TCP or UDP flow survives loss of either path;
6. the improvement is measurable on a real train journey.

Version 0 may be CLI-based and may require administrative privileges.

3.2 Version 1: private alpha

Version 1 provides:

* Swift menu-bar application;
* Network Extension-based packet tunnel;
* access-key entry;
* Keychain storage;
* one fixed gateway;
* automatic interface discovery;
* dual-path duplication;
* single-device lease enforcement;
* basic diagnostics;
* signed development builds;
* manual installation;
* manual VPS deployment.

3.3 Version 2: private beta

Version 2 provides:

* signed and notarised macOS application;
* safe automatic reconnection;
* sleep/wake handling;
* network-change handling;
* captive-portal warning;
* secure application updates;
* hosted payment flow;
* key purchase and renewal;
* production monitoring;
* gateway deployment automation;
* support diagnostic bundle;
* documented privacy and acceptable-use policies.

3.4 Version 3: first paid release

Version 3 provides:

* stable one-click client;
* production entitlement service;
* one-device lease enforcement;
* gateway health monitoring;
* payment reconciliation;
* abuse controls;
* support runbooks;
* published licences and notices;
* independent security review;
* rollback-capable releases;
* basic disaster recovery.

3.5 Explicitly deferred

The following are outside the first paid release:

* Windows;
* Linux desktop;
* iOS;
* Android client;
* Mac App Store;
* aggregate bandwidth bonding;
* Multipath QUIC;
* MASQUE;
* multiple gateway regions;
* automatic best-region selection;
* split tunnelling;
* per-application routing;
* IPv6 client traffic;
* customer accounts;
* family plans;
* organisation plans;
* traffic dashboards;
* referral programmes;
* recurring subscriptions;
* cryptocurrency payments;
* Tor integration;
* port forwarding;
* static customer IPs.

⸻

4. Core architecture

4.1 Logical architecture

┌─────────────────────────────────────────────────────────┐
│ macOS application                                       │
│                                                         │
│ SwiftUI menu-bar UI                                     │
│ ├── access-key management                               │
│ ├── connection state                                    │
│ ├── path status                                         │
│ └── diagnostics                                         │
│                                                         │
│ Network Extension                                       │
│ ├── NEPacketTunnelProvider                              │
│ ├── packetFlow                                          │
│ ├── route configuration                                 │
│ └── DNS configuration                                   │
│                                                         │
│ Go transport library                                    │
│ ├── packet encapsulation                                │
│ ├── packet sequence identifiers                         │
│ ├── duplicate transmission                              │
│ ├── per-path socket binding                             │
│ ├── path health                                         │
│ ├── encryption/WireGuard integration                    │
│ └── server protocol                                     │
└───────────────┬───────────────────┬─────────────────────┘
                │                   │
        Greater Anglia Wi-Fi    Android USB
                │                   │
                └─────────┬─────────┘
                          │
                   encrypted UDP
                          │
┌─────────────────────────▼───────────────────────────────┐
│ Hetzner gateway                                         │
│                                                         │
│ Go gateway daemon                                       │
│ ├── session authentication                              │
│ ├── tunnel decryption                                   │
│ ├── duplicate suppression                              │
│ ├── packet replay protection                            │
│ ├── return-path duplication                             │
│ ├── client session table                                │
│ └── metrics                                             │
│                                                         │
│ Linux networking                                        │
│ ├── TUN                                                 │
│ ├── IPv4 forwarding                                     │
│ ├── nftables                                            │
│ └── NAT                                                 │
└─────────────────────────┬───────────────────────────────┘
                          │
                       Internet

4.2 Control plane

Customer payment
       │
       ▼
Hosted payment provider
       │ webhook
       ▼
Entitlement API
├── create key
├── extend key
├── revoke key
├── issue short-lived entitlement
└── enforce one active lease
       │
       ▼
SQLite

4.3 Trust boundaries

Treat these as separate trust zones even if they initially share one server:

1. macOS application;
2. Network Extension;
3. transport engine;
4. gateway data plane;
5. entitlement service;
6. payment webhook;
7. administrative interface;
8. observability system.

No gateway process may receive payment credentials.

No payment webhook may control Linux routing directly.

No raw customer access key may be stored in logs.

⸻

5. Technology decisions

5.1 macOS language

Use Swift for:

* application lifecycle;
* SwiftUI;
* menu-bar integration;
* Keychain access;
* NetworkExtension integration;
* system notifications;
* sleep/wake handling;
* user-visible diagnostics;
* automatic updates.

Do not write the macOS UI in Electron.

Do not use Python for the shipping client.

5.2 Shared networking language

Use Go for:

* packet framing;
* path handling;
* duplicate suppression;
* replay window;
* session protocol;
* gateway;
* entitlement API;
* command-line diagnostic utilities;
* network simulation tools.

Reasons:

* strong networking standard library;
* good concurrency model;
* easy Linux deployment;
* easy static server builds;
* direct availability of wireguard-go;
* Engarde is written in Go and can be studied;
* future mqVPN concepts can be translated or integrated more easily than if the core were written in Swift.

5.3 Apple integration boundary

The Swift Packet Tunnel Provider owns:

* NEPacketTunnelProvider;
* packetFlow;
* Network Extension lifecycle;
* Apple network settings;
* user/system events.

The Go core owns:

* transport state machine;
* packet identifiers;
* duplication;
* deduplication;
* encryption adapter;
* path liveness;
* protocol messages.

Define a narrow C-compatible boundary between Swift and Go.

Example conceptual API:

gemina_client_t *gemina_client_create(
    const gemina_client_config_t *config
);
int gemina_client_start(gemina_client_t *client);
int gemina_client_submit_packet(
    gemina_client_t *client,
    const uint8_t *packet,
    size_t length
);
int gemina_client_add_path(
    gemina_client_t *client,
    const gemina_path_config_t *path
);
int gemina_client_remove_path(
    gemina_client_t *client,
    const char *path_id
);
void gemina_client_stop(gemina_client_t *client);

The boundary must not expose Go pointers unsafely to Swift.

5.4 Cryptography

Do not design a new cryptographic primitive.

Preferred approach:

* reuse WireGuardKit and wireguard-go components where feasible;
* use established AEAD and key-exchange constructions;
* keep the continuity layer outside the cryptographic design;
* treat packet duplication as transport behaviour, not encryption behaviour.

The project must include a written cryptographic architecture before any claim of production readiness.

5.5 Database

Use SQLite for the initial control plane.

Use SQL migrations from the first commit.

Do not introduce an ORM that prevents inspection of generated SQL.

Suggested Go library:

* database/sql;
* a maintained SQLite driver;
* explicit repository layer;
* migration files checked into source control.

5.6 Infrastructure

Use:

* OpenTofu or Terraform for Hetzner resources;
* cloud-init for baseline operating-system configuration;
* Ansible or idempotent shell for gateway installation;
* systemd units;
* nftables;
* Debian stable;
* Caddy or nginx for HTTPS;
* Prometheus-compatible metrics.

Do not use Kubernetes in the initial project.

⸻

6. Source repositories

6.1 Repositories permitted for direct reuse

WireGuard Apple

Repository:

https://github.com/WireGuard/wireguard-apple

Purpose:

* study a production Network Extension implementation;
* reuse WireGuardKit where technically appropriate;
* study Swift/Go bridging;
* study tunnel configuration;
* study macOS packaging;
* study provider lifecycle.

Licence:

* MIT.

Rules:

* preserve copyright notices;
* preserve licence text;
* record exact commit;
* do not remove attribution;
* maintain a patch ledger for modifications.

wireguard-go

Repository:

https://github.com/WireGuard/wireguard-go

Purpose:

* cryptographic tunnel implementation;
* Go device architecture;
* packet queues;
* tunnel abstractions;
* Apple-compatible WireGuard bridge;
* logging and lifecycle patterns.

Licence:

* MIT.

Rules:

* prefer dependency or maintained fork over copied files;
* do not edit vendored source without recording the patch;
* preserve SPDX headers.

Hetzner libraries and providers

Repositories:

https://github.com/hetznercloud/hcloud-go
https://github.com/hetznercloud/terraform-provider-hcloud
https://github.com/hetznercloud/cli

Purpose:

* infrastructure automation;
* future gateway provisioning;
* API integration;
* reference implementation.

Prefer OpenTofu/Terraform provider use over writing a new Hetzner provisioning client.

6.2 Repositories permitted for reusable algorithms after review

Glorytun

Repository:

https://github.com/angt/glorytun

Purpose:

* packet framing inspiration;
* path management;
* sequence handling;
* failover behaviour;
* MTU handling;
* multipath tunnel concepts.

Licence:

* BSD 2-Clause.

Use:

* reference and potential selective reuse;
* not the first-choice production cryptographic layer;
* every copied component must retain attribution.

MLVPN

Repository:

https://github.com/zehome/MLVPN

Purpose:

* path health;
* multi-link configuration;
* monitoring interface;
* aggregation-server operations;
* failure recovery;
* source binding.

Licence:

* BSD-style permissive licence; verify exact repository licence before copying.

Use:

* primarily as operational and algorithmic reference;
* do not inherit its legacy cryptographic construction;
* do not assume its macOS support is production quality.

mptunnel

Repository:

https://github.com/greensea/mptunnel

Purpose:

* simple multipath UDP framing;
* packet sequence handling;
* server reconstruction;
* conceptual tests.

Licence:

* BSD 2-Clause.

Use:

* reference only unless a reviewed component is clearly useful;
* do not adopt the complete protocol unchanged.

6.3 Repositories for inspiration only

Engarde

Repository:

https://github.com/porech/engarde

Purpose:

* primary conceptual reference for Version 1;
* duplicate packet transmission;
* first-arriving-packet acceptance;
* multiple interface management;
* WireGuard-oriented continuity.

Licence:

* GPL-2.0.

Rules:

* do not copy GPL-covered implementation into proprietary components;
* do not translate files line by line;
* do not ask an LLM to “rewrite this file in our style”;
* create a behaviour specification from public documentation and black-box tests;
* implement independently from that specification;
* keep notes identifying the concepts studied;
* obtain legal review before distributing any Engarde-derived code.

OpenMPTCProuter

Repository:

https://github.com/Ysurac/openmptcprouter

Purpose:

* operational requirements catalogue;
* health checks;
* route policy;
* captive-portal behaviour;
* diagnostics;
* DNS controls;
* gateway administration;
* failover patterns.

Licence:

* GPL-based distribution.

Rules:

* use as requirements inspiration;
* do not copy platform-specific implementation into proprietary code;
* do not import GPL web assets into the commercial application.

mqVPN

Repository:

https://github.com/mp0rta/mqvpn

Purpose:

* future aggregate-bonding architecture;
* Multipath QUIC;
* MASQUE CONNECT-IP;
* scheduler designs;
* multi-user gateway concepts;
* possible Version 4 transport.

Licence:

* Apache 2.0.

Use:

* architecture reference during Version 1;
* maintain an abstraction that permits later transport replacement;
* do not introduce Multipath QUIC into the continuity MVP.

6.4 Source intake procedure

Create:

research/upstream/

Do not commit full upstream repositories to the primary product repository.

Create a script:

scripts/fetch-research-sources.sh

The script must clone pinned commits into a Git-ignored workspace:

.research-src/

Create:

research/upstream-manifest.yaml

Example:

sources:
  - name: wireguard-apple
    repository: https://github.com/WireGuard/wireguard-apple
    commit: "<pinned-sha>"
    licence: MIT
    permitted_use:
      - direct-reuse
      - modification
      - inspiration
  - name: engarde
    repository: https://github.com/porech/engarde
    commit: "<pinned-sha>"
    licence: GPL-2.0
    permitted_use:
      - behavioural-study
      - architectural-inspiration
    prohibited_use:
      - copy-into-proprietary-core
      - line-by-line-translation

⸻

7. Repository strategy

7.1 Use one monorepo initially

Repository name:

gemina

Do not split the project into multiple repositories until there are independent release cycles or access-control requirements.

7.2 Proposed structure

gemina/
├── AGENTS.md
├── README.md
├── LICENSE
├── NOTICE
├── SECURITY.md
├── CONTRIBUTING.md
├── CODEOWNERS
├── Makefile
├── go.work
├── .editorconfig
├── .gitignore
├── .swiftlint.yml
├── .github/
│   ├── workflows/
│   │   ├── go-ci.yml
│   │   ├── macos-ci.yml
│   │   ├── infra-ci.yml
│   │   ├── licence-scan.yml
│   │   └── release.yml
│   ├── ISSUE_TEMPLATE/
│   └── pull_request_template.md
│
├── apps/
│   └── macos/
│       ├── GeminaVPN.xcodeproj
│       ├── App/
│       ├── PacketTunnelExtension/
│       ├── Shared/
│       ├── UITests/
│       └── UnitTests/
│
├── cmd/
│   ├── gateway/
│   ├── entitlement-api/
│   ├── geminactl/
│   ├── packet-generator/
│   └── network-simulator/
│
├── internal/
│   ├── transport/
│   ├── protocol/
│   ├── framing/
│   ├── dedup/
│   ├── replay/
│   ├── paths/
│   ├── sessions/
│   ├── entitlement/
│   ├── gateway/
│   ├── metrics/
│   ├── logging/
│   └── platform/
│       ├── darwin/
│       └── linux/
│
├── pkg/
│   ├── clientcore/
│   ├── protocoltypes/
│   └── testkit/
│
├── bridge/
│   ├── include/
│   ├── darwin/
│   ├── build/
│   └── README.md
│
├── api/
│   ├── openapi.yaml
│   ├── protocol/
│   │   ├── messages.md
│   │   └── framing.md
│   └── generated/
│
├── db/
│   ├── migrations/
│   ├── queries/
│   └── fixtures/
│
├── deploy/
│   ├── tofu/
│   │   ├── modules/
│   │   └── environments/
│   │       ├── dev/
│   │       └── production/
│   ├── cloud-init/
│   ├── ansible/
│   ├── systemd/
│   ├── nftables/
│   └── docker/
│
├── observability/
│   ├── prometheus/
│   ├── grafana/
│   ├── alerts/
│   └── log-schemas/
│
├── tests/
│   ├── integration/
│   ├── end-to-end/
│   ├── network-chaos/
│   ├── compatibility/
│   └── performance/
│
├── scripts/
│   ├── bootstrap.sh
│   ├── fetch-research-sources.sh
│   ├── build-apple-bridge.sh
│   ├── deploy-dev-gateway.sh
│   ├── collect-diagnostics.sh
│   └── run-network-tests.sh
│
├── docs/
│   ├── architecture/
│   ├── adr/
│   ├── product/
│   ├── security/
│   ├── operations/
│   ├── testing/
│   └── legal/
│
└── research/
    ├── upstream-manifest.yaml
    ├── comparison-matrix.md
    ├── clean-room-notes/
    ├── protocol-notes/
    └── experiments/

7.3 Package boundaries

The following dependencies are allowed:

Swift UI
    ↓
Swift tunnel provider
    ↓
C bridge
    ↓
Go clientcore
    ↓
Go protocol/transport packages

The reverse dependency is prohibited.

The transport package must not import:

* database code;
* payment code;
* Swift UI concepts;
* Hetzner APIs;
* HTTP handlers.

The entitlement API must not import:

* macOS code;
* Packet Tunnel Provider code;
* Xcode-generated artefacts.

⸻

8. Protocol specification

8.1 Version 1 packet envelope

The continuity layer requires a packet identifier independent of the original IP packet.

Conceptual header:

version          uint8
message_type     uint8
flags            uint16
session_id       uint64
sequence_number  uint64
payload_length   uint16
reserved         uint16
payload          bytes
authentication   handled by cryptographic layer

Initial message types:

HELLO
HELLO_ACK
DATA
KEEPALIVE
PATH_PROBE
PATH_ACK
SESSION_CLOSE
ERROR

8.2 Deduplication

Each direction maintains an independent sequence space.

The receiver maintains:

* highest accepted sequence;
* sliding replay/deduplication window;
* bitmap or equivalent sparse structure;
* expiration timer;
* counters for duplicates, stale packets and invalid packets.

Required behaviour:

first valid copy:
    accept and forward
later copy within replay window:
    discard as duplicate
packet older than replay window:
    discard as stale
packet outside expected session:
    discard and record bounded metric

The deduplication implementation must be:

* bounded in memory;
* lock-efficient;
* benchmarked;
* fuzz tested;
* safe on sequence-number rollover;
* reset only during authenticated session re-establishment.

8.3 Path model

Each client path has:

path_id
interface_name
interface_index
source_ip
gateway_ip
transport_socket
state
last_send
last_receive
round_trip_time
loss_estimate
bytes_sent
bytes_received

States:

discovering
available
degraded
unavailable
captive
disabled

Version 1 behaviour:

* send each DATA packet over every available path;
* do not send over captive or unavailable paths;
* continue with one available path;
* reintroduce a recovered path automatically;
* do not reorder locally;
* let the server accept the first valid packet.

8.4 Return path

The gateway duplicates return traffic across all active client paths.

The client deduplicates before writing packets to packetFlow.

8.5 MTU

Start conservatively.

Initial proposed tunnel MTU:

1280 bytes

Do not optimise the MTU until:

* fragmentation tests exist;
* both IPv4 and encapsulation overhead are documented;
* train Wi-Fi testing has been completed.

Avoid relying entirely on ICMP Packet Too Big messages.

8.6 Session establishment

Initial sequence:

client -> entitlement API:
    access key + device public key
entitlement API -> client:
    signed short-lived entitlement token
client -> gateway path A:
    HELLO + token + path identity
client -> gateway path B:
    HELLO + token + path identity
gateway:
    verifies token
    creates or joins session
    validates one-device lease
gateway -> client:
    HELLO_ACK + session details
client:
    enables packet forwarding

8.7 One-device enforcement

Use an expiring active lease rather than permanent hardware binding.

A lease includes:

key_id
device_public_key_hash
session_id
issued_at
expires_at
gateway_id

Rules:

* a key may have one active device public key;
* several paths from that same device are one session;
* reconnections from the same device renew the lease;
* a different device is rejected unless takeover is requested;
* stale leases expire automatically;
* device identifiers are cryptographic, not serial numbers or MAC addresses.

⸻

9. Entitlement model

9.1 Access key

Generate at least 128 bits of cryptographic randomness.

Display the raw key once.

Store:

key_id
key_verifier
created_at
valid_until
status

Do not store the raw access key.

Do not log it.

Use a version prefix:

cv1_<encoded-random-secret>

9.2 Entitlement token

Use a short-lived signed token.

Prefer a compact, deterministic encoding.

Claims:

version
key_id
device_public_key_hash
gateway_id
issued_at
expires_at
nonce

Do not place payment-provider identifiers in the token.

9.3 Suggested validity

Initial values:

entitlement token: 12 hours
session lease: 2 minutes, renewable
gateway path liveness: 15 seconds

These are initial engineering defaults, not immutable product rules.

9.4 Initial API

POST /v1/keys/issue
POST /v1/keys/extend
POST /v1/keys/revoke
POST /v1/entitlements/exchange
POST /v1/sessions/takeover
GET  /v1/status
POST /v1/payments/webhook

Administrative endpoints must not be exposed publicly without authentication and network restrictions.

⸻

10. macOS application specification

10.1 Application structure

The macOS application contains:

1. menu-bar application;
2. Packet Tunnel Provider extension;
3. shared configuration module;
4. Go bridge;
5. Keychain module;
6. diagnostics module;
7. update module.

10.2 Minimum UI

Disconnected state:

Gemina VPN: Disconnected
Access key: Configured
Wi-Fi: GreaterAngliaWiFi
Android USB: Connected
[Connect]
[Settings]
[Diagnostics]

Connected state:

Gemina VPN: Connected
Wi-Fi: Active
Android USB: Active
Gateway: eu-central-1
Session: 00:31:42
Packets protected: 128,442
Duplicates discarded: 116,120
[Disconnect]

Do not expose low-level configuration by default.

10.3 Interface discovery

The client must discover:

* active Wi-Fi interface;
* Android USB tethering interface;
* source IP;
* interface index;
* default gateway where obtainable;
* Internet reachability;
* captive-portal symptoms.

Do not assume fixed names such as en0 or en5.

Do not identify Android tethering solely from an interface name.

Use multiple signals:

* interface type;
* hardware information where available;
* service order;
* assigned address;
* default routes;
* user confirmation when ambiguous.

10.4 Captive portal

Version 1 does not automate login.

Required behaviour:

Train Wi-Fi requires sign-in.
Complete sign-in in your browser, then reconnect.

The client may provide an “Open login page” action later.

10.5 Keychain

Store:

* access key;
* generated device private key;
* gateway trust data;
* entitlement refresh material.

Do not store these in:

* UserDefaults;
* plist files;
* logs;
* crash reports;
* shell history.

10.6 Sleep and wake

Required events:

* stop sending before system sleep where possible;
* preserve non-sensitive session metadata;
* re-enumerate interfaces after wake;
* obtain a new entitlement if required;
* establish new paths;
* restore the tunnel without requiring a full application restart.

10.7 Failure handling

The application must distinguish:

No physical network
Captive Wi-Fi
Android tether disconnected
Gateway unreachable
Access key expired
Access key already active
Transport authentication failure
DNS failure
Tunnel setup failure
Unsupported macOS version

Do not report all failures as “VPN failed”.

⸻

11. Gateway specification

11.1 Components

The gateway host runs:

gemina-gateway
gemina-entitlement-api
Caddy or nginx
nftables
node exporter
application metrics endpoint
SQLite

For private beta, entitlement API and gateway may be separated into different systemd units on the same VPS.

11.2 Linux networking

Required:

net.ipv4.ip_forward=1

The gateway must:

* create the required TUN interface;
* assign tunnel addresses;
* route client packets;
* NAT outbound traffic;
* restrict inbound ports;
* reject spoofed tunnel-source addresses;
* prevent customers reaching gateway management services;
* prevent client-to-client traffic unless explicitly required.

11.3 Gateway process privileges

Do not run the complete process permanently as unrestricted root.

Investigate:

* capability-based privileges;
* privileged setup helper;
* dropping privileges after TUN and socket creation;
* systemd hardening;
* read-only filesystem options;
* restricted address families;
* no-new-privileges;
* isolated temporary directories.

11.4 Metrics

Required aggregate metrics:

active_sessions
active_paths
packets_received_total
packets_forwarded_total
duplicates_discarded_total
invalid_packets_total
stale_packets_total
bytes_client_to_internet
bytes_internet_to_client
path_failures_total
session_auth_failures_total
gateway_cpu
gateway_memory
gateway_network_bytes

Do not label metrics with:

* raw access key;
* public IP indefinitely;
* payment identifier;
* personal name;
* email.

Use bounded-cardinality pseudonymous session identifiers where necessary.

⸻

12. Infrastructure specification

12.1 Hetzner development environment

Create:

deploy/tofu/environments/dev

It provisions:

* one Hetzner VPS;
* one IPv4 address;
* firewall rules;
* SSH administrative access from an allowlist;
* UDP transport port;
* HTTPS entitlement API port;
* monitoring access restricted appropriately;
* cloud-init user;
* persistent volume only if required.

12.2 Initial ports

Do not hard-code final ports until protocol design review.

Example:

22/tcp       administration, IP-restricted
443/tcp      entitlement API
51820/udp    continuity transport path
9100/tcp     metrics, private or IP-restricted

Consider supporting several UDP destination ports later because public Wi-Fi may restrict outbound UDP.

12.3 Deployment properties

Deployment must be:

* reproducible;
* idempotent;
* versioned;
* rollback-capable;
* free of plaintext secrets in Git;
* testable in a disposable development project.

12.4 Secrets

Initial options:

* SOPS with age;
* 1Password CLI;
* environment files provisioned outside Git.

Do not place:

* Hetzner API tokens;
* payment webhook secrets;
* token-signing private keys;
* WireGuard private keys;
* database backups

inside the repository.

⸻

13. Development stages

Stage 0 — project bootstrap and source due diligence

Objective

Create the repository and establish legal, architectural and engineering controls before implementation.

Deliverables

* repository structure;
* AGENTS.md;
* product specification;
* architecture overview;
* upstream manifest;
* licence matrix;
* ADR template;
* security threat-model template;
* Makefile;
* Go workspace;
* baseline Swift project;
* baseline CI;
* research-fetch script.

LLM profile

Use a high-reasoning coding agent with:

* strong repository navigation;
* Git competence;
* software architecture ability;
* licence-awareness;
* ability to run shell commands and tests.

Recommended role:

Primary: Codex high-reasoning model
Reviewer: separate reasoning model/session

Do not use a small autocomplete model for licence classification or initial architecture.

Exit criteria

* clean checkout bootstraps successfully;
* all upstream projects are pinned;
* licence classifications are documented;
* no GPL source has been copied into product directories;
* CI runs on an empty skeleton;
* ADR-0001 records the continuity-first decision.

⸻

Stage 1 — transport laboratory

Objective

Prove dual-interface UDP transmission and server-side deduplication without creating a VPN.

Build

Client CLI:

geminactl probe

Server:

gemina-gateway --probe-mode

The client must:

* discover two interfaces;
* bind one UDP socket to each source;
* send the same numbered datagram over both;
* receive acknowledgements;
* show latency and loss per path.

The server must:

* authenticate a test session;
* record first-arriving packet;
* discard duplicate;
* report statistics.

Tests

* both paths healthy;
* path A disabled;
* path B disabled;
* high latency on one path;
* packet loss;
* duplicated packets;
* reordered packets;
* stale packets;
* process restart;
* sequence rollover simulation.

LLM profile

Primary coding:

* strong Go model;
* capable of concurrency, sockets and tests.

Review:

* networking-focused reasoning model;
* ask it specifically to inspect race conditions, sequence handling and memory bounds.

Exit criteria

* packet duplication works across two distinct interfaces;
* one path can disappear without ending the logical session;
* deduplication has unit, property and fuzz tests;
* no unbounded packet map exists;
* benchmark results are recorded.

⸻

Stage 2 — Linux end-to-end tunnel prototype

Objective

Pass real IP traffic through a TUN device between a Linux client and Hetzner gateway.

Build

* Linux client TUN;
* dual UDP transport;
* gateway TUN;
* NAT;
* DNS;
* test entitlement;
* basic encryption adapter;
* integration test scripts.

Required demonstrations

* curl through tunnel;
* persistent SSH session while one path is disabled;
* UDP media stream while one path is impaired;
* DNS requests through tunnel;
* recovery after path restoration.

LLM profile

Primary:

* senior Go/Linux networking agent.

Review:

* security-oriented model;
* infrastructure model for nftables and systemd.

Exit criteria

* one continuous SSH session survives deliberate path failure;
* no client traffic escapes through an unprotected default route during the test mode;
* NAT and firewall rules are reproducible;
* packet capture confirms encryption;
* gateway rebuild succeeds from code.

⸻

Stage 3 — macOS interface proof of concept

Objective

Prove that the Go transport can send independently through Wi-Fi and Android USB tethering on macOS.

Build

Initially CLI or development application:

* enumerate network interfaces;
* select Wi-Fi path;
* select USB path;
* bind sockets;
* execute probe protocol;
* monitor interface changes.

Important constraint

Do not begin full Packet Tunnel Provider integration until independent path binding is demonstrated on the actual target MacBook.

Real-world test

Use:

* Greater Anglia Wi-Fi;
* Android USB tethering;
* Hetzner development VPS.

Collect:

* RTT;
* jitter;
* packet loss;
* path outages;
* captive-portal events;
* reconnection times;
* percentage of packets won by each path;
* mobile-data overhead.

LLM profile

Primary:

* model strong in Swift, Go and Darwin networking.

Review:

* separate Apple-platform specialist;
* networking specialist.

Exit criteria

* both interfaces are independently used;
* packet captures prove distinct source interfaces;
* failure of either interface does not terminate the probe session;
* findings are recorded in docs/testing/train-test-001.md.

⸻

Stage 4 — Network Extension integration

Objective

Replace the CLI packet source with an Apple Packet Tunnel Provider.

Build

* SwiftUI menu-bar host;
* Packet Tunnel Provider target;
* NEPacketTunnelProvider;
* packet reads from packetFlow;
* Go bridge;
* packet writes back to packetFlow;
* tunnel network settings;
* controlled start and stop;
* basic DNS;
* Keychain storage.

Development sequence

1. loopback Packet Tunnel Provider;
2. single-path remote tunnel;
3. dual-path remote tunnel;
4. path failure;
5. sleep/wake;
6. UI integration.

Do not implement everything simultaneously.

LLM profile

Primary:

* strongest available Apple-platform coding model;
* high-context Codex session with Xcode project access.

Reviewer:

* Apple Network Extension specialist session;
* Go FFI/concurrency reviewer.

Exit criteria

* Internet traffic passes through the gateway;
* both paths are active;
* one-path failure does not break an established SSH session;
* extension stops cleanly;
* routes and DNS are restored after disconnect;
* no Go callback accesses released Swift memory;
* no deadlock occurs after repeated connect/disconnect cycles.

⸻

Stage 5 — entitlement and one-device lease

Objective

Introduce the commercial key model without payment processing.

Build

* access-key generation;
* verifier storage;
* signed entitlement token;
* session lease;
* renewal;
* expiry;
* revocation;
* takeover;
* Keychain integration;
* admin CLI.

LLM profile

Primary:

* Go backend/security-capable agent.

Reviewer:

* cryptographic protocol reviewer;
* privacy reviewer;
* database concurrency reviewer.

Exit criteria

* raw keys are never stored server-side;
* one key cannot maintain two device leases;
* two paths from one device are treated as one lease;
* crashed sessions expire;
* takeover works;
* token signing keys can rotate;
* audit tests cover expiry and clock skew.

⸻

Stage 6 — private alpha

Objective

Produce an installable build for a small number of testers.

Build

* menu-bar status;
* connect/disconnect;
* key entry;
* path indicators;
* clear failure messages;
* diagnostics export;
* signed development build;
* manual update process;
* operational dashboard;
* support runbook.

LLM profile

Primary:

* Swift product-engineering model;
* Go operational model.

Review:

* UX review model;
* supportability review model;
* adversarial tester model.

Exit criteria

* ten or more repeated connect/disconnect cycles succeed;
* sleep/wake recovery succeeds;
* one network can disappear and return;
* diagnostic package contains no secret keys;
* testers can install without source code;
* gateway resource usage is recorded.

⸻

Stage 7 — payment and renewal

Objective

Connect prepaid £5 purchases to key issuance and extension.

Build

* hosted checkout;
* webhook verification;
* idempotent payment processing;
* issue new key;
* extend existing key;
* payment reconciliation;
* refund administration;
* simple key-status page;
* no general account portal.

LLM profile

Primary:

* backend/payment integration model.

Reviewer:

* financial-event and idempotency reviewer;
* security reviewer.

Exit criteria

* duplicate webhooks do not double-extend time;
* failed payments do not activate keys;
* refunds are auditable;
* payment provider data is isolated from gateway logs;
* customer can extend a key without creating an account.

⸻

Stage 8 — security and production hardening

Objective

Prepare for paid external users.

Workstreams

* threat model;
* dependency scan;
* SBOM;
* licence report;
* fuzzing;
* race detection;
* gateway privilege reduction;
* rate limiting;
* abuse controls;
* signing and notarisation;
* secure updates;
* backup and recovery;
* incident-response plan;
* privacy notice;
* acceptable-use policy.

LLM profile

LLMs assist but do not approve production security.

Use:

* security-focused reasoning model;
* static-analysis model;
* operational failure-mode reviewer;
* privacy/compliance model.

Require:

* human security review;
* legal licence review;
* payment terms review.

Exit criteria

* no critical security findings;
* all high findings resolved or formally accepted;
* reproducible release build;
* rollback tested;
* signing keys protected;
* restoration from backup tested;
* dependency licences published.

⸻

Stage 9 — paid launch

Objective

Release to a deliberately limited number of customers.

Launch limits

Initial suggested limits:

* one gateway;
* controlled number of keys;
* no public advertising campaign;
* manual capacity review;
* defined support channel;
* weekly release window;
* documented shutdown procedure.

Exit criteria

* service-level indicators defined;
* support process working;
* capacity headroom maintained;
* payment reconciliation working;
* no unexplained session leaks;
* no repeated transport crashes;
* customer-visible status page available.

⸻

14. LLM orchestration model

14.1 LLM roles

A. Planner/orchestrator

Responsibilities:

* maintain stage plan;
* decompose epics;
* create issues;
* enforce scope;
* review dependencies;
* update ADRs;
* coordinate specialist agents;
* reject premature optimisation.

Use the highest reasoning model available.

B. Go networking implementer

Responsibilities:

* transport;
* protocol;
* gateway;
* deduplication;
* path manager;
* benchmarks;
* fuzz tests.

Needs:

* strong Go;
* sockets;
* concurrency;
* Linux networking.

C. Apple implementer

Responsibilities:

* Swift;
* SwiftUI;
* NetworkExtension;
* Keychain;
* Xcode;
* signing;
* notarisation;
* lifecycle.

Needs:

* Apple platform expertise;
* mixed Swift/Go integration.

D. Infrastructure implementer

Responsibilities:

* Hetzner;
* OpenTofu;
* Debian;
* systemd;
* nftables;
* monitoring;
* deployment.

E. Security reviewer

Responsibilities:

* threat modelling;
* secret handling;
* authentication;
* replay protection;
* parser safety;
* privilege boundaries;
* dependency risks.

The security reviewer must not be the same active context that wrote the code being reviewed.

F. Test engineer

Responsibilities:

* network impairment simulation;
* property tests;
* fuzz tests;
* chaos tests;
* repeatability;
* train-test protocol;
* regression suite.

G. Licence/provenance reviewer

Responsibilities:

* source manifest;
* copied-code detection;
* attribution;
* dependency licences;
* GPL boundaries;
* NOTICE generation.

14.2 Model size allocation

Use a high-reasoning model for:

* architecture;
* protocol design;
* concurrency;
* cryptographic integration;
* Network Extension design;
* security review;
* licence analysis;
* major refactors.

Use a medium coding model for:

* routine tests;
* API handlers;
* migrations;
* Terraform modules;
* documentation synchronisation;
* CI workflows;
* repetitive SwiftUI elements.

Use a local or smaller model only for:

* formatting;
* simple comments;
* renaming;
* straightforward test-data generation;
* mechanical documentation updates.

Do not delegate to a small model:

* packet protocol design;
* replay-window logic;
* cryptography;
* entitlement validation;
* payment webhook logic;
* route manipulation;
* memory ownership across Swift and Go;
* GPL clean-room decisions.

14.3 Agent workflow

For every substantial feature:

1. Planner creates issue and acceptance criteria.
2. Research agent identifies relevant upstream concepts.
3. Implementer writes a design note.
4. Planner approves design note.
5. Implementer creates tests.
6. Implementer writes code.
7. Specialist reviewer reviews independently.
8. Test agent runs failure and regression tests.
9. Licence agent checks provenance.
10. Planner closes issue and updates stage status.

No feature is complete because it compiles.

⸻

15. Codex working rules

15.1 Required behaviour

Codex must:

* inspect the repository before editing;
* read AGENTS.md;
* state which stage and issue it is addressing;
* make small coherent commits;
* run relevant tests;
* update documentation when behaviour changes;
* record architectural decisions;
* avoid speculative abstractions;
* preserve licence headers;
* flag security uncertainty;
* keep generated files out of manual edits;
* use British English in documentation.

15.2 Prohibited behaviour

Codex must not:

* copy GPL implementation into proprietary directories;
* translate GPL files line by line;
* design new cryptographic primitives;
* introduce Kubernetes;
* add user accounts;
* add traffic quotas;
* add multiple pricing tiers;
* add aggregate bonding before continuity is proven;
* add a second client OS;
* hard-code interface names;
* log secrets;
* store raw access keys;
* skip tests because networking is difficult;
* claim success without packet captures or measurable evidence;
* merge generated and handwritten code without clear boundaries.

15.3 Definition of done

Every issue must include:

[ ] acceptance criteria met
[ ] unit tests
[ ] integration tests where applicable
[ ] race detector considered/run
[ ] fuzz test considered
[ ] documentation updated
[ ] threat model updated where applicable
[ ] metrics added
[ ] logs contain no secrets
[ ] licence/provenance checked
[ ] rollback considered
[ ] reviewer approval

⸻

16. Testing strategy

16.1 Unit tests

Mandatory for:

* framing;
* sequence arithmetic;
* deduplication;
* replay window;
* token verification;
* lease decisions;
* packet classification;
* path state machine;
* configuration parsing.

16.2 Property tests

Properties:

* a packet is forwarded at most once;
* duplicates cannot increase delivered packet count;
* packet order does not affect deduplication correctness;
* expired tokens are never accepted;
* one key cannot own two device leases;
* memory use remains bounded;
* reconnecting the same device does not create duplicate sessions.

16.3 Fuzz tests

Targets:

* packet parser;
* entitlement-token parser;
* configuration parser;
* gateway handshake;
* deduplication state;
* length fields;
* malformed IP packets.

16.4 Network simulation

Use Linux network namespaces and tc netem.

Profiles:

stable
high-latency
high-jitter
1-percent-loss
5-percent-loss
burst-loss
path-blackhole
path-flap
asymmetric-return
reordering
duplicate-input
restricted-UDP

16.5 macOS tests

Test:

* Intel if supported later;
* Apple Silicon;
* supported macOS versions;
* Wi-Fi only;
* USB only;
* both;
* Wi-Fi captive;
* USB disconnect;
* sleep/wake;
* rapid network changes;
* VPN connect/disconnect loops;
* application upgrade;
* extension crash.

16.6 Real train testing

Each test run records:

date
route
train service
carriage where known
Mac model
macOS version
Android model
mobile provider
gateway location
Wi-Fi login behaviour
session duration
path outages
call quality observations
SSH continuity
packet duplication overhead
mobile data used
gateway traffic

Do not use anecdotal “felt better” as the only measure.

⸻

17. Performance targets

Initial engineering targets, subject to validation:

Metric	Target
Client connection setup	under 5 seconds after entitlement is available
Path failure detection	under 3 seconds
Session interruption during single-path loss	no TCP session termination
Duplicate-delivery rate to client IP stack	zero
Gateway duplicate suppression	above 99.999% for valid duplicates
Client idle CPU	below 3% on target Mac
Client active call CPU	below 10%
Client memory	below 200 MB
Gateway sessions per small VPS	measure before setting target
Memory per session	bounded and documented
Reconnect after wake	under 10 seconds
Additional latency	under 20 ms excluding VPS geography

The project must measure these rather than designing around unverified assumptions.

⸻

18. Security requirements

18.1 Threats to consider

* stolen access key;
* key guessing;
* replayed entitlement;
* replayed tunnel packet;
* forged path registration;
* session takeover;
* malicious gateway client;
* malformed packet parser attack;
* denial of service;
* excessive session creation;
* payment webhook forgery;
* database theft;
* signing-key theft;
* malicious software update;
* traffic correlation;
* customer abuse of exit IP;
* logging of sensitive metadata.

18.2 Minimum controls

* cryptographically random keys;
* rate-limited entitlement exchange;
* short-lived signed tokens;
* replay window;
* authenticated path association;
* one-device lease;
* gateway process hardening;
* restricted administrative access;
* encrypted backups;
* signed updates;
* Keychain storage;
* minimal logs;
* secret scanning;
* SBOM;
* dependency monitoring.

⸻

19. Legal and provenance requirements

Maintain:

docs/legal/upstream-licences.md
docs/legal/code-provenance.md
docs/legal/dependency-inventory.md
NOTICE

Every reused file must record:

* source repository;
* original path;
* source commit;
* licence;
* modifications;
* date imported.

For inspiration-only projects, maintain clean-room notes that describe:

* behaviour observed;
* public documentation consulted;
* independently written requirements;
* confirmation that implementation code was not copied.

An LLM-generated translation of source code is still potentially derivative. Changing the language does not remove licence obligations.

⸻

20. Initial backlog

Epic 0: repository bootstrap

* create monorepo;
* create AGENTS.md;
* add CI;
* add ADR framework;
* add upstream manifest;
* add licence scan;
* create fetch script.

Epic 1: dual-path probe

* enumerate interfaces;
* bind source addresses;
* send duplicated probes;
* deduplicate server-side;
* collect metrics;
* simulate failure.

Epic 2: tunnel core

* define framing;
* implement sequence space;
* implement replay window;
* implement TUN adapters;
* implement gateway routing;
* implement encryption adapter.

Epic 3: macOS transport

* Darwin interface enumeration;
* USB-path identification;
* network-change monitor;
* Go bridge;
* macOS CLI probe.

Epic 4: Packet Tunnel Provider

* create extension;
* create packet bridge;
* configure routes;
* configure DNS;
* connect and disconnect;
* handle failures.

Epic 5: entitlement

* issue key;
* verify key;
* token signing;
* lease;
* expiry;
* takeover;
* Keychain.

Epic 6: infrastructure

* Hetzner dev environment;
* cloud-init;
* firewall;
* systemd;
* NAT;
* metrics;
* deployment script.

Epic 7: product UI

* menu bar;
* key entry;
* connection state;
* path state;
* errors;
* diagnostics.

Epic 8: commercialisation

* payment checkout;
* webhook;
* renewal;
* notices;
* privacy;
* acceptable-use policy;
* support process.

⸻

21. First Codex session instructions

The first Codex session must perform only Stage 0.

It must not implement the VPN transport yet.

Tasks:

1. Create the repository structure defined in this specification.
2. Create AGENTS.md containing the project rules.
3. Create ADR-0001: continuity-first rather than aggregation-first.
4. Create ADR-0002: Swift client and Go transport/gateway.
5. Create ADR-0003: one opaque key per active device.
6. Create ADR-0004: monorepo.
7. Create the upstream manifest.
8. Create the research-source fetch script.
9. Create the legal/provenance templates.
10. Create skeleton Go modules.
11. Create a skeleton macOS Xcode project or, if Xcode project generation is unreliable, document the exact manual creation steps and add source directories.
12. Add Go CI.
13. Add Swift lint/test CI.
14. Add OpenTofu validation CI.
15. Add licence scanning.
16. Add a Makefile with:

make bootstrap
make test
make test-go
make test-macos
make lint
make licence-check
make fetch-research
make docs-check

17. Create issues or markdown backlog entries for Stage 1.
18. Produce a final report listing:
    * files created;
    * commands run;
    * tests passed;
    * unresolved decisions;
    * risks;
    * recommended next issue.

The first session must stop after Stage 0 and request review through its final report. It must not start Stage 1 opportunistically.

⸻

22. Initial Codex prompt

You are the lead implementation agent for a new commercial macOS networking product.

Read the complete project specification before changing files.

Your current assignment is Stage 0 only: repository bootstrap and source due diligence.

Create a clean monorepo for a continuity VPN that uses public Wi-Fi and Android USB tethering simultaneously. The first product duplicates protected traffic over both paths to a Hetzner gateway. The gateway accepts the first valid packet and drops duplicates. This is a reliability product, not an aggregate-bandwidth product.

The client stack is Swift and NetworkExtension. The shared transport and Linux gateway are written in Go. The first control plane uses Go and SQLite. Infrastructure uses OpenTofu/Terraform and Debian.

Follow these non-negotiable rules:

* Do not copy GPL implementation code into proprietary project directories.
* Engarde and OpenMPTCProuter are inspiration-only.
* WireGuard Apple and wireguard-go are MIT-licensed reusable foundations.
* Do not invent cryptography.
* Do not add Kubernetes.
* Do not add usernames, passwords or conventional customer accounts.
* Do not add quotas or traffic-based billing.
* Do not implement aggregate bonding.
* Do not assume fixed macOS interface names.
* Do not store or log raw access keys.
* Record all external-source provenance.
* Use British English in documentation.

Complete the Stage 0 tasks listed in the specification.

Before editing:

1. inspect the current directory;
2. explain the proposed file changes;
3. identify any environmental limitations;
4. create a task checklist.

During implementation:

* keep commits or logical changes small;
* run tests and validation after each major group;
* do not conceal failed commands;
* do not proceed into Stage 1.

At completion, provide:

* repository tree;
* files created;
* commands executed;
* test results;
* licence findings;
* unresolved decisions;
* recommended first Stage 1 issue.

Important architectural correction

For the continuity MVP, I would not describe the design as simply “run two WireGuard tunnels”. Standard WireGuard associates a peer with an endpoint and roaming behaviour, but the product needs a deliberate layer that controls two simultaneous transport paths, packet identity and first-copy acceptance. The clean design is:

Network Extension packet
        ↓
continuity framing and sequence ID
        ↓
authenticated encrypted transport
        ├── Wi-Fi socket
        └── USB socket
        ↓
gateway deduplication
        ↓
Internet

WireGuard Apple is valuable for its Apple integration, tunnel lifecycle and Go bridge, while Engarde is valuable as behavioural inspiration for redundant-path delivery. The official WireGuard Apple project already contains shared macOS components and links WireGuardKit to a Go bridge.   Engarde confirms that a Go utility can maintain a point-to-point tunnel over multiple connections, but its GPL-2.0 licence means it should remain inspiration-only unless the product is intentionally distributed under compatible terms.

Recommended first engineering decision

Before committing to the complete Network Extension implementation, Stage 1 should prove this exact capability on your Mac:

UDP socket A explicitly leaves through Greater Anglia Wi-Fi
UDP socket B explicitly leaves through Android USB tethering
Both reach the same Hetzner process
One logical packet is delivered once
Either path can disappear without ending the logical session

That experiment determines whether the product is technically viable under macOS routing constraints. It should precede payment integration, polished UI and most control-plane work.