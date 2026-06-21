# Reproducible build for the userspace RNDIS viability spike.
# Requires: clang and libusb (`brew install libusb`).

BREW_PREFIX := $(shell brew --prefix 2>/dev/null || echo /opt/homebrew)
CFLAGS := -Wall -Wextra -I$(BREW_PREFIX)/include
LDFLAGS := -L$(BREW_PREFIX)/lib -lusb-1.0

rndis_probe: rndis_probe.c
	clang $(CFLAGS) $< -o $@ $(LDFLAGS)

.PHONY: run clean
run: rndis_probe
	./rndis_probe

clean:
	rm -f rndis_probe
