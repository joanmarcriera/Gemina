// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "GeminaVPNMacOS",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .library(name: "GeminaVPNShared", targets: ["GeminaVPNShared"])
    ],
    targets: [
        .target(name: "GeminaVPNShared", path: "Shared"),
        // Pure app logic (no AppKit/NetworkExtension): path policy, protection
        // status, consent defaults, impact maths. Unit-tested headless.
        .target(name: "GeminaVPNCore", path: "Core"),
        // Headless self-checking harness for the core logic. Runs with the plain
        // toolchain (no Xcode): `swift run GeminaVPNCoreCheck`. Becomes an
        // XCTest/Swift-Testing target once full Xcode is installed; the test
        // bodies port across unchanged.
        .executableTarget(
            name: "GeminaVPNCoreCheck",
            dependencies: ["GeminaVPNCore"],
            path: "CoreCheck"
        ),
        // C module exposing the Go transport core's ABI
        // (bridge/include/geminacore.h). The symbols are linked from the Go
        // c-archive in the Xcode project; this target only carries the
        // declarations so the Swift side compiles against them.
        .target(name: "CGeminaCore", path: "CGeminaCore"),
        .target(
            name: "GeminaVPNPacketTunnelExtension",
            dependencies: ["GeminaVPNShared", "CGeminaCore", "GeminaVPNCore"],
            path: "PacketTunnelExtension"
        )
    ]
)
