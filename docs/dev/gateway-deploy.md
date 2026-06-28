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

## Stage-2: the data + exit gateway (WS-F prerequisite) — DRAFT

> **DRAFT (2026-06-28), not yet validated on hardware.** Validate during WS-F
> (`docs/superpowers/plans/2026-06-26-phase3-wifi-tunnel.md`); see also the
> step-by-step handoff in `docs/dev/handoff-2026-06-28.md`. The probe gateway
> above only deduplicates probes — it is **not** a tunnel. The macOS app needs a
> gateway running the real handshake + encrypted data plane **and** the internet
> exit path. `cmd/gateway` supports this (`GEMINA_GATEWAY_MODE=data` +
> `GEMINA_GATEWAY_EXIT=on`); the deploy below is the new, unvalidated part.

Why it differs from probe mode: exit mode opens a Linux TUN device
(`CAP_NET_ADMIN` + `/dev/net/tun`), routes leased client IPs, and relies on a
host NAT MASQUERADE rule (the Go exit engine deliberately never touches the
firewall — `internal/exit/nat_linux.go`). So the container is **not** read-only,
uses host networking, and mounts a writable volume for the persisted Ed25519
identity (a fresh key would break every client's pin).

```bash
# 1. Ship the image (reuses the same build as probe mode).
scripts/deploy-dev-gateway.sh                      # builds gemina-gateway:latest on oracle

# 2. Prepare the host: ip_forward, persistent TUN gemina0, pool addr, NAT.
ssh oracle 'cd /opt/gemina && sudo WAN_IF=<egress-iface> scripts/setup-exit-host.sh'

# 3. Install + start the data+exit unit (instead of the probe unit).
ssh oracle '
  sudo install -m0644 /opt/gemina/deploy/systemd/gemina-gateway-data.service \
       /etc/systemd/system/gemina-gateway-data.service
  sudo systemctl daemon-reload
  sudo systemctl enable --now gemina-gateway-data.service'

# 4. Read the pinned identity the app needs (base64 Ed25519 public key).
ssh oracle 'sudo journalctl -u gemina-gateway-data.service | grep public_key | tail -1'
```

Then open ingress **UDP 51820** in the cloud firewall (VCN) as for probe mode.
In open/self-host mode (the default) admission ignores the token, so the app only
needs the gateway host + that base64 public key. Do **not** run both the probe and
the data unit on the same port at once.

To remove: `sudo systemctl disable --now gemina-gateway-data.service`, remove the
unit, then undo the host setup (`sudo ip link del gemina0`, drop the MASQUERADE
rule, `sudo rm /etc/sysctl.d/99-gemina-exit.conf`).
