# systemd

`continuity-gateway.service` runs the Stage-1 probe gateway container under
systemd (restart on boot and on crash). It is installed and refreshed by
`scripts/deploy-dev-gateway.sh`. See `docs/dev/gateway-deploy.md`.
