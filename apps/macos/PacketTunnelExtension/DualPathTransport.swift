import Foundation

// Swift-side contract for the dual-path data plane (ADR-0005). These protocols
// describe the seam the NEPacketTunnelProvider fills: the Go transport core
// (`pkg/clientcore.Session`) reached over the narrow C boundary, and the two
// egress paths (a Wi-Fi-bound socket and the userspace RNDIS cellular uplink).
//
// This file is deliberately pure Swift with no NetworkExtension or cgo
// dependency so it builds and documents the architecture ahead of the real
// extension wiring. The provider supplies concrete conforming types.

/// Mirrors `pkg/clientcore.Session` across the Swift/Go boundary. The same framed
/// bytes are sent over every path; the peer deduplicates by identity.
public protocol TransportCore {
    /// Frame an outbound tunnel packet for transmission over every active path.
    func outbound(_ payload: Data) throws -> Data

    /// Decode a received datagram, returning its payload and whether this is the
    /// first copy to deliver (`true`) or a duplicate to drop (`false`).
    func inbound(_ wire: Data, path: String) throws -> (payload: Data, deliver: Bool)
}

/// A single egress path: a Wi-Fi-bound UDP socket, or the cellular RNDIS uplink.
public protocol PathSender {
    var name: String { get }
    func send(_ datagram: Data) throws
}

/// Drives the proven dual-path duplicate/deduplicate behaviour. It frames each
/// outbound packet once and sends identical copies over every active path, and
/// deduplicates inbound datagrams via the core — so one logical packet is
/// delivered to the tunnel once even when both paths carry it.
public struct DualPathRelay {
    private let core: TransportCore
    private let paths: [PathSender]

    public init(core: TransportCore, paths: [PathSender]) {
        self.core = core
        self.paths = paths
    }

    /// Duplicate one outbound tunnel packet across all active paths. Per-path
    /// send failures are collected so one dead path does not stop the others.
    public func sendOutbound(_ packet: Data) throws -> [String: Error] {
        let framed = try core.outbound(packet)
        var failures: [String: Error] = [:]
        for path in paths {
            do {
                try path.send(framed)
            } catch {
                failures[path.name] = error
            }
        }
        return failures
    }

    /// Handle one inbound datagram. Returns the payload to write back to the
    /// tunnel, or `nil` if it was a duplicate that should be dropped.
    public func receiveInbound(_ datagram: Data, path: String) throws -> Data? {
        let (payload, deliver) = try core.inbound(datagram, path: path)
        return deliver ? payload : nil
    }
}
