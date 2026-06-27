import Foundation
import GeminaVPNPacketTunnelExtension

// Headless glue check for CoreTransport.connect.
// Run with `swift run CoreTransportCheck`. Exits 0 on full pass, non-zero on any
// failure.
//
// Strategy: drive the handshake factory against the deterministic
// CGeminaCoreStubs fake (cc_handshake_begin returns a fixed ClientHello;
// cc_handshake_complete reports the assigned IP 10.99.0.5 and a fake session
// handle). This exercises the Swift plumbing — closure ordering, buffer handling
// and assigned-IP extraction — without the real Go crypto, which the Go bridge
// tests already cover.

nonisolated(unsafe) var failures = 0
func check(_ condition: Bool, _ message: String) {
    if !condition {
        failures += 1
        print("FAIL \(message)")
    } else {
        print("PASS \(message)")
    }
}

let gatewayKey = Data(repeating: 0xAB, count: 32)

// T1: connect drives begin -> send -> recv -> complete and surfaces the IP.
do {
    var order: [String] = []
    var sentHello: Data?
    let result = try CoreTransport.connect(
        gatewayPublicKey: gatewayKey,
        token: "headless-token",
        dedupCapacity: 64,
        sendClientHello: { hello in
            order.append("send")
            sentHello = hello
        },
        receiveServerHello: {
            order.append("recv")
            return Data(repeating: 0x5A, count: 122) // content ignored by the fake
        }
    )
    check(order == ["send", "recv"], "T1 connect calls sendClientHello before receiveServerHello")
    check(sentHello?.count == 8 && sentHello?.first == 0xC0, "T1 client hello bytes plumbed from the core")
    check(result.assignedIPv4 == [10, 99, 0, 5], "T1 assigned tunnel IP surfaced from the handshake")
} catch {
    check(false, "T1 connect threw unexpectedly: \(error)")
}

// T2: a receive failure propagates out of connect (the socket error is not swallowed).
struct RecvError: Error {}
do {
    _ = try CoreTransport.connect(
        gatewayPublicKey: gatewayKey,
        token: "t",
        dedupCapacity: 64,
        sendClientHello: { _ in },
        receiveServerHello: { throw RecvError() }
    )
    check(false, "T2 connect should rethrow a receive failure")
} catch is RecvError {
    check(true, "T2 connect propagates a receive failure")
} catch {
    check(false, "T2 connect threw the wrong error: \(error)")
}

if failures == 0 {
    print("PASS CoreTransportCheck: all checks passed")
} else {
    print("FAIL CoreTransportCheck: \(failures) check(s) failed")
    exit(1)
}
