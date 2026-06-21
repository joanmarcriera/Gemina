package main

import (
	"errors"
	"testing"

	"continuity-vpn/internal/protocol"
)

func TestParseProbeConfigDefaults(t *testing.T) {
	cfg, err := parseProbeConfig([]string{"-interface", "en0", "-to", "gw.example:51820"})
	if err != nil {
		t.Fatalf("parseProbeConfig: %v", err)
	}
	if cfg.iface != "en0" || cfg.to != "gw.example:51820" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.path != protocol.PathWiFi {
		t.Fatalf("default path = %v, want wi-fi", cfg.path)
	}
	if cfg.count != 1 || cfg.duplicate {
		t.Fatalf("defaults count=%d duplicate=%v", cfg.count, cfg.duplicate)
	}
}

func TestParseProbeConfigRequiresInterfaceAndTo(t *testing.T) {
	for _, args := range [][]string{
		{"-to", "gw:51820"},
		{"-interface", "en0"},
	} {
		if _, err := parseProbeConfig(args); err == nil {
			t.Fatalf("parseProbeConfig(%v): expected error, got nil", args)
		}
	}
}

func TestParseProbeConfigPathTag(t *testing.T) {
	tests := map[string]protocol.PathTag{
		"wifi":               protocol.PathWiFi,
		"wi-fi":              protocol.PathWiFi,
		"usb":                protocol.PathAndroidUSBTether,
		"android-usb-tether": protocol.PathAndroidUSBTether,
	}
	for in, want := range tests {
		cfg, err := parseProbeConfig([]string{"-interface", "en0", "-to", "gw:1", "-path", in})
		if err != nil {
			t.Fatalf("parseProbeConfig(path=%q): %v", in, err)
		}
		if cfg.path != want {
			t.Fatalf("path %q -> %v, want %v", in, cfg.path, want)
		}
	}
}

func TestParseProbeConfigRejectsUnknownPath(t *testing.T) {
	_, err := parseProbeConfig([]string{"-interface", "en0", "-to", "gw:1", "-path", "ethernet"})
	if !errors.Is(err, errUnknownPath) {
		t.Fatalf("unknown path error = %v, want errUnknownPath", err)
	}
}
