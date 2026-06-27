#if canImport(NetworkExtension)
import Foundation
import Network
import NetworkExtension
import GeminaVPNCore
import os.lock

// GeminaTunnelBootstrap is the concrete production packet tunnel. It reads the
// gateway endpoint + pinned Ed25519 identity + entitlement token from the VPN
// profile's providerConfiguration, performs the on-wire handshake (ADR-0007)
// over a Wi-Fi-bound UDP socket — learning its gateway-leased tunnel IP in-band
// from the ServerHello — installs real NEPacketTunnelNetworkSettings from that
// address, and runs the proven DualPathRelay.
//
// Single-path Wi-Fi for now; the cellular/RNDIS uplink layers on later as a
// second PathSender (out of scope here, see userspace-rndis-dataplane).
//
// Set as the extension's NSExtensionPrincipalClass in project.yml.

enum BootstrapError: Error, LocalizedError {
    case missingProviderConfiguration
    case missingField(String)
    case invalidGatewayKey

    var errorDescription: String? {
        switch self {
        case .missingProviderConfiguration:
            return "The VPN profile carried no provider configuration."
        case .missingField(let name):
            return "The VPN configuration is missing the \"\(name)\" field."
        case .invalidGatewayKey:
            return "The gateway public key is not a valid 32-byte Ed25519 key."
        }
    }
}

final class GeminaTunnelBootstrap: GeminaTunnelProvider, @unchecked Sendable {

    /// Keys in the VPN profile's providerConfiguration dictionary.
    private enum ConfigKey {
        static let port = "port"
        static let gatewayPublicKey = "gatewayPublicKey" // base64 32-byte Ed25519 identity
        static let token = "token"
        static let dnsServers = "dnsServers"
    }

    /// Inbound dedup-window width (recent packet numbers remembered for replay).
    private static let dedupCapacity: Int32 = 1024
    /// MTU with headroom for the 30-byte CVD1 header + UDP/IP encapsulation.
    private static let tunnelMTU: NSNumber = 1380
    /// How long to wait for the handshake's ServerHello before giving up.
    private static let handshakeTimeout: TimeInterval = 5

    // Set during makeRelay(), read while building network settings (both run on
    // the startTunnel thread); guarded so the @unchecked Sendable contract holds.
    private let assignedAddress = OSAllocatedUnfairLock<String?>(initialState: nil)
    private let dnsServers = OSAllocatedUnfairLock<[String]>(initialState: [])
    private let wifiSender = OSAllocatedUnfairLock<WiFiPathSender?>(initialState: nil)

    override func makeRelay() throws -> DualPathRelay {
        guard let proto = protocolConfiguration as? NETunnelProviderProtocol,
              let host = proto.serverAddress, !host.isEmpty else {
            throw TunnelError.missingGatewayAddress
        }
        guard let config = proto.providerConfiguration else {
            throw BootstrapError.missingProviderConfiguration
        }
        let port = try Self.requireUInt16(config, ConfigKey.port)
        let token = try Self.requireString(config, ConfigKey.token)
        let gatewayKey = try Self.requireGatewayKey(config, ConfigKey.gatewayPublicKey)
        if let dns = config[ConfigKey.dnsServers] as? [String] {
            dnsServers.withLock { $0 = dns }
        }

        // Pin the gateway socket to the Wi-Fi interface so it leaves over Wi-Fi and
        // never loops back through the tunnel we are about to install.
        let wifi = try WiFiPathSender(
            gatewayHost: host,
            gatewayPort: port,
            boundInterface: Self.primaryWiFiInterfaceName()
        )
        wifiSender.withLock { $0 = wifi }

        let handshake = try CoreTransport.connect(
            gatewayPublicKey: gatewayKey,
            token: token,
            dedupCapacity: Self.dedupCapacity,
            sendClientHello: { try wifi.send($0) },
            receiveServerHello: { try wifi.receiveOneDatagram(timeout: Self.handshakeTimeout) }
        )

        // Stash the dotted-quad form of the leased tunnel IP (nil if unassigned).
        let ip = handshake.assignedIPv4
        if ip.0 != 0 || ip.1 != 0 || ip.2 != 0 || ip.3 != 0 {
            assignedAddress.withLock { $0 = "\(ip.0).\(ip.1).\(ip.2).\(ip.3)" }
        }

        // Feed datagrams arriving on Wi-Fi through dedup/decrypt and back to the tunnel.
        wifi.receiveLoop { [weak self] datagram in
            self?.handleInbound(datagram, path: "wifi")
        }

        return DualPathRelay(
            core: handshake.core,
            paths: [wifi],
            policy: PathPolicy(mode: .auto)
        )
    }

