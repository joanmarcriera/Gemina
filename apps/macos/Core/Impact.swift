import Foundation

// Impact maths: turn a session's path-state timeline into the honest, measured
// "how useful was this" figures shown in the popover and the Impact tab. Computed
// from segments of (duration, number-of-paths-up); stored locally, never shared.

/// One stretch of a session during which the number of up paths was constant.
public struct ImpactSegment: Sendable, Equatable {
    public let duration: TimeInterval
    public let upCount: Int

    public init(duration: TimeInterval, upCount: Int) {
        self.duration = duration
        self.upCount = upCount
    }
}

public struct ImpactStats: Sendable, Equatable {
    /// Total session time.
    public var sessionDuration: TimeInterval = 0
    /// Time with two or more paths up (a drop would have been absorbed).
    public var protectedDuration: TimeInterval = 0
    /// Time spent surviving on a single path after another dropped — the outage
    /// the product absorbed.
    public var outageAbsorbed: TimeInterval = 0
    /// Number of times a path dropped (from ≥2 up to 1 up) without ending the
    /// session.
    public var failoversSurvived: Int = 0
    /// The longest single stretch survived on one path.
    public var longestDropSurvived: TimeInterval = 0
    /// Share of the session that was fully protected (0…1).
    public var protectedFraction: Double = 0
}

// Note: the spec's "data sent per path" figure is a separate running byte counter
// maintained by the provider/relay (it is not derivable from the up/down
// timeline these segments carry); the Impact tab aggregates both. It is
// deliberately not part of computeImpact.

/// Computes the impact statistics from a session's ordered segments.
///
/// "Outage absorbed" and "longest drop survived" count only single-path time that
/// is a *survived drop from redundancy* — i.e. time at one path up that began
/// with a fall from two-or-more up. Single-path time in a session that was never
/// protected (started on one path, or recovered from a total outage) is not an
/// absorbed outage, because there was no second path doing the saving.
public func computeImpact(segments: [ImpactSegment]) -> ImpactStats {
    var stats = ImpactStats()
    var previousUp = 0
    var inSurvivedDrop = false // at one path, having fallen from >= 2
    var currentDropRun: TimeInterval = 0

    for segment in segments {
        stats.sessionDuration += segment.duration

        switch segment.upCount {
        case let upCount where upCount >= 2:
            stats.protectedDuration += segment.duration
            inSurvivedDrop = false
            currentDropRun = 0
        case 1:
            if previousUp >= 2 {
                stats.failoversSurvived += 1
                inSurvivedDrop = true
                currentDropRun = segment.duration
            } else if inSurvivedDrop {
                currentDropRun += segment.duration // contiguous survived-drop stretch
            }
            if inSurvivedDrop {
                stats.outageAbsorbed += segment.duration
                stats.longestDropSurvived = max(stats.longestDropSurvived, currentDropRun)
            }
        default: // 0 up: a total outage, not something the second path absorbed
            inSurvivedDrop = false
            currentDropRun = 0
        }
        previousUp = segment.upCount
    }

    if stats.sessionDuration > 0 {
        stats.protectedFraction = stats.protectedDuration / stats.sessionDuration
    }
    return stats
}
