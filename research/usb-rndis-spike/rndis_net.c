// rndis_net.c — shared socket/env helpers (see rndis_net.h).

#include "rndis_net.h"

#include <arpa/inet.h>
#include <net/if.h>
#include <netinet/in.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>

int gateway_from_env(uint8_t ip[4], int *port) {
    const char *ip_s = getenv("GEMINA_GATEWAY_IP");
    const char *port_s = getenv("GEMINA_GATEWAY_PORT");
    if (!ip_s)
        return -1;
    int a, b, c, d;
    if (sscanf(ip_s, "%d.%d.%d.%d", &a, &b, &c, &d) != 4)
        return -1;
    if (a < 0 || a > 255 || b < 0 || b > 255 || c < 0 || c > 255 || d < 0 ||
        d > 255)
        return -1;
    ip[0] = (uint8_t)a;
    ip[1] = (uint8_t)b;
    ip[2] = (uint8_t)c;
    ip[3] = (uint8_t)d;
    int p = port_s ? atoi(port_s) : 51820;
    if (p <= 0 || p > 65535)
        p = 51820;
    *port = p;
    return 0;
}

void random_session(uint8_t session[16]) {
    for (int i = 0; i < 16; i++)
        session[i] = (uint8_t)rand();
    session[0] |= 1; // ensure non-zero
}

int wifi_socket_open(const char *iface, const uint8_t ip[4], int port) {
    int fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0)
        return -1;

    // Bind egress to the named interface regardless of the default route.
    unsigned idx = if_nametoindex(iface);
    if (idx == 0) {
        close(fd);
        return -1;
    }
    if (setsockopt(fd, IPPROTO_IP, IP_BOUND_IF, &idx, sizeof(idx)) != 0) {
        close(fd);
        return -1;
    }

    struct sockaddr_in dst;
    memset(&dst, 0, sizeof(dst));
    dst.sin_family = AF_INET;
    dst.sin_port = htons((uint16_t)port);
    memcpy(&dst.sin_addr, ip, 4);
    if (connect(fd, (struct sockaddr *)&dst, sizeof(dst)) != 0) {
        close(fd);
        return -1;
    }
    return fd;
}

int wifi_socket_send(int fd, const uint8_t *payload, int len) {
    ssize_t n = send(fd, payload, (size_t)len, 0);
    return (n == len) ? 0 : -1;
}

void wifi_socket_close(int fd) {
    if (fd >= 0)
        close(fd);
}
