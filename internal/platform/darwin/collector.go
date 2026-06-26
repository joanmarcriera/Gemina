package darwin

import (
	"errors"
	"fmt"
	"net"

	"github.com/joanmarcriera/gemina/internal/paths"
)

// ErrNilInterfaceSource is returned when collection is requested without an
// explicit source.
var ErrNilInterfaceSource = errors.New("darwin interface source is nil")

// InterfaceSource supplies live or fixture interface records for snapshot
// collection.
type InterfaceSource interface {
	Interfaces() ([]InterfaceRecord, error)
}

// InterfaceRecord is the collector input before it is normalised into the
// project-wide InterfaceSnapshot boundary.
type InterfaceRecord struct {
	BSDName      string
	DisplayName  string
	Flags        net.Flags
	AddressCIDRs []string
	Kind         paths.LinkKind
	Evidence     []Evidence
}

// NetInterfaceSource reads BSD interface state through Go's standard net
// package. It does not classify Wi-Fi or Android USB tethering.
type NetInterfaceSource struct{}

// LiveInterfaceSnapshots collects the conservative live interface state
// available through NetInterfaceSource.
func LiveInterfaceSnapshots() ([]InterfaceSnapshot, error) {
	return CollectInterfaceSnapshots(NetInterfaceSource{})
}

// Interfaces returns live interface records from the host operating system.
func (NetInterfaceSource) Interfaces() ([]InterfaceRecord, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}

	records := make([]InterfaceRecord, 0, len(interfaces))
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("list addresses for interface %s: %w", iface.Name, err)
		}

		addressCIDRs := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			addressCIDRs = append(addressCIDRs, addr.String())
		}

		records = append(records, InterfaceRecord{
			BSDName:      iface.Name,
			Flags:        iface.Flags,
			AddressCIDRs: addressCIDRs,
			Kind:         paths.LinkKindUnknown,
		})
	}

	return records, nil
}

// CollectInterfaceSnapshots normalises interface records into snapshots without
// inferring link kinds from BSD names or display names.
func CollectInterfaceSnapshots(source InterfaceSource) ([]InterfaceSnapshot, error) {
	if source == nil {
		return nil, ErrNilInterfaceSource
	}

	records, err := source.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("collect darwin interface records: %w", err)
	}

	snapshots := make([]InterfaceSnapshot, 0, len(records))
	for _, record := range records {
		snapshots = append(snapshots, snapshotFromInterfaceRecord(record))
	}

	return snapshots, nil
}

func snapshotFromInterfaceRecord(record InterfaceRecord) InterfaceSnapshot {
	hasIPv4 := hasIPv4CIDR(record.AddressCIDRs)
	kind := record.Kind
	if kind == paths.LinkKindUnknown {
		kind = LinkKindFromEvidence(record.Evidence)
	}

	return InterfaceSnapshot{
		BSDName:     record.BSDName,
		DisplayName: record.DisplayName,
		Kind:        kind,
		Up:          record.Flags&net.FlagUp != 0,
		Running:     record.Flags&net.FlagRunning != 0,
		Loopback:    record.Flags&net.FlagLoopback != 0,
		HasIPv4:     hasIPv4,
		Evidence:    append(networkStateEvidence(record.Flags, hasIPv4), record.Evidence...),
	}
}

func networkStateEvidence(flags net.Flags, hasIPv4 bool) []Evidence {
	return []Evidence{
		{Source: EvidenceSourceBSDNetworkState, Key: "flag-up", Value: boolEvidence(flags&net.FlagUp != 0)},
		{Source: EvidenceSourceBSDNetworkState, Key: "flag-running", Value: boolEvidence(flags&net.FlagRunning != 0)},
		{Source: EvidenceSourceBSDNetworkState, Key: "flag-loopback", Value: boolEvidence(flags&net.FlagLoopback != 0)},
		{Source: EvidenceSourceBSDNetworkState, Key: "ipv4", Value: boolEvidence(hasIPv4)},
	}
}

func hasIPv4CIDR(addressCIDRs []string) bool {
	for _, cidr := range addressCIDRs {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			ip = net.ParseIP(cidr)
		}
		if ip != nil && ip.To4() != nil {
			return true
		}
	}
	return false
}

func boolEvidence(value bool) string {
	if value {
		return "present"
	}
	return "absent"
}
