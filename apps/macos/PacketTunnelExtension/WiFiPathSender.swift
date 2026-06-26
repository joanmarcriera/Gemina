import Foundation
import Network
import os

// WiFiPathSender: a Wi-Fi-bound UDP egress path conforming to PathSender
// (ADR-0005). Uses Network.framework NWConnection so it runs safely under the
// app sandbox and honours macOS socket entitlements. When a boundInterface name
// is supplied the connection is pinned to that interface via
// NWParameters.requiredInterface; no pin when nil.
//
// Lifecycle:
//   1. init  — creates and starts the NWConnection (async, ready when state == .ready)
//   2. send  — writes a datagram (may be called before receiveLoop)
//   3. receiveOneDatagram — blocking single-datagram read for the WS-B handshake
//   4. receiveLoop — re-arms continuously; runs on its own serial queue
//   5. close — cancels the connection and stops any active receive loop
//
// Concurrency: all mutable state is guarded by an OSAllocatedUnfairLock.
// NWConnection's own thread-safety is relied on for the connection object itself;
// only the wrapper's auxiliary state (pendingContinuation, loopActive) needs a lock.

// MARK: - Errors

public enum WiFiPathSenderError: Error, Equatable {
    /// The connection did not reach .ready within the given deadline.
    case connectTimeout
    /// The connection entered a failed state.
    case connectionFailed(String)
    /// receiveOneDatagram timed out waiting for a datagram.
    case receiveTimeout
    /// An operation was attempted after close() was called.
    case closed
    /// No NWInterface was found matching the requested name.
    case interfaceNotFound(String)
}

// MARK: - WiFiPathSender

public final class WiFiPathSender: PathSender, @unchecked Sendable {

    // MARK: PathSender

    public let name = "wifi"

    // MARK: Private state

    private let connection: NWConnection
    private let receiveQueue: DispatchQueue
    private let stateQueue: DispatchQueue

    // Protected by lock: whether a receive loop is active and whether close was called.
    private struct MutableState {
        var isClosed: Bool = false
        var loopActive: Bool = false
        // One-shot datagram vended to receiveOneDatagram via a semaphore.
        var pendingDatagram: Result<Data, Error>? = nil
    }
    private let lock = OSAllocatedUnfairLock<MutableState>(initialState: MutableState())

    // Semaphore signalled each time a datagram arrives while pendingDatagram is nil
    // and the one-shot consumer is waiting.
    private let receiveSemaphore = DispatchSemaphore(value: 0)

    // MARK: Init

    /// Create a UDP sender/receiver to `gatewayHost`:`gatewayPort`, optionally
    /// pinned to the named network interface (e.g. "en0"). Throws if the interface
    /// name is given but cannot be resolved, or if the connection fails to reach
    /// .ready within 5 seconds.
    public init(
        gatewayHost: String,
        gatewayPort: UInt16,
        boundInterface: String?
    ) throws {
        let params = NWParameters.udp
        params.prohibitedInterfaceTypes = []

        if let ifaceName = boundInterface {
            guard let iface = WiFiPathSender.resolveInterface(named: ifaceName) else {
                throw WiFiPathSenderError.interfaceNotFound(ifaceName)
            }
            params.requiredInterface = iface
        }

        let endpoint = NWEndpoint.hostPort(
            host: NWEndpoint.Host(gatewayHost),
            port: NWEndpoint.Port(integerLiteral: gatewayPort)
        )

        self.connection = NWConnection(to: endpoint, using: params)
        self.receiveQueue = DispatchQueue(label: "gemina.wifi-path.receive", qos: .userInitiated)
        self.stateQueue = DispatchQueue(label: "gemina.wifi-path.state", qos: .userInitiated)

        // Wait for .ready (or failure) synchronously so the caller gets a usable object.
        try WiFiPathSender.awaitReady(connection: connection, queue: stateQueue)
    }

    // MARK: PathSender

    public func send(_ datagram: Data) throws {
        try lock.withLock { state in
            guard !state.isClosed else { throw WiFiPathSenderError.closed }
        }
        // NWConnection.send is thread-safe; capture is fine in a @Sendable closure.
        let result = sendSynchronously(datagram)
        if let err = result { throw err }
    }

    // MARK: Receive

    /// Blocking single-datagram read. Waits at most `timeout` seconds.
    /// Intended for handshake exchanges before the data-plane loop starts.
    public func receiveOneDatagram(timeout: TimeInterval) throws -> Data {
        try lock.withLock { state in
            guard !state.isClosed else { throw WiFiPathSenderError.closed }
        }

        // Arm the one-shot receive path, then signal is triggered by armReceive.
        armOneShotReceive()

        let deadline = DispatchTime.now() + timeout
        let waited = receiveSemaphore.wait(timeout: deadline)
        guard waited == .success else {
            throw WiFiPathSenderError.receiveTimeout
        }

        return try lock.withLock { state in
            defer { state.pendingDatagram = nil }
            switch state.pendingDatagram {
            case .success(let data): return data
            case .failure(let err): throw err
            case .none: throw WiFiPathSenderError.receiveTimeout
            }
        }
    }

