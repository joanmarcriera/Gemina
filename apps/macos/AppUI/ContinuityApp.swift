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
    // Live path state arrives from the Network Extension (Phase 3). A release build
    // therefore shows the real "off / not configured" state; only a debug build
    // carries representative preview data so the layout can be exercised without
    // the NE wired up.
    #if DEBUG
    private let paths: [PathInfo] = [
        PathInfo(id: "Wi-Fi", up: true, metered: false),
        PathInfo(id: "Phone (cellular)", up: true, metered: true),
    ]
    #else
    private let paths: [PathInfo] = []
    #endif

    var body: some View {
        let status = protectionStatus(paths: paths, paused: false, connecting: false)
        VStack(alignment: .leading, spacing: 8) {
            Text(status.headline)
                .font(.headline)
                .accessibilityAddTraits(.isHeader)
            ForEach(paths, id: \.id) { path in
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
            #if DEBUG
            Text("Preview data — live status arrives with the network extension.")
                .font(.caption)
                .foregroundStyle(.secondary)
            #endif
            Divider()
            Button("Quit") { NSApplication.shared.terminate(nil) }
        }
        .padding(12)
        .frame(minWidth: 240, idealWidth: 280, maxWidth: 360)
    }

    /// VoiceOver description for a path row — state is conveyed by colour alone in
    /// the visual, so it must be spelled out for assistive technologies.
    private func accessibilityLabel(for path: PathInfo) -> String {
        var label = "\(path.id), \(path.up ? "connected" : "disconnected")"
        if path.metered { label += ", metered" }
        return label
    }
}
