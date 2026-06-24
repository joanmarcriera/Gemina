import ContinuityVPNCore
import Foundation

// A dependency-free, headless check harness for the ContinuityVPNCore logic.
// Run with `swift run ContinuityVPNCoreCheck`. Exits non-zero on any failure.
// When full Xcode is installed these checks port directly to XCTest/Swift
// Testing (one assertion each).

// Single-threaded check harness; the counter is intentionally unguarded.
nonisolated(unsafe) var failures = 0
func check(_ condition: Bool, _ message: String) {
    if !condition {
        failures += 1
        print("FAIL \(message)")
    }
}

let wifi = PathInfo(id: "wifi", up: true, metered: false)
let phone = PathInfo(id: "phone", up: true, metered: true)
let line2 = PathInfo(id: "line2", up: true, metered: false)

// PathPolicy
check(Set(PathPolicy(mode: .duplicate).sendPaths([wifi, phone], primaryUnstable: false)) == ["wifi", "phone"],
      "duplicate sends over every up path")
check(PathPolicy(mode: .failover, preferredID: "wifi").sendPaths([wifi, phone], primaryUnstable: false) == ["wifi"],
      "failover sends over primary only")
let wifiDown = PathInfo(id: "wifi", up: false, metered: false)
check(PathPolicy(mode: .failover, preferredID: "wifi").sendPaths([wifiDown, phone], primaryUnstable: false) == ["phone"],
      "failover falls back when primary down")
let smart = PathPolicy(mode: .smart, preferredID: "wifi")
check(smart.sendPaths([wifi, phone], primaryUnstable: false) == ["wifi"], "smart: primary only when stable")
check(Set(smart.sendPaths([wifi, phone], primaryUnstable: true)) == ["wifi", "phone"], "smart: duplicate when unstable")
let auto = PathPolicy(mode: .auto, preferredID: "wifi")
check(Set(auto.sendPaths([wifi, line2], primaryUnstable: false)) == ["wifi", "line2"], "auto: duplicate when all unmetered")
check(auto.sendPaths([wifi, phone], primaryUnstable: false) == ["wifi"], "auto: smart when metered (stable)")
check(Set(auto.sendPaths([wifi, phone], primaryUnstable: true)) == ["wifi", "phone"], "auto: smart when metered (unstable)")
check(PathPolicy(mode: .duplicate).sendPaths([wifi], primaryUnstable: false) == ["wifi"], "single path used alone")

// ProtectionStatus
func two(_ a: Bool, _ b: Bool) -> [PathInfo] {
    [PathInfo(id: "a", up: a, metered: false), PathInfo(id: "b", up: b, metered: true)]
}
check(protectionStatus(paths: two(true, true), paused: false, connecting: false) == .protected, "status protected")
check(protectionStatus(paths: two(true, false), paused: false, connecting: false) == .degraded, "status degraded")
check(protectionStatus(paths: two(false, true), paused: false, connecting: false) == .degraded, "status degraded is symmetric")
let three1 = [PathInfo(id: "a", up: true, metered: false), PathInfo(id: "b", up: false, metered: false), PathInfo(id: "c", up: false, metered: true)]
check(protectionStatus(paths: three1, paused: false, connecting: false) == .degraded, "status degraded with 3 configured, 1 up")
check(protectionStatus(paths: two(false, false), paused: false, connecting: false) == .down, "status down")
check(protectionStatus(paths: [wifi], paused: false, connecting: false) == .atRisk, "status at risk")
check(protectionStatus(paths: two(true, true), paused: true, connecting: false) == .paused, "status paused wins")
check(protectionStatus(paths: two(true, true), paused: false, connecting: true) == .connecting, "status connecting wins")
check(protectionStatus(paths: [], paused: false, connecting: false) == .off, "status off when none configured")

// Consent
check(defaultShareEnabled(tier: .free) == false, "free tier is opt-in")
check(defaultShareEnabled(tier: .hosted) == true, "paid tier is opt-out")

// Impact
let s = computeImpact(segments: [
    .init(duration: 10, upCount: 2),
    .init(duration: 5, upCount: 1),
    .init(duration: 20, upCount: 2),
    .init(duration: 3, upCount: 1),
])
check(s.sessionDuration == 38, "impact session duration")
check(s.protectedDuration == 30, "impact protected duration")
check(s.outageAbsorbed == 8, "impact outage absorbed")
check(s.failoversSurvived == 2, "impact failovers survived")
check(s.longestDropSurvived == 5, "impact longest drop survived")
check(abs(s.protectedFraction - 30.0 / 38.0) < 0.0001, "impact protected fraction")

let merged = computeImpact(segments: [
    .init(duration: 4, upCount: 2),
    .init(duration: 3, upCount: 1),
    .init(duration: 2, upCount: 1),
])
check(merged.failoversSurvived == 1, "impact merges contiguous drops (one transition)")
check(merged.longestDropSurvived == 5, "impact merges contiguous drops (longest run)")

// A session that starts on one path was never protected: no absorbed outage.
let neverProtected = computeImpact(segments: [.init(duration: 5, upCount: 1), .init(duration: 5, upCount: 1)])
check(neverProtected.outageAbsorbed == 0, "never-protected session absorbs no outage")
check(neverProtected.failoversSurvived == 0, "never-protected session survives no failover")

// Recovery from a total outage (0 up) is not a survived drop.
let totalOutage = computeImpact(segments: [
    .init(duration: 10, upCount: 2),
    .init(duration: 4, upCount: 1), // survived drop from redundancy
    .init(duration: 2, upCount: 0), // total outage
    .init(duration: 3, upCount: 1), // recovery, not a survived drop
])
check(totalOutage.outageAbsorbed == 4, "only the drop from redundancy counts as absorbed")
check(totalOutage.failoversSurvived == 1, "recovery from total outage is not a failover survived")

// Smart with the preferred path down still picks the surviving path.
let smartPreferredDown = PathPolicy(mode: .smart, preferredID: "wifi")
let wifiDown2 = PathInfo(id: "wifi", up: false, metered: false)
check(smartPreferredDown.sendPaths([wifiDown2, line2], primaryUnstable: false) == ["line2"],
      "smart with preferred down uses the surviving path")

if failures == 0 {
    print("PASS ContinuityVPNCore: all checks passed")
    exit(0)
}
print("FAIL ContinuityVPNCore: \(failures) check(s) failed")
exit(1)