    override func makeTunnelSettings(tunnelRemoteAddress: String) -> NEPacketTunnelNetworkSettings {
        let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: tunnelRemoteAddress)
        guard let address = assignedAddress.withLock({ $0 }) else {
            return settings // no lease: stay route-less rather than claim a bogus address
        }
        // /32 host address for the tunnel interface.
        let ipv4 = NEIPv4Settings(addresses: [address], subnetMasks: ["255.255.255.255"])
        // Full-tunnel for the single-path demo. The gateway socket bypasses the
        // tunnel because WiFiPathSender pins it to the Wi-Fi interface, so there is
        // no routing loop. Scope to included routes per footprint.md later.
        ipv4.includedRoutes = [NEIPv4Route.default()]
        settings.ipv4Settings = ipv4
        settings.mtu = Self.tunnelMTU

        let dns = dnsServers.withLock { $0 }
        if !dns.isEmpty {
            settings.dnsSettings = NEDNSSettings(servers: dns)
        }
        return settings
    }

    override func stopTunnel(with reason: NEProviderStopReason, completionHandler: @escaping () -> Void) {
        let sender = wifiSender.withLock { current -> WiFiPathSender? in
            let existing = current
            current = nil
            return existing
        }
        sender?.close()
        super.stopTunnel(with: reason, completionHandler: completionHandler)
    }

    // MARK: - Config helpers

    private static func requireString(_ config: [String: Any], _ key: String) throws -> String {
        guard let value = config[key] as? String, !value.isEmpty else {
            throw BootstrapError.missingField(key)
        }
        return value
    }

    private static func requireUInt16(_ config: [String: Any], _ key: String) throws -> UInt16 {
        if let number = config[key] as? NSNumber { return number.uint16Value }
        if let text = config[key] as? String, let parsed = UInt16(text) { return parsed }
        throw BootstrapError.missingField(key)
    }

    private static func requireGatewayKey(_ config: [String: Any], _ key: String) throws -> Data {
        let base64 = try requireString(config, key)
        guard let data = Data(base64Encoded: base64), data.count == 32 else {
            throw BootstrapError.invalidGatewayKey
        }
        return data
    }

    /// The BSD name of the Wi-Fi interface (e.g. "en0"), or nil if none is found.
    /// Uses NWPathMonitor (sandbox-friendly) rather than SystemConfiguration.
    private static func primaryWiFiInterfaceName() -> String? {
        let monitor = NWPathMonitor(requiredInterfaceType: .wifi)
        let semaphore = DispatchSemaphore(value: 0)
        let result = OSAllocatedUnfairLock<String?>(initialState: nil)
        let queue = DispatchQueue(label: "gemina.bootstrap.wifi-lookup")
        monitor.pathUpdateHandler = { path in
            let name = path.availableInterfaces.first(where: { $0.type == .wifi })?.name
            result.withLock { $0 = name }
            semaphore.signal()
        }
        monitor.start(queue: queue)
        _ = semaphore.wait(timeout: .now() + 2)
        monitor.cancel()
        return result.withLock { $0 }
    }
}
#endif
