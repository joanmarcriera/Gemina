<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# i don't want wifi tethering, I want my macos connected to wifi AND then also network through the cellphone. find me the way.

You can’t make macOS “natively” load‑balance between Wi‑Fi and an Android tether at the same time; macOS will prefer one interface. To actually use both together you need: (1) a way to expose the phone to macOS as a second network interface, and (2) a channel‑bonding/VPN app (e.g. Speedify, MASV Multiconnect) that combines that interface with your Wi‑Fi.[^1][^2][^3]

***

## What macOS can and can’t do by itself

- macOS will happily have multiple interfaces up (Wi‑Fi, Ethernet, “Android USB”, “Bluetooth PAN”), but it generally routes traffic via the highest‑priority service, not truly in parallel. You can change service order for failover, but not real bonding.[^3]
- Apple explicitly notes that Android USB tethering isn’t supported out of the box, and Bluetooth tethering for Android was removed from macOS Monterey and later. On modern Apple‑Silicon macOS you must rely on third‑party drivers or apps.[^4][^5][^1]

So the pattern is: get a second pipe (Android) visible to macOS, then use a bonding app to make that and Wi‑Fi act like one logical connection.

***

## Step 1 – Expose Android as a second interface

You have three realistic paths, depending on macOS version.

### A. Bluetooth tethering (Big Sur and earlier)

If you can run Big Sur or older on one of your Macs, Bluetooth PAN is supported and Speedify/MASV explicitly document using it as a second link.[^4][^1][^3]

On Android:

- Enable Bluetooth and mobile data.
- Go to **Settings → Network \& Internet → Hotspot \& tethering → Bluetooth tethering** and toggle it on.[^4]

On macOS (Big Sur or earlier):

- Pair the phone via **System Preferences → Bluetooth**.
- In **System Preferences → Network**, add a **Bluetooth PAN** service if needed, then connect to the phone’s network.[^3][^4]

At that point macOS sees:

- Wi‑Fi: your normal WLAN.
- Bluetooth PAN: Android’s mobile data.

Speedify/Multiconnect can then bond those two.[^2][^3][^4]

### B. USB tethering via third‑party driver / script (Apple Silicon)

On newer Apple‑Silicon Macs, people are using community drivers or scripts to get Android USB tethering working again:

- Apple Support and MASV both state that Android USB tethering is not supported by macOS without extra software.[^5][^1]
- An Apple Community thread describes using HoRNDIS or similar to make Android show up as an Ethernet interface on an M‑series Mac.[^6][^1]
- A detailed video tutorial shows enabling Android USB tethering on M1/M2 by partially relaxing System Integrity Protection, installing a custom extension, approving it in **Privacy \& Security**, and then seeing the Android device appear under **Network** as a new interface.[^7]

Typical workflow (high‑level, because this involves system security changes):

- Boot to macOS recovery, adjust SIP/security policy to allow third‑party network kexts (as in the USB‑tethering video).[^7]
- Install the provided RNDIS/Android‑tethering extension (or HoRNDIS‑style driver) and approve it in **System Settings → Privacy \& Security**.[^6][^7]
- Connect Android via USB, enable **USB tethering** on the phone, then in **System Settings → Network** you’ll see a new “Android” or vendor‑named interface you can enable.[^8][^7]

Once that’s up, macOS sees:

- Wi‑Fi: normal WLAN.
- “Android USB” / Ethernet: mobile via the phone.

A bonding app can then combine them.[^9][^2][^3]

This route is powerful but carries risk (kexts + reduced SIP). I’d only use it on a machine you’re comfortable treating as “lab gear.”

### C. iPhone + Android + Wi‑Fi (if you’re willing to involve iOS)

Speedify’s own demo shows using a tethered iPhone and a tethered Android simultaneously on a Mac and combining them with Wi‑Fi.[^10][^2][^3]

- iPhone: tether via USB or Bluetooth iPhone‑USB/iPhone‑Bluetooth (natively supported).[^10][^3]
- Android: tether via Bluetooth (Big Sur) or via USB driver (Apple Silicon, per above).[^1][^7][^4]
- Mac: still on Wi‑Fi to your AP.

This gives you 2–3 independent paths, bonded by Speedify.

***

## Step 2 – Bond Wi‑Fi + phone on macOS

Once macOS sees the phone as a valid network interface, you use channel‑bonding software:

### Option 1 – Speedify (most straightforward)

Speedify is specifically built to “combine Wi‑Fi and a tethered phone” on Mac and PC, and their docs and videos walk through this exact scenario.[^2][^10][^3][^4]

On the Mac:

