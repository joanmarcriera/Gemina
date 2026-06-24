import Foundation

// PathPolicy decides, for each outbound packet, which of the available paths to
// send it over. The transport sends the same framed bytes over the chosen paths
// and the gateway deduplicates, so every mode works over the existing transport
// (ADR-0005/0006). This is the send-side knob behind "prioritise one network".

/// How aggressively to use the second path.
public enum PolicyMode: Sendable, Equatable {
    /// Every packet over every up path. Seamless, zero-loss failover; double data.
    case duplicate
    /// Primary only; the other is hot standby (used when the primary is down).
    case failover
    /// Primary only normally; duplicate over all up paths while the primary is
    /// unstable, then settle back.
    case smart
    /// Duplicate when all up paths are unmetered; behave as Smart when a metered
    /// path is present. The default — maximum protection when data is free,
    /// conserve it when it is not.
    case auto
}

/// A configured path and its current state.
public struct PathInfo: Sendable, Equatable {
    public let id: String
    public let up: Bool
    public let metered: Bool

    public init(id: String, up: Bool, metered: Bool) {
        self.id = id
        self.up = up
        self.metered = metered
    }
}

public struct PathPolicy: Sendable, Equatable {
    public var mode: PolicyMode
    /// The user's preferred primary path id, or nil to auto-pick.
    public var preferredID: String?

    public init(mode: PolicyMode, preferredID: String? = nil) {
        self.mode = mode
        self.preferredID = preferredID
    }

    /// The ids to send the current packet over. `primaryUnstable` reflects recent
    /// loss/latency on the primary (measured by the benchmark pings / provider).
    public func sendPaths(_ paths: [PathInfo], primaryUnstable: Bool) -> [String] {
        let up = paths.filter(\.up)
        guard up.count > 1 else { return up.map(\.id) } // 0 or 1 path: no choice

        let primary = choosePrimary(up)
        switch effectiveMode(up) {
        case .duplicate:
            return up.map(\.id)
        case .failover:
            return [primary.id]
        case .smart:
            return primaryUnstable ? up.map(\.id) : [primary.id]
        case .auto:
            // effectiveMode never returns .auto.
            return up.map(\.id)
        }
    }

    /// Auto collapses to Duplicate (all unmetered) or Smart (a metered path
    /// present); other modes pass through.
    private func effectiveMode(_ up: [PathInfo]) -> PolicyMode {
        guard mode == .auto else { return mode }
        return up.allSatisfy { !$0.metered } ? .duplicate : .smart
    }

    /// The preferred path if it is up; otherwise the first up unmetered path;
    /// otherwise the first up path.
    private func choosePrimary(_ up: [PathInfo]) -> PathInfo {
        if let id = preferredID, let p = up.first(where: { $0.id == id }) {
            return p
        }
        return up.first(where: { !$0.metered }) ?? up[0]
    }
}
