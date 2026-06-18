# Upstream Licences

This document records Stage 0 licence due diligence. It is not legal advice.

No upstream implementation source has been copied into product directories. Root licence files were inspected from pinned clones under `.research-src/` on 2026-06-17. This table is a Stage 0 engineering record, not legal advice.

| Project | Repository | Root Licence Observed | Intended Use | Status |
| --- | --- | --- | --- | --- |
| WireGuard Apple | https://github.com/WireGuard/wireguard-apple | MIT | Direct reuse or study after review | Pinned; legal review before import |
| wireguard-go | https://github.com/WireGuard/wireguard-go | MIT | Direct reuse or dependency after review | Pinned; legal review before import |
| Engarde | https://github.com/porech/engarde | GPL-2.0 text | Behavioural study only | Pinned; inspiration-only |
| Glorytun | https://github.com/angt/glorytun | BSD-2-Clause | Reference, possible selective reuse after review | Pinned; legal review before import |
| MLVPN | https://github.com/zehome/MLVPN | BSD-2-Clause | Operational and algorithmic reference | Pinned; legal review before import |
| mptunnel | https://github.com/greensea/mptunnel | BSD-2-Clause | Reference only | Pinned; legal review before import |
| OpenMPTCProuter | https://github.com/Ysurac/openmptcprouter | GPL-3.0 text | Behavioural and operational inspiration only | Pinned; inspiration-only |
| mqVPN | https://github.com/mp0rta/mqvpn | Apache-2.0 | Future architecture reference | Pinned; legal review before study |
| hcloud-go | https://github.com/hetznercloud/hcloud-go | MIT | Future API reference | Pinned; legal review before import |
| terraform-provider-hcloud | https://github.com/hetznercloud/terraform-provider-hcloud | MPL-2.0 | Provider use and infrastructure reference | Pinned; legal review before import |
| hcloud CLI | https://github.com/hetznercloud/cli | MIT | Operational reference | Pinned; legal review before import |

Before copying, vendoring or modifying any upstream file, update `docs/legal/code-provenance.md` and `NOTICE`.
