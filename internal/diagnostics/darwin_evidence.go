package diagnostics

import (
	"continuity-vpn/internal/paths"
	"continuity-vpn/internal/platform/darwin"
)

const DarwinEvidenceReportType = "stage-1-darwin-evidence"

// IssueCodeTetherPresentNotUsable flags an uplink whose USB function is present
// on the bus but is not yet a usable path (no host driver, so no NIC/IP/link).
// It explains why an expected candidate is absent without faking one — the
// signal a pre-purchase compatibility check consumes.
const IssueCodeTetherPresentNotUsable = "tether-present-not-usable"

type DarwinEvidenceReport struct {
	Type                 string                     `json:"type"`
	Stage                string                     `json:"stage"`
	Claim                string                     `json:"claim"`
	ClassificationStatus string                     `json:"classification_status"`
	Complete             bool                       `json:"complete"`
	Interfaces           []DiagnosticInterface      `json:"interfaces"`
	Candidates           []DiagnosticCandidate      `json:"candidates"`
	DeviceFunctions      []DiagnosticDeviceFunction `json:"device_functions"`
	Issues               []DiagnosticIssue          `json:"issues"`
}

// DiagnosticDeviceFunction reports a tether-capable USB function seen at the
// device layer, before any network interface exists. Usable is false whenever no
// host driver has claimed it, which keeps a present-but-unusable function from
// being mistaken for a working path.
type DiagnosticDeviceFunction struct {
	Transport         string `json:"transport"`
	HostDriverClaimed bool   `json:"host_driver_claimed"`
	Usable            bool   `json:"usable"`
}

type DiagnosticInterface struct {
	BSDName  string               `json:"bsd_name"`
	Kind     string               `json:"kind"`
	Usable   bool                 `json:"usable"`
	Up       bool                 `json:"up"`
	Running  bool                 `json:"running"`
	Loopback bool                 `json:"loopback"`
	HasIPv4  bool                 `json:"has_ipv4"`
	Evidence []DiagnosticEvidence `json:"evidence"`
}

type DiagnosticEvidence struct {
	Source string `json:"source"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

type DiagnosticCandidate struct {
	Role    string `json:"role"`
	BSDName string `json:"bsd_name"`
	Kind    string `json:"kind"`
}

type DiagnosticIssue struct {
	Code  string `json:"code"`
	Role  string `json:"role"`
	Count int    `json:"count,omitempty"`
}

func BuildDarwinEvidenceReport(snapshots []darwin.InterfaceSnapshot, deviceFunctions []darwin.USBTetherFunction) DarwinEvidenceReport {
	observations := darwin.ObservationsFromSnapshots(snapshots)
	classification := paths.Classify(observations)
	complete := classification.Complete()

	status := "incomplete"
	if complete {
		status = "complete"
	}

	issues := diagnosticIssues(classification.Issues)
	issues = append(issues, deviceFunctionIssues(classification, deviceFunctions)...)

	report := DarwinEvidenceReport{
		Type:                 DarwinEvidenceReportType,
		Stage:                "stage-1-probe",
		Claim:                "diagnostic-only-not-path-success",
		ClassificationStatus: status,
		Complete:             complete,
		Interfaces:           diagnosticInterfaces(snapshots),
		Candidates:           diagnosticCandidates(classification.Candidates),
		DeviceFunctions:      diagnosticDeviceFunctions(deviceFunctions),
		Issues:               issues,
	}

	return report
}

func diagnosticDeviceFunctions(functions []darwin.USBTetherFunction) []DiagnosticDeviceFunction {
	items := make([]DiagnosticDeviceFunction, 0, len(functions))
	for _, fn := range functions {
		items = append(items, DiagnosticDeviceFunction{
			Transport:         fn.Transport,
			HostDriverClaimed: fn.HostDriverClaimed,
			Usable:            fn.HostDriverClaimed,
		})
	}
	return items
}

// deviceFunctionIssues explains, per present-but-unusable tether function, why a
// usable candidate for its role is absent. It never fabricates a candidate; it
// only adds an honest issue when the role has no usable candidate yet.
func deviceFunctionIssues(classification paths.Classification, functions []darwin.USBTetherFunction) []DiagnosticIssue {
	var issues []DiagnosticIssue
	for _, fn := range functions {
		if fn.HostDriverClaimed {
			continue
		}
		role, ok := roleForTransport(fn.Transport)
		if !ok || classification.Candidate(role) != nil {
			continue
		}
		issues = append(issues, DiagnosticIssue{
			Code: IssueCodeTetherPresentNotUsable,
			Role: role.String(),
		})
	}
	return issues
}

func roleForTransport(transport string) (paths.Role, bool) {
	switch transport {
	case darwin.EvidenceValueAndroidRNDIS:
		return paths.RoleAndroidUSBTether, true
	default:
		return 0, false
	}
}

func diagnosticInterfaces(snapshots []darwin.InterfaceSnapshot) []DiagnosticInterface {
	interfaces := make([]DiagnosticInterface, 0, len(snapshots))
	for _, snapshot := range snapshots {
		observation := snapshot.Observation()
		interfaces = append(interfaces, DiagnosticInterface{
			BSDName:  snapshot.BSDName,
			Kind:     snapshot.Kind.String(),
			Usable:   observation.Usable(),
			Up:       snapshot.Up,
			Running:  snapshot.Running,
			Loopback: snapshot.Loopback,
			HasIPv4:  snapshot.HasIPv4,
			Evidence: diagnosticEvidence(snapshot.Evidence),
		})
	}
	return interfaces
}

func diagnosticEvidence(evidence []darwin.Evidence) []DiagnosticEvidence {
	items := make([]DiagnosticEvidence, 0, len(evidence))
	for _, item := range evidence {
		items = append(items, DiagnosticEvidence{
			Source: item.Source.String(),
			Key:    item.Key,
			Value:  item.Value,
		})
	}
	return items
}

func diagnosticCandidates(candidates []paths.Candidate) []DiagnosticCandidate {
	items := make([]DiagnosticCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, DiagnosticCandidate{
			Role:    candidate.Role.String(),
			BSDName: candidate.Observation.ID,
			Kind:    candidate.Observation.Kind.String(),
		})
	}
	return items
}

func diagnosticIssues(issues []paths.Issue) []DiagnosticIssue {
	items := make([]DiagnosticIssue, 0, len(issues))
	for _, issue := range issues {
		items = append(items, DiagnosticIssue{
			Code:  issue.Code.String(),
			Role:  issue.Role.String(),
			Count: issue.Count,
		})
	}
	return items
}
