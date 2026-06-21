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

1. rsyncs first-party source to `oracle:/opt/continuity-vpn` (never `.research-src`);
2. builds the image natively (arm64) from `deploy/docker/gateway.Dockerfile`;
3. installs/refreshes `deploy/systemd/continuity-gateway.service` and restarts it;
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
sudo journalctl -u continuity-gateway.service -n 20 --no-pager
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

* Image `continuity-gateway:latest` (distroless, non-root, read-only rootfs).
* systemd unit `/etc/systemd/system/continuity-gateway.service`.
* Source tree under `/opt/continuity-vpn`.
* Host firewall: one UDP port.

To remove: `sudo systemctl disable --now continuity-gateway.service`, remove the
unit file, `sudo docker rmi continuity-gateway:latest`, remove
`/opt/continuity-vpn`, and `sudo firewall-cmd --remove-port=<port>/udp
--permanent --reload`.
