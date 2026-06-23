package main

import (
	"encoding/json"
	"fmt"
	"os"

	"continuity-vpn/internal/bootstrap"
	"continuity-vpn/internal/diagnostics"
	"continuity-vpn/internal/platform/darwin"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println(bootstrap.ComponentStage("continuityctl"))
		return
	}

	switch os.Args[1] {
	case "darwin-evidence":
		if err := runDarwinEvidence(); err != nil {
			fmt.Fprintf(os.Stderr, "continuityctl darwin-evidence: %v\n", err)
			os.Exit(1)
		}
	case "preflight":
		if err := runPreflight(os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "continuityctl preflight: %v\n", err)
			os.Exit(1)
		}
	case "probe":
		if err := runProbe(os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "continuityctl probe: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "usage: %s [darwin-evidence|preflight|probe]\n", os.Args[0])
		os.Exit(2)
	}
}

func runDarwinEvidence() error {
	snapshots, err := darwin.LiveEvidenceInterfaceSnapshots()
	if err != nil {
		return err
	}

	deviceFunctions, err := darwin.LiveUSBTetherFunctions()
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(diagnostics.BuildDarwinEvidenceReport(snapshots, deviceFunctions))
}