1. Connect to your normal Wi‑Fi as usual.[^2][^3]
2. Bring up the Android interface (Bluetooth PAN on Big Sur, or USB‑Ethernet via driver/script on Apple Silicon). Confirm both appear as connected in **System Settings → Network**.[^3][^7][^4]
3. Install Speedify from the Mac App Store or speedify.com and sign in.[^10][^2][^3]
4. Speedify automatically detects all working connections (Wi‑Fi, tethered Android, tethered iPhone, Ethernet, etc.) and lists them as separate pipes.[^2][^3]
5. In Speedify’s settings, ensure both **Wi‑Fi** and **Android** are enabled for bonding. Speedify then uses channel bonding (multi‑path VPN with packet‑level aggregation) to send/receive across both at once, giving you higher aggregate throughput and resilience.[^11][^4][^3][^2]

They explicitly say:

> “Speedify can use any internet connection that works on your computer: Wi‑Fi, Ethernet, tethered iPhones, tethered Androids… Combining Wi‑Fi and a tethered Android on a Mac is supported.”[^3][^2]

### Option 2 – MASV Multiconnect (for file‑transfer workloads)

MASV’s Multiconnect feature is similar but focused on speeding up large file transfers (it bonds connections just for that app):[^1]

- MASV Desktop can bond multiple connections (Wi‑Fi, Ethernet, tethered phone). On macOS, it notes you may need HoRNDIS or similar to make Android USB appear as a connection.[^1]
- In the MASV app, you enable **Multiconnect** and select the Wi‑Fi interface and the Android interface; MASV then uses both for its transfer traffic.[^1]

This is attractive if your main need is high‑bandwidth uploads/downloads rather than full‑system multi‑path.

***

## If you only want redundancy (not bonding)

If you only care that the phone is a backup pipe (not simultaneous use), you can do this natively:

- Bring both Wi‑Fi and Android (Bluetooth/USB) up.
- In **System Settings → Network**, set service order so Wi‑Fi is first, Android second. macOS will use Wi‑Fi unless it fails; then it will fall back to the phone.[^3]

That does not give you more bandwidth, but it does give resilience without extra software.

***

## Putting it together for your requirement

Given your constraints and risk tolerance as a technical lead:

- **Goal:** Mac on Wi‑Fi, plus an Android cellular link available at the same time, with real multi‑path capability.
- **Realistic path on current Apple Silicon:**
    - Use the USB‑tethering kext/script approach (or any maintained RNDIS driver) to expose the Android as “Android USB/Ethernet” on macOS.[^6][^7][^1]
    - Install Speedify, verify it sees both Wi‑Fi and the Android link, and enable bonding.[^10][^2][^3]

If you have a Big Sur box available and want something cleaner:

- Use Android Bluetooth tethering → Bluetooth PAN, plus Wi‑Fi; then bond them via Speedify without touching SIP or kernel extensions.[^4][^3]

If you tell me your exact macOS version(s) and whether you’re willing to relax SIP/install kexts on one machine, I can give you a step‑by‑step sequence tailored to that environment (including how to keep Wi‑Fi as default while still letting Speedify bond the Android pipe).
<span style="display:none">[^12][^13][^14][^15]</span>

<div align="center">⁂</div>

[^1]: https://help.massive.io/en/how-do-i-use-multiconnect

[^2]: https://speedify.com/blog/combining-internet-connections/how-to-combine-wi-fi-and-tethered-phone-connections/

[^3]: https://support.speedify.com/article/434-internet-connections-mac

[^4]: https://support.speedify.com/article/416-tether-android-mac-bluetooth

[^5]: https://discussions.apple.com/thread/254654422

[^6]: https://discussions.apple.com/thread/256249468

[^7]: https://www.youtube.com/watch?v=zlhc1V01EK4

[^8]: https://www.reddit.com/r/macbookpro/comments/1p73ein/internet_sharing_from_android_usb_tethering_to_mbp/

[^9]: https://support.speedify.com/article/419-combine-ethernet-tethered-android-mac

[^10]: https://www.youtube.com/watch?v=YUZTNmxsbQU\&vl=en

[^11]: https://community.spiceworks.com/t/bonding-two-networks-with-a-vpn/537303

[^12]: https://forum.netgate.com/topic/14711/dual-wan-bonding

[^13]: https://www.reddit.com/r/mac/comments/1mo2u9l/how_do_you_use_usb_tethering_from_android_to_mac/

[^14]: https://doc.ransnet.com/hsademo/layer2-vpnbonding

[^15]: https://github.com/mullvad/mullvadvpn-app/issues/8360

