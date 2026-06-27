import Foundation
import Security

// Minimal Keychain wrapper for the gateway entitlement token — a bearer secret,
// so it must not live in UserDefaults. Stored as a generic password, accessible
// only when the device is unlocked and never synced to iCloud Keychain. Under the
// App Sandbox the app uses its default keychain access group (its
// application-identifier), so no extra entitlement is needed for app-local use.
//
// NOTE (on-hardware follow-up): to also keep the token out of the VPN profile's
// providerConfiguration, store it in a shared keychain-access-group and set
// NETunnelProviderProtocol.passwordReference so the extension resolves it by
// persistent reference. That cross-process path can only be validated on hardware.
enum KeychainStore {
    private static let service = "com.joanmarcriera.gemina"
    private static let tokenAccount = "gateway-token"

    /// Persist (or clear, when empty) the gateway token in the Keychain.
    static func setToken(_ token: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: tokenAccount,
        ]
        guard !token.isEmpty else {
            SecItemDelete(query as CFDictionary)
            return
        }
        let data = Data(token.utf8)
        let update: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
        ]
        let status = SecItemUpdate(query as CFDictionary, update as CFDictionary)
        if status == errSecItemNotFound {
            var insert = query
            insert[kSecValueData as String] = data
            insert[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly
            SecItemAdd(insert as CFDictionary, nil)
        }
    }

    /// Read the gateway token, or "" if none is stored.
    static func token() -> String {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: tokenAccount,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var result: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let data = result as? Data,
              let token = String(data: data, encoding: .utf8) else {
            return ""
        }
        return token
    }
}
