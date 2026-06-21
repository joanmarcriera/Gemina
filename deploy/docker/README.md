# Containers

`gateway.Dockerfile` builds the Stage-1 probe gateway: a multi-stage build that
compiles a static binary and ships it on a distroless, non-root base. It is built
natively on the arm64 target host by `scripts/deploy-dev-gateway.sh` (no registry,
no cross-compilation). See `docs/dev/gateway-deploy.md`.
