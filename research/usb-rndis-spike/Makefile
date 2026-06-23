# Reproducible build for the userspace RNDIS viability spike.
# Requires: clang and libusb (`brew install libusb`).

BREW_PREFIX := $(shell brew --prefix 2>/dev/null || echo /opt/homebrew)
CFLAGS := -Wall -Wextra -I$(BREW_PREFIX)/include
LDFLAGS := -L$(BREW_PREFIX)/lib -lusb-1.0

# Shared object sources for the libusb-backed drivers.
CORE := rndis_lib.c rndis_usb.c rndis_uplink.c rndis_net.c
CORE_HDRS := rndis_lib.h rndis_usb.h rndis_uplink.h rndis_net.h

rndis_probe: rndis_probe.c
	clang $(CFLAGS) $< -o $@ $(LDFLAGS)

rndis_dataplane: rndis_dataplane.c rndis_lib.c rndis_usb.c rndis_lib.h rndis_usb.h
	clang $(CFLAGS) rndis_dataplane.c rndis_lib.c rndis_usb.c -o $@ $(LDFLAGS)

rndis_egress: rndis_egress.c $(CORE) $(CORE_HDRS)
	clang $(CFLAGS) rndis_egress.c $(CORE) -o $@ $(LDFLAGS)

rndis_dualpath: rndis_dualpath.c $(CORE) $(CORE_HDRS)
	clang $(CFLAGS) rndis_dualpath.c $(CORE) -o $@ $(LDFLAGS)

# Pure unit tests: no libusb, no hardware.
rndis_lib_test: rndis_lib_test.c rndis_lib.c rndis_lib.h
	clang $(CFLAGS) rndis_lib_test.c rndis_lib.c -o $@

.PHONY: run run-dataplane run-egress run-dualpath test clean
run: rndis_probe
	./rndis_probe

run-dataplane: rndis_dataplane
	./rndis_dataplane

run-egress: rndis_egress
	./rndis_egress

run-dualpath: rndis_dualpath
	./rndis_dualpath

test: rndis_lib_test
	./rndis_lib_test

clean:
	rm -f rndis_probe rndis_dataplane rndis_egress rndis_dualpath rndis_lib_test
