# systemd

`gemina-gateway.service` runs the Stage-1 probe gateway container under
systemd (restart on boot and on crash). It is installed and refreshed by
`scripts/deploy-dev-gateway.sh`. See `docs/dev/gateway-deploy.md`.

`gemina-gateway-data.service` runs the Stage-2 data + exit gateway (host
networking, TUN, NAT). It starts as uid 0 with only NET_ADMIN/SETUID/SETGID and
the gateway drops to uid 65532 after opening its socket and TUN (issue #5); its
`ExecStartPre` re-provisions the host via `scripts/setup-exit-host.sh`. Deploy
with `GATEWAY_UNIT=gemina-gateway-data.service scripts/deploy-dev-gateway.sh`.
The two units conflict (same UDP port).
