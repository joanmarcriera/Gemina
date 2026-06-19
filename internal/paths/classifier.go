package paths

type LinkKind uint8

const (
	LinkKindUnknown LinkKind = iota
	LinkKindWiFi
	LinkKindAndroidUSBTether
)

func (kind LinkKind) String() string {
	switch kind {
	case LinkKindWiFi:
		return "wi-fi"
	case LinkKindAndroidUSBTether:
		return "android-usb-tether"
	default:
		return "unknown"
	}
}

type Role uint8

const (
	RoleWiFi Role = iota + 1
	RoleAndroidUSBTether
)

func (role Role) String() string {
	switch role {
	case RoleWiFi:
		return "wi-fi"
	case RoleAndroidUSBTether:
		return "android-usb-tether"
	default:
		return "unknown"
	}
}

type Observation struct {
	ID          string
	DisplayName string
	Kind        LinkKind
	Up          bool
	Running     bool
	Loopback    bool
	HasIPv4     bool
}

func (observation Observation) Usable() bool {
	return observation.ID != "" &&
		observation.Up &&
		observation.Running &&
		!observation.Loopback &&
		observation.HasIPv4
}

type Candidate struct {
	Role        Role
	Observation Observation
}

type IssueCode uint8

const (
	IssueMissingCandidate IssueCode = iota + 1
	IssueAmbiguousCandidate
)

func (code IssueCode) String() string {
	switch code {
	case IssueMissingCandidate:
		return "missing-candidate"
	case IssueAmbiguousCandidate:
		return "ambiguous-candidate"
	default:
		return "unknown"
	}
}

type Issue struct {
	Code  IssueCode
	Role  Role
	Count int
}

type Classification struct {
	Candidates []Candidate
	Issues     []Issue
}

func (classification Classification) Complete() bool {
	return len(classification.Issues) == 0 &&
		classification.Candidate(RoleWiFi) != nil &&
		classification.Candidate(RoleAndroidUSBTether) != nil
}

func (classification Classification) Candidate(role Role) *Candidate {
	for i := range classification.Candidates {
		if classification.Candidates[i].Role == role {
			return &classification.Candidates[i]
		}
	}
	return nil
}

func Classify(observations []Observation) Classification {
	var result Classification
	classifyRole(&result, RoleWiFi, LinkKindWiFi, observations)
	classifyRole(&result, RoleAndroidUSBTether, LinkKindAndroidUSBTether, observations)
	return result
}

func classifyRole(result *Classification, role Role, kind LinkKind, observations []Observation) {
	matches := matchingUsable(kind, observations)
	switch len(matches) {
	case 0:
		result.Issues = append(result.Issues, Issue{
			Code: IssueMissingCandidate,
			Role: role,
		})
	case 1:
		result.Candidates = append(result.Candidates, Candidate{
			Role:        role,
			Observation: matches[0],
		})
	default:
		result.Issues = append(result.Issues, Issue{
			Code:  IssueAmbiguousCandidate,
			Role:  role,
			Count: len(matches),
		})
	}
}

func matchingUsable(kind LinkKind, observations []Observation) []Observation {
	var matches []Observation
	for _, observation := range observations {
		if observation.Kind == kind && observation.Usable() {
			matches = append(matches, observation)
		}
	}
	return matches
}
