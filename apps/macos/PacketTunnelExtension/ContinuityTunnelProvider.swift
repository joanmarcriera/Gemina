#if canImport(NetworkExtension)
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

open class ContinuityTunnelProvider: NEPacketTunnelProvider {
    private var relay: DualPathRelay?

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
            for packet in packets {
                // Per-path send failures are surfaced by the relay; one dead path
                // must not stop the other (that is the whole point).
                _ = try? relay.sendOutbound(packet)
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
        packetFlow.writePackets([payload], withProtocols: [NSNumber(value: AF_INET)])
    }
}
#endif
