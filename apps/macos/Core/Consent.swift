import Foundation

// Consent defaults for sharing anonymous compatibility data. The free / self-host
// tier is opt-in (off by default; onboarding invites helping the catalogue); the
// paid hosted tier is opt-out (on by default, disclosed clearly at purchase, one
// toggle to stop). Whatever the default, the only data shared is the redacted
// ShareReport tokens, and the user can preview the exact payload first.

public enum Tier: Sendable, Equatable {
    case free   // self-host or unpaid
    case hosted // paid hosted gateway
}

/// The default state of the "Share anonymous compatibility data" toggle for a
/// tier: false (opt-in) for free/self-host, true (opt-out) for the paid tier.
public func defaultShareEnabled(tier: Tier) -> Bool {
    switch tier {
    case .free: return false
    case .hosted: return true
    }
}