    /// Re-arming receive loop. Each datagram is delivered to `onDatagram` on
    /// an internal serial queue. Calling this after close() is a no-op.
    public func receiveLoop(onDatagram: @escaping @Sendable (Data) -> Void) {
        let alreadyRunning = lock.withLock { state -> Bool in
            if state.isClosed || state.loopActive { return true }
            state.loopActive = true
            return false
        }
        guard !alreadyRunning else { return }
        receiveNext(onDatagram: onDatagram)
    }

    /// Cancel the connection and mark this sender as closed.
    public func close() {
        lock.withLock { state in
            state.isClosed = true
            state.loopActive = false
        }
        connection.cancel()
    }

    // MARK: Private helpers

    /// Send one datagram synchronously, waiting for completion.
    private func sendSynchronously(_ datagram: Data) -> Error? {
        let sem = DispatchSemaphore(value: 0)
        let errBox = OSAllocatedUnfairLock<Error?>(initialState: nil)
        connection.send(
            content: datagram,
            completion: .contentProcessed { err in
                errBox.withLock { $0 = err }
                sem.signal()
            }
        )
        sem.wait()
        return errBox.withLock { $0 }
    }

    /// Arm a single receive from the connection. Stores the result in pendingDatagram
    /// and signals receiveSemaphore so receiveOneDatagram can unblock.
    private func armOneShotReceive() {
        connection.receiveMessage { [weak self] data, _, isComplete, error in
            guard let self else { return }
            let result: Result<Data, Error>
            if let data, !data.isEmpty {
                result = .success(data)
            } else if let error {
                result = .failure(error)
            } else if isComplete {
                result = .failure(WiFiPathSenderError.closed)
            } else {
                // No data + no error + not complete: spurious, re-arm.
                self.armOneShotReceive()
                return
            }
            self.lock.withLock { state in
                state.pendingDatagram = result
            }
            self.receiveSemaphore.signal()
        }
    }

    /// Recursive re-arming data-plane receive loop.
    private func receiveNext(onDatagram: @escaping @Sendable (Data) -> Void) {
        let shouldContinue = lock.withLock { $0.loopActive }
        guard shouldContinue else { return }

        connection.receiveMessage { [weak self] data, _, isComplete, error in
            guard let self else { return }

            if let data, !data.isEmpty {
                onDatagram(data)
            }
            // On NWError the loop stops; caller observes silence and can re-connect.
            if error != nil || isComplete { return }
            self.receiveNext(onDatagram: onDatagram)
        }
    }

    /// Resolve a named network interface to an NWInterface by enumerating
    /// the path monitor's current path. Returns nil if not found.
    private static func resolveInterface(named ifaceName: String) -> NWInterface? {
        let monitor = NWPathMonitor()
        let sem = DispatchSemaphore(value: 0)
        let foundBox = OSAllocatedUnfairLock<NWInterface?>(initialState: nil)
        let q = DispatchQueue(label: "gemina.wifi-path.iface-lookup")
        monitor.pathUpdateHandler = { path in
            if let iface = path.availableInterfaces.first(where: { $0.name == ifaceName }) {
                foundBox.withLock { $0 = iface }
            }
            sem.signal()
        }
        monitor.start(queue: q)
        sem.wait()
        monitor.cancel()
        return foundBox.withLock { $0 }
    }

    /// Wait until the connection reaches .ready or fails. Times out after 5 s.
    private static func awaitReady(connection: NWConnection, queue: DispatchQueue) throws {
        let sem = DispatchSemaphore(value: 0)
        let errBox = OSAllocatedUnfairLock<Error?>(initialState: nil)
        connection.stateUpdateHandler = { state in
            switch state {
            case .ready:
                sem.signal()
            case .failed(let err):
                errBox.withLock { $0 = WiFiPathSenderError.connectionFailed(err.localizedDescription) }
                sem.signal()
            case .cancelled:
                errBox.withLock { $0 = WiFiPathSenderError.closed }
                sem.signal()
            default:
                break
            }
        }
        connection.start(queue: queue)
        let result = sem.wait(timeout: .now() + 5)
        if result == .timedOut {
            connection.cancel()
            throw WiFiPathSenderError.connectTimeout
        }
        if let err = errBox.withLock({ $0 }) { throw err }
    }
}
