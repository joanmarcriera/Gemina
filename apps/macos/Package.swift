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
        // C module exposing the Go transport core's ABI
        // (bridge/include/continuitycore.h). The symbols are linked from the Go
        // c-archive in the Xcode project; this target only carries the
        // declarations so the Swift side compiles against them.
        .target(name: "CContinuityCore", path: "CContinuityCore"),
        .target(
            name: "ContinuityVPNApp",
            dependencies: ["ContinuityVPNShared"],
            path: "App"
        ),
        .target(
            name: "ContinuityVPNPacketTunnelExtension",
            dependencies: ["ContinuityVPNShared", "CContinuityCore"],
            path: "PacketTunnelExtension"
        )
    ]
)
