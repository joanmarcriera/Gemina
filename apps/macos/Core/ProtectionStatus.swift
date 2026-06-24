import Foundation

// The single source of truth for the menu-bar status. The "degraded" state — a
// configured path is down but another is carrying you — is deliberately distinct
// from "at risk" (only one connection, no redundancy), so the UI can frame a
// survived drop as success rather than an alarm.

public enum ProtectionStatus: Sendable, Equatable {
    /// Two or more paths up: a drop would be absorbed.
    case protected
    /// A configured second path is down; still covered by the surviving one.
    case degraded
    /// Only one connection is configured; not protected against a drop.
    case atRisk
    /// Configured for redundancy but nothing is up right now.
    case down
    /// The user paused protection.
    case paused
    /// Establishing the connection.
    case connecting
    /// Not running / no connections configured.
    case off
}

/// Derives the protection status from the configured paths and the run state.
public func protectionStatus(paths: [PathInfo], paused: Bool, connecting: Bool) -> ProtectionStatus {
    if paused { return .paused }
    if connecting { return .connecting }
    if paths.isEmpty { return .off }

    let upCount = paths.filter(\.up).count
    switch (paths.count, upCount) {
    case (_, let n) where n >= 2:
        return .protected
    case (let configured, 1) where configured >= 2:
        return .degraded
    case (1, 1):
        return .atRisk
    default:
        return .down
    }
}

public extension ProtectionStatus {
    /// A short, plain-language label for the popover headline.
    var headline: String {
        switch self {
        case .protected: return "Protected"
        case .degraded: return "Protected — one link down"
        case .atRisk: return "Not protected — add a second connection"
        case .down: return "No connection"
        case .paused: return "Paused"
        case .connecting: return "Connecting…"
        case .off: return "Off"
        }
    }
}
