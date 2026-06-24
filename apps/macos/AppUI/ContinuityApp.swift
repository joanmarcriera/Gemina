import AppKit
import ContinuityVPNCore
import SwiftUI

// The menu-bar app (Phase 1). It is a status-led menu bar item, matching the
// approved design. Live path state arrives later from the Network Extension; for
// now it renders the real ProtectionStatus logic over clearly-labelled preview
// data so the app builds, signs, runs and shows the intended shape.

@main
struct ContinuityApp: App {
    var body: some Scene {
        MenuBarExtra("Continuity", systemImage: "antenna.radiowaves.left.and.right") {
            StatusView()
        }
        .menuBarExtraStyle(.window)
    }
}

struct StatusView: View {
    // Preview data until the Network Extension supplies live path state.
    private let paths: [PathInfo] = [
        PathInfo(id: "Wi-Fi", up: true, metered: false),
        PathInfo(id: "Phone (cellular)", up: true, metered: true),
    ]

    var body: some View {
        let status = protectionStatus(paths: paths, paused: false, connecting: false)
        VStack(alignment: .leading, spacing: 8) {
            Text(status.headline)
                .font(.headline)
            ForEach(paths, id: \.id) { path in
                HStack(spacing: 6) {
                    Circle()
                        .fill(path.up ? Color.green : Color.secondary)
                        .frame(width: 8, height: 8)
                    Text(path.id)
                    if path.metered {
                        Text("· metered").font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
            Text("Preview data — live status arrives with the network extension.")
                .font(.caption)
                .foregroundStyle(.secondary)
            Divider()
            Button("Quit") { NSApplication.shared.terminate(nil) }
        }
        .padding(12)
        .frame(width: 280)
    }
}
