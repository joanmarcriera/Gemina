#if canImport(CContinuityCore)
import CContinuityCore // SwiftPM: the C module. In Xcode the symbols come from
// the bridging header (Bridging-Header.h) instead, so the import is skipped.
#endif
import Foundation

// CoreTransport is the Swift face of the Go transport core (pkg/clientcore) over
// the C ABI (bridge/continuitycore, ADR-0005). It implements TransportCore so the
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

// Negative return codes from the C ABI (see continuitycore.h).
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

    private let handle: UInt64

    /// Create a session from a 16-byte session id, a 32-byte key, the role, and
    /// the inbound dedup-window capacity.
    public init(sessionID: Data, key: Data, role: Role, dedupCapacity: Int32) throws {
        precondition(sessionID.count == 16, "session id must be 16 bytes")
        precondition(key.count == 32, "key must be 32 bytes")

        var created: UInt64 = 0
        sessionID.withUnsafeBytes { sid in
            key.withUnsafeBytes { k in
                created = cc_session_new(
                    UnsafeMutablePointer(mutating: sid.bindMemory(to: UInt8.self).baseAddress),
                    UnsafeMutablePointer(mutating: k.bindMemory(to: UInt8.self).baseAddress),
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
        // Header + AEAD tag headroom above the plaintext length.
        var out = [UInt8](repeating: 0, count: payload.count + 64)
        let written: Int32 = payload.withUnsafeBytes { p in
            cc_outbound(
                handle,
                UnsafeMutablePointer(mutating: p.bindMemory(to: UInt8.self).baseAddress),
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
        let written: Int32 = wire.withUnsafeBytes { w in
            path.withCString { cPath in
                cc_inbound(
                    handle,
                    UnsafeMutablePointer(mutating: w.bindMemory(to: UInt8.self).baseAddress),
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
