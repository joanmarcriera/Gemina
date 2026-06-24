# Self-hosting the gateway

The gateway is the server half of Continuity VPN. It is a single container that
listens on one UDP port, deduplicates the copies of each packet that arrive over
a client's two uplinks, and delivers the first valid copy. It holds no accounts
and no database, so running your own is deliberately simple.

> **Pre-release note.** Today's gateway is the Stage-1 probe gateway: it
> deduplicates probe packets and logs each decision as redacted JSON. It does not
> yet carry encrypted VPN traffic. See [`PROJECT_STATE.md`](../PROJECT_STATE.md)
> for the current status.

## What you need

* A host reachable from the internet (any small VPS or home server will do; the
  architecture is host-agnostic).
* Docker, or any OCI-compatible container runtime.
* One inbound **UDP** port open to the host — `51820` by default.

## Run the container

Build the image from
[`deploy/docker/gateway.Dockerfile`](../deploy/docker/gateway.Dockerfile), or use
your own published image, then run it:

```bash
docker run --rm \
  --read-only \
  -p 51820:51820/udp \
  -e CONTINUITY_GATEWAY_ADDR=:51820 \
  ghcr.io/example/continuity-gateway:latest
```

`ghcr.io/example/continuity-gateway:latest` is a placeholder — substitute your
own image reference.

### Configuration

The gateway is configured entirely through environment variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `CONTINUITY_GATEWAY_ADDR` | `:51820` | UDP listen address and port. |
| `CONTINUITY_GATEWAY_DEDUP_CAPACITY` | `8192` | Size of the in-memory deduplication window. |
| `CONTINUITY_GATEWAY_READ_BUFFER` | `4 MiB` | Socket read buffer, for tolerating bursts. |
| `CONTINUITY_GATEWAY_LOG_LEVEL` | `info` | Set to `debug` for per-packet decision logs. |

The container runs non-root on a read-only root filesystem; keep it that way.

### Open the port

The container publishes the UDP port to the host, but you must also allow it
through any host firewall **and** any cloud-provider network filter (security
group, network security list, and the like). The client's source port is an
ephemeral high port, so the ingress rule must allow **any source port** to the
gateway's destination port — restricting the source port to the gateway port is a
common mistake that silently drops every packet.

A reference deployment as a systemd-managed container, including the firewall
steps, is documented in [`dev/gateway-deploy.md`](dev/gateway-deploy.md).

## Point the client at your gateway

The client takes the gateway address as configuration. Give it your host's name
and port, for example:

```
gateway.example.com:51820
```

**The gateway address is always configurable and is never hard-coded.** The same
client works against a gateway you host yourself and against the hosted option —
only the address differs. Prefer a hostname over a raw IP address so you can move
the gateway without reconfiguring clients.

## Verifying it works

With `CONTINUITY_GATEWAY_LOG_LEVEL=debug`, the gateway logs one line per packet
decision as JSON. A working dual-path session shows `"decision":"first-copy"` for
the first copy of each packet identity and `"decision":"duplicate"` for the later
copy arriving over the second path. No client source address ever appears in the
logs — the handler is never given it.

## Important: do not run the gateway at home

The gateway must sit on a **well-connected, independent network — not on your home
or office line.** Here is why. The client duplicates your protected traffic and
sends *both* copies to the gateway. If the gateway is on your home connection,
both copies converge on that single home WAN, so:

* you lose the whole point — there is no longer a second independent path at the
  gateway end; if the home line blips, both copies are lost;
* your home **upload** bandwidth (usually small and asymmetric) becomes the
  bottleneck for all your duplicated traffic;
* you add a hop through your home before traffic reaches the internet.

Run the gateway on a small cloud VPS instead (any provider; a tiny instance is
plenty for a UDP relay). That gives it an independent, well-provisioned uplink —
which is exactly what the dual-path design needs. **Our hosted gateway is simply
the zero-effort version of this**: a maintained, monitored VPS endpoint in a
datacentre, so you do not have to provision, secure and update one yourself.

## Self-host vs hosted: the trade-off

Both options use the same open-source client and gateway. The only difference is
who runs the server (and, per the note above, self-hosting means a cheap VPS,
not your home box).

**Self-host** when you want:

* full control over where your traffic exits and who can see the gateway;
* no recurring cost beyond your own host;
* to audit or modify the gateway you depend on.

You take on running and updating the container, keeping the port open, and the
availability of your own host.

**Use the hosted gateway** (planned, optional, paid — pricing TBD) when you want:

* to skip running and maintaining a server entirely;
* a maintained, monitored endpoint to point the client at.

The hosted tier is the project's commercial model: it funds the open-source core
without putting any feature behind a paywall in the client itself. If it ever
stops meeting your needs, self-hosting remains a first-class, fully supported
path — that is the point of keeping the gateway open source.
