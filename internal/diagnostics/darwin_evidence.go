package diagnostics

import (
	"continuity-vpn/internal/paths"
	"continuity-vpn/internal/platform/darwin"
)

const DarwinEvidenceReportType = "stage-1-darwin-evidence"

type DarwinEvidenceReport struct {
	Type                 string                `json:"type"`
	Stage                string                `json:"stage"`
	Claim                string                `json:"claim"`
	ClassificationStatus string                `json:"classification_status"`
	Complete             bool                  `json:"complete"`
	Interfaces           []DiagnosticInterface `json:"interfaces"`
	Candidates           []DiagnosticCandidate `json:"candidates"`
	Issues               []DiagnosticIssue     `json:"issues"`
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

func BuildDarwinEvidenceReport(snapshots []darwin.InterfaceSnapshot) DarwinEvidenceReport {
	observations := darwin.ObservationsFromSnapshots(snapshots)
	classification := paths.Classify(observations)
	complete := classification.Complete()

	status := "incomplete"
	if complete {
		status = "complete"
	}

	report := DarwinEvidenceReport{
		Type:                 DarwinEvidenceReportType,
		Stage:                "stage-1-probe",
		Claim:                "diagnostic-only-not-path-success",
		ClassificationStatus: status,
		Complete:             complete,
		Interfaces:           diagnosticInterfaces(snapshots),
		Candidates:           diagnosticCandidates(classification.Candidates),
		Issues:               diagnosticIssues(classification.Issues),
	}

	return report
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
