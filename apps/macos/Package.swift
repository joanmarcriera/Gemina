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
        .target(
            name: "ContinuityVPNApp",
            dependencies: ["ContinuityVPNShared"],
            path: "App"
        ),
        .target(
            name: "ContinuityVPNPacketTunnelExtension",
            dependencies: ["ContinuityVPNShared"],
            path: "PacketTunnelExtension"
        )
    ]
)
