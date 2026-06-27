#if canImport(NetworkExtension)
import GeminaVPNCore
import Foundation
import NetworkExtension
import os.lock

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
//
// Swift 6 concurrency: `GeminaTunnelProvider` is declared `@unchecked Sendable`
// (required because `NEPacketTunnelProvider` is an ObjC class that cannot itself
// conform). The three mutable fields are annotated `nonisolated(unsafe)` — the
// Swift 6 opt-out for fields whose Sendable-ness cannot be inferred statically —
// and every read/write is serialised through `stateLock` (OSAllocatedUnfairLock)
// using `withLockUnchecked` to avoid requiring the contained types to be Sendable.

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

open class GeminaTunnelProvider: NEPacketTunnelProvider, @unchecked Sendable {

    // MARK: - Protected mutable state
    //
    // `nonisolated(unsafe)` tells the Swift 6 compiler that we know these fields
    // are accessed from multiple concurrency domains but we are managing safety
    // ourselves — every access goes through `stateLock.withLockUnchecked`, which
    // serialises without imposing a Sendable requirement on the contained types.

    private nonisolated(unsafe) var relay: DualPathRelay?
    /// Current path states + primary health, maintained from the NE path monitor
    /// and the benchmark pings; consulted by the policy on each outbound packet.
    private nonisolated(unsafe) var currentPathStates: [PathInfo] = []
    private nonisolated(unsafe) var primaryUnstable = false

    /// Serialises all reads and writes of the three mutable fields above.
    /// `withLockUnchecked` is used throughout so that non-Sendable types (DualPathRelay,
    /// PathInfo, Bool) can be read and written without requiring conformance to Sendable.
    private let stateLock = OSAllocatedUnfairLock<Void>(uncheckedState: ())

    // MARK: - Overridable factory

    /// Build the relay: the transport core (cgo bridge) plus the active path
    /// senders. Overridden by a bootstrap subclass; the default refuses to start.
    open func makeRelay() throws -> DualPathRelay {
        throw TunnelError.notConfigured
    }

    /// Build the tunnel's network settings for the gateway `tunnelRemoteAddress`.
    /// The default is route-less (the skeleton refuses to start anyway); a
    /// bootstrap subclass overrides it to install the gateway-assigned tunnel IP,
    /// routes and DNS learned during `makeRelay()`.
    open func makeTunnelSettings(tunnelRemoteAddress: String) -> NEPacketTunnelNetworkSettings {
        NEPacketTunnelNetworkSettings(tunnelRemoteAddress: tunnelRemoteAddress)
    }

    // MARK: - NEPacketTunnelProvider lifecycle

    open override func startTunnel(
        options: [String: NSObject]?,
        completionHandler: @escaping @Sendable (Error?) -> Void
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

        let built: DualPathRelay
        do {
            built = try makeRelay()
        } catch {
            completionHandler(error)
            return
        }
        stateLock.withLockUnchecked { _ in self.relay = built }

        // The subclass installs the assigned tunnel IP, routes and DNS here; the
        // route scoping (gateway-only, exclude the management subnet) lives in the
        // override (footprint contract, docs/product/footprint.md).
        let settings = makeTunnelSettings(tunnelRemoteAddress: address)
        // Capture self strongly: NEPacketTunnelProvider is system-retained for the
        // duration of startTunnel, and the closure runs exactly once, so there is no
        // cycle. A weak capture risked silently skipping readOutboundLoop (review M1).
        setTunnelNetworkSettings(settings) { error in
            if let error = error {
                completionHandler(error)
                return
            }
            self.readOutboundLoop()
            completionHandler(nil)
        }
    }

    open override func stopTunnel(
        with reason: NEProviderStopReason,
        completionHandler: @escaping () -> Void
    ) {
        stateLock.withLockUnchecked { _ in relay = nil }
        completionHandler()
    }

    // MARK: - Packet I/O

    /// Read outbound IP packets from the tunnel and duplicate each over both
    /// paths. Re-arms itself until the tunnel stops.
    private func readOutboundLoop() {
        packetFlow.readPackets { [weak self] packets, _ in
            guard let self else { return }
            // Snapshot state under the lock so the closure body runs lock-free.
            // `withLockUnchecked` avoids the Sendable requirement on DualPathRelay / PathInfo.
            let (currentRelay, pathStates, unstable): (DualPathRelay?, [PathInfo], Bool) =
                self.stateLock.withLockUnchecked { _ in
                    (self.relay, self.currentPathStates, self.primaryUnstable)
                }
            guard let currentRelay else { return }
            // currentPathStates / primaryUnstable are maintained from the NE path
            // monitor + the benchmark pings; the policy uses them to choose how
            // many paths to send each packet over (Duplicate / Failover / Smart).
            for packet in packets {
                _ = try? currentRelay.sendOutbound(packet,
                                                   pathStates: pathStates,
                                                   primaryUnstable: unstable)
            }
            self.readOutboundLoop()
        }
    }

    /// Feed a datagram received on a path through dedup/decrypt and, if it is the
    /// first copy of its logical packet, write the inner IP packet back to the
    /// tunnel. The path receivers call this.
    public func handleInbound(_ datagram: Data, path: String) {
        let currentRelay: DualPathRelay? = stateLock.withLockUnchecked { _ in self.relay }
        guard let currentRelay else { return }
        guard let payload = try? currentRelay.receiveInbound(datagram, path: path) else {
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
