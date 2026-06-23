// rndis_net.h — small shared helpers for the spike drivers: gateway config from
// the environment, random session ids, and an OS UDP socket bound to a specific
// interface (the Wi-Fi path A in the dual-path proof).
//
// Provenance: original; uses only standard BSD sockets + Darwin IP_BOUND_IF.

#ifndef RNDIS_NET_H
#define RNDIS_NET_H

#include <stdint.h>

// Continuity probe PathTag values (must match internal/protocol).
#define PATH_WIFI 1
#define PATH_ANDROID_USB_TETHER 2

// Read CONTINUITY_GATEWAY_IP (dotted quad) and CONTINUITY_GATEWAY_PORT (default
// 51820) from the environment. Returns 0 on success, -1 if the IP is missing or
// malformed. Keeping the address in the environment means no server IP is ever
// compiled into the binary (repo redaction invariant).
int gateway_from_env(uint8_t ip[4], int *port);

// Fill a non-zero 16-byte session id. Assumes rand() has already been seeded.
void random_session(uint8_t session[16]);

// Open a connected UDP socket whose egress is bound to `iface` (e.g. "en0") via
// Darwin IP_BOUND_IF, so datagrams leave that interface regardless of the
// default route — the OS-side equivalent of the RNDIS userspace path. Connected
// to ip:port. Returns the fd, or -1 on error.
int wifi_socket_open(const char *iface, const uint8_t ip[4], int port);

// Send one datagram on a socket from wifi_socket_open. Returns 0 on success.
int wifi_socket_send(int fd, const uint8_t *payload, int len);

void wifi_socket_close(int fd);

#endif // RNDIS_NET_H
