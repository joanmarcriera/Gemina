#if canImport(CGeminaCore)
import CGeminaCore // SwiftPM: the C module. In Xcode the symbols come from
// the bridging header (Bridging-Header.h) instead, so the import is skipped.
#endif
import Foundation

// CoreTransport is the Swift face of the Go transport core (pkg/clientcore) over
// the C ABI (bridge/geminacore, ADR-0005). It implements TransportCore so the
// DualPathRelay / NEPacketTunnelProvider can frame, encrypt and deduplicate
// without knowing about cgo.
//
// Memory-ownership contract (per the C header): all buffers are owned Swift-side.
// The bridge copies inputs into Go memory during the call and writes results into
// the caller's output buffers; it retains no Swift pointer and returns no Go
// pointer. Sessions are addressed by an opaque handle.

public enum CoreTransportError: Error {
    case sessionCreateFailed
    case badHandle
    case bufferTooSmall
    case coreRejected
    case unknown(Int32)
}

extension CoreTransportError: LocalizedError {
    public var errorDescription: String? {
        switch self {
        case .sessionCreateFailed: return "Failed to create the transport session."
        case .badHandle: return "Invalid transport session handle."
        case .bufferTooSmall: return "Output buffer too small for the framed packet."
        case .coreRejected: return "The transport core rejected the packet."
        case .unknown(let code): return "Transport core error (code \(code))."
        }
    }
}

// Negative return codes from the C ABI (see geminacore.h).
private func coreError(_ code: Int32) -> CoreTransportError {
    switch code {
    case -1: return .badHandle
    case -2: return .bufferTooSmall
    case -3: return .coreRejected
    default: return .unknown(code)
    }
}

public final class CoreTransport: TransportCore {
    /// Role values matching the C ABI (CC_ROLE_INITIATOR / CC_ROLE_RESPONDER).
    public enum Role: Int32 {
        case initiator = 0
        case responder = 1
    }

    /// Bytes reserved above the plaintext length for the frame header + AEAD tag.
    /// Sized with generous headroom; `cc_outbound` returns `bufferTooSmall` (-2) if
    /// a future framing change ever exceeds it, so this is a safe over-allocation.
    private static let frameOverhead = 64

    private let handle: UInt64

    /// Create a session from a 16-byte session id, a 32-byte key, the role, and
    /// the inbound dedup-window capacity.
    public init(sessionID: Data, key: Data, role: Role, dedupCapacity: Int32) throws {
        precondition(sessionID.count == 16, "session id must be 16 bytes")
        precondition(key.count == 32, "key must be 32 bytes")

        var created: UInt64 = 0
        sessionID.withUnsafeBytes { sid in
            key.withUnsafeBytes { keyBytes in
                created = cc_session_new(
                    UnsafeMutablePointer(mutating: sid.bindMemory(to: UInt8.self).baseAddress),
                    UnsafeMutablePointer(mutating: keyBytes.bindMemory(to: UInt8.self).baseAddress),
                    role.rawValue,
                    dedupCapacity
                )
            }
        }
        guard created != 0 else { throw CoreTransportError.sessionCreateFailed }
        handle = created
    }

    deinit { cc_session_free(handle) }

    public func outbound(_ payload: Data) throws -> Data {
        var out = [UInt8](repeating: 0, count: payload.count + Self.frameOverhead)
        let written: Int32 = payload.withUnsafeBytes { payloadBytes in
            cc_outbound(
                handle,
                UnsafeMutablePointer(mutating: payloadBytes.bindMemory(to: UInt8.self).baseAddress),
                Int32(payload.count),
                &out,
                Int32(out.count)
            )
        }
        guard written >= 0 else { throw coreError(written) }
        return Data(out.prefix(Int(written)))
    }

    public func inbound(_ wire: Data, path: String) throws -> (payload: Data, deliver: Bool) {
        var out = [UInt8](repeating: 0, count: max(wire.count, 1))
        var deliver: Int32 = 0
        let written: Int32 = wire.withUnsafeBytes { wireBytes in
            path.withCString { cPath in
                cc_inbound(
                    handle,
                    UnsafeMutablePointer(mutating: wireBytes.bindMemory(to: UInt8.self).baseAddress),
                    Int32(wire.count),
                    UnsafeMutablePointer(mutating: cPath),
                    &out,
                    Int32(out.count),
                    &deliver
                )
            }
        }
        guard written >= 0 else { throw coreError(written) }
        return (Data(out.prefix(Int(written))), deliver == 1)
    }
}
