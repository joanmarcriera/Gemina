package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"continuity-vpn/internal/diagnostics"
	"continuity-vpn/internal/platform/darwin"
)

// runPreflight is the onboarding/pre-purchase compatibility check: it gathers
// live device evidence and the macOS version, then prints a plain verdict
// telling the user whether their Mac + Android combination will work and, if
// not, the one thing to change. Defaults to a one-line summary; -json prints the
// full redacted report for the app/website to consume.
func runPreflight(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("preflight", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "print the full JSON report instead of a one-line summary")
	if err := fs.Parse(args); err != nil {
		return err
	}

	snapshots, err := darwin.LiveEvidenceInterfaceSnapshots()
	if err != nil {
		return err
	}
	deviceFunctions, err := darwin.LiveUSBTetherFunctions()
	if err != nil {
		return err
	}
	version, err := darwin.LiveMacOSProductVersion()
	if err != nil {
		return err
	}

	report := diagnostics.BuildCompatibilityReport(snapshots, deviceFunctions, version)
	return writePreflight(out, report, *asJSON)
}

// writePreflight renders the report. Separated from the live collection so the
// output shape is unit-testable.
func writePreflight(out io.Writer, report diagnostics.CompatibilityReport, asJSON bool) error {
	if asJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	_, err := fmt.Fprintln(out, report.Summary())
	return err
}
