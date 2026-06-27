import Foundation
import NetworkExtension

// TunnelController installs, starts and stops the Gemina packet tunnel via
// NETunnelProviderManager, and publishes the live NEVPNStatus so the menu bar can
// reflect it. The gateway endpoint, pinned identity and token are configuration
// carried in the provider profile (never hard-coded), matching the open-core
// self-host / hosted split.

enum TunnelControllerError: Error, LocalizedError {
    case notInstalled

    var errorDescription: String? {
        switch self {
        case .notInstalled:
            return "The VPN profile is not installed yet."
        }
    }
}

@MainActor
final class TunnelController: ObservableObject {
    /// The bundle id of the packet-tunnel app-extension (see project.yml).
    static let tunnelBundleID = "com.joanmarcriera.gemina.tunnel"

    @Published private(set) var status: NEVPNStatus = .invalid
    @Published private(set) var lastError: String?

    private var manager: NETunnelProviderManager?
    // nonisolated(unsafe): mutated only on the main actor (adopt), and read once by
    // the nonisolated deinit at dealloc — no concurrent access. removeObserver is
    // itself thread-safe.
    private nonisolated(unsafe) var statusObserver: NSObjectProtocol?

    init() {
        Task { await loadExisting() }
    }

    deinit {
        if let statusObserver {
            NotificationCenter.default.removeObserver(statusObserver)
        }
    }

    /// Adopt any already-installed manager so the UI reflects current state on launch.
    private func loadExisting() async {
        let managers = (try? await NETunnelProviderManager.loadAllFromPreferences()) ?? []
        if let existing = managers.first {
            adopt(existing)
        }
    }

    /// Create or update the VPN profile with the given gateway configuration and
    /// enable it. Saving triggers the one-time system approval prompt. Reloading
    /// afterwards is required before the connection can be started (Apple gotcha).
    func installIfNeeded(
        gatewayHost: String,
        gatewayPort: UInt16,
        gatewayPublicKeyBase64: String,
        token: String,
        dnsServers: [String] = []
    ) async throws {
        let managers = try await NETunnelProviderManager.loadAllFromPreferences()
        let mgr = managers.first ?? NETunnelProviderManager()

        let proto = NETunnelProviderProtocol()
        proto.providerBundleIdentifier = Self.tunnelBundleID
        proto.serverAddress = gatewayHost
        // The token is held in the Keychain by the UI (KeychainStore); it is passed
        // here into providerConfiguration, which is stored in the system-protected NE
        // configuration store (not UserDefaults). The hardened end-state moves it to
        // a shared keychain-access-group + proto.passwordReference so the raw token
        // never enters providerConfiguration — that cross-process resolution must be
        // validated on hardware (WS-F), so it is staged, not done here.
        var providerConfig: [String: Any] = [
            "port": NSNumber(value: gatewayPort),
            "gatewayPublicKey": gatewayPublicKeyBase64,
            "token": token,
        ]
        if !dnsServers.isEmpty {
            providerConfig["dnsServers"] = dnsServers
        }
        proto.providerConfiguration = providerConfig

        mgr.protocolConfiguration = proto
        mgr.localizedDescription = "Gemina"
        mgr.isEnabled = true

        try await mgr.saveToPreferences()   // one-time approval prompt
        try await mgr.loadFromPreferences() // reload so connection is usable
        adopt(mgr)
    }

    /// Start the tunnel. The provider performs the handshake and applies settings.
    func start() throws {
        guard let manager else { throw TunnelControllerError.notInstalled }
        try manager.connection.startVPNTunnel()
    }

    /// Stop the tunnel; the provider tears down its socket.
    func stop() {
        manager?.connection.stopVPNTunnel()
    }

    private func adopt(_ mgr: NETunnelProviderManager) {
        manager = mgr
        status = mgr.connection.status
        if let statusObserver {
            NotificationCenter.default.removeObserver(statusObserver)
        }
        // The notification object is the connection; read its status from there so
        // the @Sendable observer closure captures no non-Sendable manager state.
        statusObserver = NotificationCenter.default.addObserver(
            forName: .NEVPNStatusDidChange,
            object: mgr.connection,
            queue: .main
        ) { [weak self] note in
            let newStatus = (note.object as? NEVPNConnection)?.status ?? .invalid
            Task { @MainActor in self?.status = newStatus }
        }
    }
}
