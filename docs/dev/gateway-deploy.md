# Deploying the Stage-1 probe gateway

The gateway is the server half of the Stage-1 dual-path proof: a UDP listener
that deduplicates probe copies arriving over multiple client paths and logs each
decision as redacted JSON. It runs as a container under systemd on the remote
arm64 host (`ssh oracle`).

## One command to (re)deploy

```bash
scripts/deploy-dev-gateway.sh            # host "oracle", UDP port 51820
GATEWAY_HOST=oracle GATEWAY_PORT=51820 scripts/deploy-dev-gateway.sh
```

This is the release path — re-run it to ship a new build. It:

1. rsyncs first-party source to `oracle:/opt/gemina` (never `.research-src`);
2. builds the image natively (arm64) from `deploy/docker/gateway.Dockerfile`;
3. installs/refreshes `deploy/systemd/gemina-gateway.service` and restarts it;
4. opens the port in the host firewall (firewalld).

The service is enabled, so it restarts on boot and on crash (`Restart=always`).

## The one manual step: cloud firewall (VCN)

The host has two firewalls. The script opens the **host** one (firewalld). The
**Oracle Cloud VCN security list / network security group** is a separate,
cloud-level ingress filter that can only be changed from the Oracle Cloud
console (or the OCI CLI with API credentials, which are not on the host). Until
it allows ingress UDP on the gateway port, probes sent from outside reach the
cloud edge and are dropped before the host.

To open it, in the Oracle Cloud console for region **uk-london-1**:

* Networking ▸ Virtual Cloud Networks ▸ (the VCN for this instance's subnet) ▸
  Security Lists ▸ the subnet's list ▸ **Add Ingress Rule**:
  * Stateless: No
  * Source type: CIDR
  * Source CIDR: anywhere (the all-zeros `/0` wildcard), or your test client's
    address for a tighter rule
  * IP protocol: UDP
  * **Source port range: All** (leave blank). This is the easy mistake: the
    client's source port is an ephemeral high port, so restricting the source
    port to the gateway port silently drops every probe.
  * Destination port range: the gateway port (default 51820)

The instance's subnet CIDR and VNIC OCID are available from the instance
metadata service on the host if a tighter rule is wanted.

## Verifying

On-box (bypasses the VCN — proves the deployed container itself works):

```bash
ssh oracle   # then send a few probes to localhost:51820 and read the logs
sudo journalctl -u gemina-gateway.service -n 20 --no-pager
```

End-to-end from your machine (requires the VCN rule above):

```bash
scripts/probe-gateway.sh
```

Expect log lines with `"decision":"first-copy"` and `"decision":"duplicate"`,
where a duplicate of the same packet identity arriving over a second path tag
reports the original `first_path`. No source address ever appears in the logs —
the handler never receives it.

## What is redacted

The gateway logs only coarse fields: decision, path tag, first path, copy count.
It deliberately discards the datagram source address, so a client identifier
cannot leak into logs or journald. Keep it that way if you add fields.

## Footprint on the host

* Image `gemina-gateway:latest` (distroless, non-root, read-only rootfs).
* systemd unit `/etc/systemd/system/gemina-gateway.service`.
* Source tree under `/opt/gemina`.
* Host firewall: one UDP port.

To remove: `sudo systemctl disable --now gemina-gateway.service`, remove the
unit file, `sudo docker rmi gemina-gateway:latest`, remove
`/opt/gemina`, and `sudo firewall-cmd --remove-port=<port>/udp
--permanent --reload`.

## Stage-2: the data + exit gateway (WS-F prerequisite)

> **Validated on the Oracle host 2026-09-25** (WS-F; Vikunja
> https://familia.riera.co.uk/tasks/2609, GitHub issue #5): hardened unit up,
> tunnel ping 5/5, egress = the gateway's public IP, 20 MB HTTPS through the
> tunnel. The probe gateway above only deduplicates probes; it is **not** a
> tunnel. The macOS app needs a gateway running the real handshake + encrypted
> data plane **and** the internet exit path (`GEMINA_GATEWAY_MODE=data` +
> `GEMINA_GATEWAY_EXIT=on`). Handoff context: `docs/dev/handoff-2026-06-28.md`.

Why it differs from probe mode: exit mode opens a Linux TUN device, routes
leased client IPs, and relies on a host NAT MASQUERADE rule (the Go exit engine
deliberately never touches the firewall — `internal/exit/nat_linux.go`). So it
uses host networking, `/dev/net/tun`, and a writable volume for the persisted
Ed25519 identity (a fresh key would break every client's pin).

**Least privilege (issue #5).** Docker never makes an added capability effective
for a non-root container user, so `--cap-add NET_ADMIN` on the image's
`:nonroot` uid (65532) cannot open the TUN (the WS-F crash-loop: `set mtu on
gemina0: operation not permitted`). The unit therefore starts the container as
uid 0 with `--cap-drop ALL --cap-add NET_ADMIN --cap-add SETUID --cap-add SETGID
--security-opt no-new-privileges --read-only`; `cmd/gateway` opens the UDP socket
and the TUN fd, then drops to `GEMINA_GATEWAY_RUN_AS` (default `65532:65532`),
which clears every capability, and verifies it cannot regain root **before** it
loads the identity or reads a datagram. It refuses to start if the drop cannot be
verified. Check on the host: `grep -E 'Uid|Cap(Prm|Eff)' /proc/$(docker inspect
-f '{{.State.Pid}}' gemina-gateway-data)/status` → uid 65532, caps all zero.

**Host provisioning is part of the unit.** `scripts/setup-exit-host.sh`
(installed as `/usr/local/sbin/gemina-setup-exit-host`) runs as `ExecStartPre` on
every start: `ip_forward`, persistent TUN `gemina0` (owner 65532, `10.99.0.1/16`,
MTU 1280), the MASQUERADE rule, and ACCEPT rules for `gemina0` ↔ WAN in Docker's
`DOCKER-USER` chain (Docker forces the iptables FORWARD policy to DROP, which
otherwise blackholes pool traffic even with NAT in place). None of that survives
a reboot on its own; re-running on start makes a reboot or docker restart safe.

```bash
# 1. Ship the image, install the setup script + data unit, disable the probe
#    unit (same port), and (re)start the data unit.
GATEWAY_UNIT=gemina-gateway-data.service scripts/deploy-dev-gateway.sh

# 2. Read the pinned identity the app needs (base64 Ed25519 public key).
ssh oracle 'sudo journalctl -u gemina-gateway-data.service | grep public_key | tail -1'
```

`WAN_IF` defaults to the default-route interface; override it with a drop-in
(`Environment=WAN_IF=...`) if the host has several. Open ingress **UDP 51820** in
the cloud firewall (VCN) as for probe mode. In open/self-host mode (the default)
admission ignores the token, so the app only needs the gateway host + that base64
public key. The two units `Conflicts=` each other; never run both on one port.

To remove: `sudo systemctl disable --now gemina-gateway-data.service`, remove the
unit and `/usr/local/sbin/gemina-setup-exit-host`, then undo the host setup
(`sudo ip link del gemina0`, `sudo nft delete table inet gemina`, delete the two
`gemina0` rules from `DOCKER-USER`, `sudo rm /etc/sysctl.d/99-gemina-exit.conf`).
