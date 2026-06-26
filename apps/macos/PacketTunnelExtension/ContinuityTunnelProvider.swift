#if canImport(NetworkExtension)
import ContinuityVPNCore
import Foundation
import NetworkExtension

// Skeleton NEPacketTunnelProvider that drives the proven dual-path transport
// (ADR-0005). It wires the tunnel's packet flow to the DualPathRelay: outbound IP
// packets are duplicated over both paths, and datagrams received on a path are
// deduplicated and written back. The concrete pieces it needs — the TransportCore
// (the cgo bridge over pkg/clientcore) and the two PathSenders (a Wi-Fi-bound UDP
// socket and the userspace RNDIS uplink) — are supplied by `makeRelay()`, which a
// bootstrap subclass overrides once the bridge and sockets exist. Keeping them
// behind that seam lets this file compile with no cgo or socket dependency.
//
// It is guarded by `canImport(NetworkExtension)` so the package still builds where
// the framework is unavailable.

enum TunnelError: Error {
    /// makeRelay() was not overridden with a real transport core + paths yet.
    case notConfigured
    /// The VPN configuration carried no gateway address to tunnel to.
    case missingGatewayAddress
}

extension TunnelError: LocalizedError {
    // These surface to the user via NE completion handlers, so they must read as
    // sentences, not enum-case names.
    var errorDescription: String? {
        switch self {
        case .notConfigured:
            return "The VPN transport is not configured yet."
        case .missingGatewayAddress:
            return "No gateway address was provided in the VPN configuration."
        }
    }
}

open class ContinuityTunnelProvider: NEPacketTunnelProvider {
    private var relay: DualPathRelay?
    /// Current path states + primary health, maintained from the NE path monitor
    /// and the benchmark pings; consulted by the policy on each outbound packet.
    private var currentPathStates: [PathInfo] = []
    private var primaryUnstable = false

    /// Build the relay: the transport core (cgo bridge) plus the active path
    /// senders. Overridden by a bootstrap subclass; the default refuses to start.
    open func makeRelay() throws -> DualPathRelay {
        throw TunnelError.notConfigured
    }

    open override func startTunnel(
        options: [String: NSObject]?,
        completionHandler: @escaping (Error?) -> Void
    ) {
        // The gateway address is configuration (self-host or hosted), never
        // hard-coded; it travels in the VPN profile's server address.
        guard
            let address = (protocolConfiguration as? NETunnelProviderProtocol)?.serverAddress,
            !address.isEmpty
        else {
            completionHandler(TunnelError.missingGatewayAddress)
            return
        }

        let relay: DualPathRelay
        do {
            relay = try makeRelay()
        } catch {
            completionHandler(error)
            return
        }
        self.relay = relay

        // Scope routing to the gateway only — never the system default route, and
        // exclude the management subnet (footprint contract, docs/product/footprint.md).
        let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: address)
        setTunnelNetworkSettings(settings) { [weak self] error in
            if let error = error {
                completionHandler(error)
                return
            }
            self?.readOutboundLoop()
            completionHandler(nil)
        }
    }

    open override func stopTunnel(
        with reason: NEProviderStopReason,
        completionHandler: @escaping () -> Void
    ) {
        relay = nil
        completionHandler()
    }

    /// Read outbound IP packets from the tunnel and duplicate each over both
    /// paths. Re-arms itself until the tunnel stops.
    private func readOutboundLoop() {
        packetFlow.readPackets { [weak self] packets, _ in
            guard let self, let relay = self.relay else { return }
            // currentPathStates / primaryUnstable are maintained from the NE path
            // monitor + the benchmark pings; the policy uses them to choose how
            // many paths to send each packet over (Duplicate / Failover / Smart).
            for packet in packets {
                _ = try? relay.sendOutbound(packet,
                                            pathStates: self.currentPathStates,
                                            primaryUnstable: self.primaryUnstable)
            }
            self.readOutboundLoop()
        }
    }

    /// Feed a datagram received on a path through dedup/decrypt and, if it is the
    /// first copy of its logical packet, write the inner IP packet back to the
    /// tunnel. The path receivers call this.
    public func handleInbound(_ datagram: Data, path: String) {
        guard let relay = relay else { return }
        guard let payload = try? relay.receiveInbound(datagram, path: path) else {
            return // duplicate, or failed authentication — drop
        }
        // The inner packet may be IPv4 or IPv6; the tunnel must be told which, or
        // v6 traffic is mis-delivered. The IP version is the high nibble of byte 0.
        packetFlow.writePackets([payload], withProtocols: [NSNumber(value: ipFamily(of: payload))])
    }

    /// Protocol family (`AF_INET` / `AF_INET6`) for an inner IP packet, from its
    /// version nibble. Defaults to `AF_INET` for an empty buffer.
    private func ipFamily(of packet: Data) -> Int32 {
        guard let first = packet.first else { return AF_INET }
        return (first >> 4) == 6 ? AF_INET6 : AF_INET
    }
}
#endif
