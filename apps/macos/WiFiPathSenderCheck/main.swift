import Foundation
import Network
import os
import GeminaVPNPacketTunnelExtension

// Headless integration check for WiFiPathSender.
// Run with `swift run WiFiPathSenderCheck`.
// Exits 0 on full pass, non-zero on any failure.
//
// Strategy: stand up UDP echo servers on loopback via NWListener so the
// sender's datagram travels the full Network.framework path.  Each test
// creates a fresh sender (boundInterface: nil) and a fresh listener.

nonisolated(unsafe) var failures = 0
func check(_ condition: Bool, _ message: String) {
    if !condition {
        failures += 1
        print("FAIL \(message)")
    } else {
        print("PASS \(message)")
    }
}

// MARK: - Echo server helper

/// Start a UDP echo server on a random loopback port. Returns (listener, port).
func makeEchoServer() -> (NWListener, UInt16) {
    let params = NWParameters.udp
    guard let listener = try? NWListener(using: params, on: .any) else {
        fatalError("could not create loopback echo listener")
    }
    let sem = DispatchSemaphore(value: 0)
    let portBox = OSAllocatedUnfairLock<UInt16>(initialState: 0)
    let queue = DispatchQueue(label: "check.echo")

    listener.newConnectionHandler = { conn in
        conn.stateUpdateHandler = { _ in }
        conn.start(queue: queue)
        @Sendable func echo() {
            conn.receiveMessage { data, _, _, _ in
                if let data {
                    conn.send(content: data, completion: .idempotent)
                }
                echo()
            }
        }
        echo()
    }

    listener.stateUpdateHandler = { state in
        if case .ready = state {
            if let port = listener.port {
                portBox.withLock { $0 = port.rawValue }
            }
            sem.signal()
        }
    }
    listener.start(queue: queue)
    sem.wait()
    return (listener, portBox.withLock { $0 })
}

// MARK: - Test 1: send + receiveOneDatagram (loopback, no interface pin)

do {
    let (server, port) = makeEchoServer()
    defer { server.cancel() }

    var sender: WiFiPathSender?
    do {
        sender = try WiFiPathSender(gatewayHost: "127.0.0.1", gatewayPort: port, boundInterface: nil)
    } catch {
        check(false, "T1 init: \(error)")
    }

    if let sender {
        let payload = Data("hello-gemina".utf8)
        do {
            try sender.send(payload)
            let received = try sender.receiveOneDatagram(timeout: 3.0)
            check(received == payload, "T1 send+receiveOneDatagram: loopback echo matches")
        } catch {
            check(false, "T1 send/receive error: \(error)")
        }
        sender.close()
    }
}

// MARK: - Test 2: receiveLoop delivers datagrams via callback

do {
    let (server, port) = makeEchoServer()
    defer { server.cancel() }

    var sender: WiFiPathSender?
    do {
        sender = try WiFiPathSender(gatewayHost: "127.0.0.1", gatewayPort: port, boundInterface: nil)
    } catch {
        check(false, "T2 init: \(error)")
    }

    if let sender {
        let payload = Data("receive-loop-test".utf8)
        let loopSem = DispatchSemaphore(value: 0)
        let loopBox = OSAllocatedUnfairLock<Data?>(initialState: nil)

        sender.receiveLoop { data in
            loopBox.withLock { $0 = data }
            loopSem.signal()
        }

        do {
            try sender.send(payload)
        } catch {
            check(false, "T2 send error: \(error)")
        }

        let waited = loopSem.wait(timeout: .now() + 3)
        check(waited == .success, "T2 receiveLoop callback fired within 3 s")
        check(loopBox.withLock { $0 } == payload, "T2 receiveLoop delivered matching datagram")
        sender.close()
    }
}

// MARK: - Test 3: close() makes subsequent send() throw

do {
    let (server, port) = makeEchoServer()
    defer { server.cancel() }

    do {
        let sender = try WiFiPathSender(gatewayHost: "127.0.0.1", gatewayPort: port, boundInterface: nil)
        sender.close()
        var threwOnClose = false
        do {
            try sender.send(Data("after-close".utf8))
        } catch {
            // Any error after close is acceptable; we specifically expect .closed.
            threwOnClose = true
        }
        check(threwOnClose, "T3 send after close() throws")
    } catch {
        check(false, "T3 init: \(error)")
    }
}

// MARK: - Result

if failures == 0 {
    print("PASS WiFiPathSenderCheck: all checks passed")
    exit(0)
}
print("FAIL WiFiPathSenderCheck: \(failures) check(s) failed")
exit(1)
