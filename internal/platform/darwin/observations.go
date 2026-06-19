package darwin

import "continuity-vpn/internal/paths"

type EvidenceSource uint8

const (
	EvidenceSourceUnknown EvidenceSource = iota
	EvidenceSourceSystemConfiguration
	EvidenceSourceNetworkFramework
	EvidenceSourceIORegistry
)

func (source EvidenceSource) String() string {
	switch source {
	case EvidenceSourceSystemConfiguration:
		return "system-configuration"
	case EvidenceSourceNetworkFramework:
		return "network-framework"
	case EvidenceSourceIORegistry:
		return "io-registry"
	default:
		return "unknown"
	}
}

type Evidence struct {
	Source EvidenceSource
	Key    string
	Value  string
}

type InterfaceSnapshot struct {
	BSDName     string
	DisplayName string
	Kind        paths.LinkKind
	Up          bool
	Running     bool
	Loopback    bool
	HasIPv4     bool
	Evidence    []Evidence
}

func (snapshot InterfaceSnapshot) EvidenceBySource(source EvidenceSource) []Evidence {
	var matches []Evidence
	for _, evidence := range snapshot.Evidence {
		if evidence.Source == source {
			matches = append(matches, evidence)
		}
	}
	return matches
}

func ObservationsFromSnapshots(snapshots []InterfaceSnapshot) []paths.Observation {
	observations := make([]paths.Observation, 0, len(snapshots))
	for _, snapshot := range snapshots {
		observations = append(observations, snapshot.Observation())
	}
	return observations
}

func (snapshot InterfaceSnapshot) Observation() paths.Observation {
	return paths.Observation{
		ID:          snapshot.BSDName,
		DisplayName: snapshot.DisplayName,
		Kind:        snapshot.Kind,
		Up:          snapshot.Up,
		Running:     snapshot.Running,
		Loopback:    snapshot.Loopback,
		HasIPv4:     snapshot.HasIPv4,
	}
}
