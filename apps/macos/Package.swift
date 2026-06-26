// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "ContinuityVPNMacOS",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .library(name: "ContinuityVPNShared", targets: ["ContinuityVPNShared"])
    ],
    targets: [
        .target(name: "ContinuityVPNShared", path: "Shared"),
        // Pure app logic (no AppKit/NetworkExtension): path policy, protection
        // status, consent defaults, impact maths. Unit-tested headless.
        .target(name: "ContinuityVPNCore", path: "Core"),
        // Headless self-checking harness for the core logic. Runs with the plain
        // toolchain (no Xcode): `swift run ContinuityVPNCoreCheck`. Becomes an
        // XCTest/Swift-Testing target once full Xcode is installed; the test
        // bodies port across unchanged.
        .executableTarget(
            name: "ContinuityVPNCoreCheck",
            dependencies: ["ContinuityVPNCore"],
            path: "CoreCheck"
        ),
        // C module exposing the Go transport core's ABI
        // (bridge/include/continuitycore.h). The symbols are linked from the Go
        // c-archive in the Xcode project; this target only carries the
        // declarations so the Swift side compiles against them.
        .target(name: "CContinuityCore", path: "CContinuityCore"),
        .target(
            name: "ContinuityVPNPacketTunnelExtension",
            dependencies: ["ContinuityVPNShared", "CContinuityCore", "ContinuityVPNCore"],
            path: "PacketTunnelExtension"
        )
    ]
)
