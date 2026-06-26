# Stage-1 probe gateway image. Built natively on the arm64 target (no registry,
# no cross-compilation). Multi-stage: compile a static binary, ship it on a
# minimal distroless base running as a non-root user.
#
# Provenance: builds only first-party packages from this module (no external
# Go dependencies, no .research-src). Base images are permissively licensed.

# syntax=docker/dockerfile:1

FROM golang:1.26 AS build
WORKDIR /src
# Single module with no external requires; GOWORK=off keeps the build to go.mod.
ENV GOWORK=off CGO_ENABLED=0
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
RUN go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot
LABEL org.opengemina.component="stage-1-probe-gateway"
COPY --from=build /out/gateway /usr/local/bin/gateway
# Default listen port; overridable via GEMINA_GATEWAY_ADDR.
EXPOSE 51820/udp
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/gateway"]
