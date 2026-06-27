import AppKit
import GeminaVPNCore
import NetworkExtension
import SwiftUI

// The menu-bar app. A status-led menu bar item that now drives the real packet
// tunnel via TunnelController: enter the gateway endpoint + pinned identity +
// token, toggle Protect, and the Network Extension performs the handshake and
// carries traffic. The headline reflects the live NEVPNStatus.

@main
struct GeminaApp: App {
    var body: some Scene {
        MenuBarExtra("Gemina", systemImage: "antenna.radiowaves.left.and.right") {
            StatusView()
        }
        .menuBarExtraStyle(.window)
    }
}

struct StatusView: View {
    @StateObject private var tunnel = TunnelController()
    @State private var showingAbout = false
    @State private var showingSettings = false

    // Gateway configuration. Persisted in UserDefaults for now; a production build
    // should keep the token in the Keychain, not UserDefaults.
    @AppStorage("gemina.gatewayHost") private var gatewayHost = ""
    @AppStorage("gemina.gatewayPort") private var gatewayPort = "51820"
    @AppStorage("gemina.gatewayPublicKey") private var gatewayPublicKey = ""
    @AppStorage("gemina.token") private var token = ""

    /// The reason for the name — shown in the "About Gemina" dialog. A Legio
    /// Gemina was two understrength Roman legions merged into one "twin" legion so
    /// the force always arrived at full strength; Gemina sends every packet down
    /// two paths and keeps the first copy through, for the same reason.
    private static let nameStory = """
    Gemina is Latin for “twinned”. When a Roman legion was too depleted to be sure \
    of holding the line, two were merged into one — a Legio Gemina, the twin legion \
    — so the force always arrived at full strength.

    Gemina does the same with your connection: it sends every packet down two paths \
    at once and keeps whichever copy arrives first, so you stay online even when one \
    path drops.
    """

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(status.headline)
                .font(.headline)
                .accessibilityAddTraits(.isHeader)

            ForEach(activePaths, id: \.id) { path in
                pathRow(path)
            }

            Toggle("Protect", isOn: protectBinding)
                .toggleStyle(.switch)
                .disabled(!isConfigured)
                .accessibilityHint(isConfigured ? "" : "Enter the gateway settings first.")

            if let error = tunnel.lastError {
                Text(error).font(.caption).foregroundStyle(.red)
            }

            DisclosureGroup("Gateway settings", isExpanded: $showingSettings) {
                settingsForm
            }
            .font(.caption)

            Divider()
            Text("Gemina — the twin legion: every packet travels two paths; the first to arrive wins.")
                .font(.caption)
                .foregroundStyle(.secondary)
            HStack {
                Button("About Gemina") { showingAbout = true }
                Spacer()
                Button("Quit") { NSApplication.shared.terminate(nil) }
            }
        }
        .padding(12)
        .frame(minWidth: 260, idealWidth: 300, maxWidth: 380)
        .alert("Gemina", isPresented: $showingAbout) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(Self.nameStory)
        }
    }

    // MARK: - Derived status

    /// The protection status shown in the headline, derived from the live tunnel
    /// state. Per-path detail arrives from the extension later; for now a connected
    /// tunnel is the single Wi-Fi path (no redundancy until the cellular path lands).
    private var status: ProtectionStatus {
        switch tunnel.status {
        case .connecting, .reasserting, .disconnecting:
            return .connecting
        case .connected:
            return protectionStatus(paths: activePaths, paused: false, connecting: false)
        default:
            return .off
        }
    }

    private var activePaths: [PathInfo] {
        tunnel.status == .connected ? [PathInfo(id: "Wi-Fi", up: true, metered: false)] : []
    }

    private var isConfigured: Bool {
        !gatewayHost.isEmpty && !gatewayPublicKey.isEmpty && !token.isEmpty
    }

    private var protectBinding: Binding<Bool> {
        Binding(
            get: { [tunnel] in
                let s = tunnel.status
                return s == .connected || s == .connecting || s == .reasserting
            },
            set: { on in
                if on { connect() } else { tunnel.stop() }
            }
        )
    }

    // MARK: - Actions

    private func connect() {
        Task {
            do {
                try await tunnel.installIfNeeded(
                    gatewayHost: gatewayHost,
                    gatewayPort: UInt16(gatewayPort) ?? 51820,
                    gatewayPublicKeyBase64: gatewayPublicKey,
                    token: token
                )
                try tunnel.start()
            } catch {
                // Errors also surface via tunnel.lastError; this is a no-op catch so
                // the toggle simply falls back to off.
            }
        }
    }

    // MARK: - Subviews

    @ViewBuilder
    private func pathRow(_ path: PathInfo) -> some View {
        HStack(spacing: 6) {
            Circle()
                .fill(path.up ? Color.green : Color.secondary)
                .frame(width: 8, height: 8)
                .accessibilityHidden(true) // decorative; the row carries the label
            Text(path.id)
            if path.metered {
                Text("· metered").font(.caption).foregroundStyle(.secondary)
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(accessibilityLabel(for: path))
    }

    @ViewBuilder
    private var settingsForm: some View {
        VStack(alignment: .leading, spacing: 4) {
            TextField("Gateway host", text: $gatewayHost)
            TextField("Port", text: $gatewayPort)
            TextField("Gateway public key (base64)", text: $gatewayPublicKey)
            TextField("Token", text: $token)
        }
        .textFieldStyle(.roundedBorder)
        .font(.caption)
        .padding(.vertical, 4)
    }

    /// VoiceOver description for a path row — state is conveyed by colour alone in
    /// the visual, so it must be spelled out for assistive technologies.
    private func accessibilityLabel(for path: PathInfo) -> String {
        var label = "\(path.id), \(path.up ? "connected" : "disconnected")"
        if path.metered { label += ", metered" }
        return label
    }
}
