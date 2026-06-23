# Reproducible build for the userspace RNDIS viability spike.
# Requires: clang and libusb (`brew install libusb`).

BREW_PREFIX := $(shell brew --prefix 2>/dev/null || echo /opt/homebrew)
CFLAGS := -Wall -Wextra -I$(BREW_PREFIX)/include
LDFLAGS := -L$(BREW_PREFIX)/lib -lusb-1.0

rndis_probe: rndis_probe.c
	clang $(CFLAGS) $< -o $@ $(LDFLAGS)

rndis_dataplane: rndis_dataplane.c rndis_lib.c rndis_usb.c rndis_lib.h rndis_usb.h
	clang $(CFLAGS) rndis_dataplane.c rndis_lib.c rndis_usb.c -o $@ $(LDFLAGS)

rndis_egress: rndis_egress.c rndis_lib.c rndis_usb.c rndis_lib.h rndis_usb.h
	clang $(CFLAGS) rndis_egress.c rndis_lib.c rndis_usb.c -o $@ $(LDFLAGS)

# Pure unit tests: no libusb, no hardware.
rndis_lib_test: rndis_lib_test.c rndis_lib.c rndis_lib.h
	clang $(CFLAGS) rndis_lib_test.c rndis_lib.c -o $@

.PHONY: run run-dataplane run-egress test clean
run: rndis_probe
	./rndis_probe

run-dataplane: rndis_dataplane
	./rndis_dataplane

run-egress: rndis_egress
	./rndis_egress

test: rndis_lib_test
	./rndis_lib_test

clean:
	rm -f rndis_probe rndis_dataplane rndis_egress rndis_lib_test
