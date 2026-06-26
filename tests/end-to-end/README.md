# End-to-End Tests

## Stage-2 server-exit demo (`rig_linux.go`)

`rig_linux.go` is a Linux **test harness** (not a shipped product) that stands in
for the macOS client so the gateway's internet exit path can be demonstrated on
real hardware. It opens a TUN device, performs the real authenticated handshake
against a running data gateway, and pumps inner IP packets over **one or more
uplinks at once** — duplicating each outbound packet across every path and
deduplicating on the way back, exactly as the macOS client will. Cutting one
uplink mid-flow must not break an established flow (the Stage-2 exit criterion:
a continuous SSH session survives losing a path).

It is build-tagged `linux && e2e` so it never enters normal builds or tests.

### Compile-check (from any host)

```sh
GOOS=linux GOARCH=arm64 go build -tags e2e ./tests/end-to-end/...
```

### Run the demo (on a Linux box, as root for the TUN)

1. Start the gateway in data + exit mode and note the **base64 Ed25519 identity**
   it logs (clients pin it) and the address pool. Enable kernel forwarding +
   `MASQUERADE` on the gateway host first.
2. The first admitted client deterministically receives the second usable address
   in the pool (the network address and the gateway host address are reserved).
   Pass that as the rig's tunnel IP for now — in-band delivery of the assigned
   address is a tracked follow-up.
3. Run the rig, pointing it at the gateway, with one entry per uplink interface:

   ```sh
   sudo GEMINA_RIG_GATEWAY=<gateway-host>:51820 \
        GEMINA_RIG_IDENTITY=<base64 ed25519 pub from the gateway log> \
        GEMINA_RIG_TOKEN=<entitlement token> \
        GEMINA_RIG_TUNNEL_IP=<assigned tunnel address> \
        GEMINA_RIG_PATHS=<iface-a>,<iface-b> \
        ./rig
   ```

4. Route demo traffic (e.g. an SSH session) through the tunnel, then take one of
   the named interfaces down. The session should survive; the other uplink keeps
   carrying the duplicated copies.

`GEMINA_RIG_PATHS` is optional — omit it to use the default route as a single
path. Each path is a UDP socket bound to its interface (`SO_BINDTODEVICE`).
