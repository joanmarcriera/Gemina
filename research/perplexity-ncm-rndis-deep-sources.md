<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# Can you triple check on the deepest sections of internet where the very geeky and hacky people share this kind of stuff and see if you find something else?

The deeper geek/hacky sources mostly confirm your current direction: **CDC‑NCM over USB is the “correct” way on modern macOS**, but whether it actually yields working tethering is heavily device‑dependent, and the remaining tricks fall into three buckets: finding the right Android UI path, rooting and hand‑rolling NAT/DHCP, or grudgingly using an RNDIS kext on macOS.[^1][^2][^3][^4]

## Native NCM path (what “just works” on some phones)

Several detailed guides and threads show that on devices like Google Pixel, you can get **fully working USB tethering to macOS with no Mac hacks** by enabling CDC‑NCM and then turning on an “Ethernet tethering” toggle in Android.[^2][^3][^1]

- A Japanese deep‑dive guide confirms that macOS Big Sur and later have native CDC‑NCM support and that **Pixel phones on Android 13+ can tether to Mac by switching the USB function to `ncm` and then enabling “Ethernet tethering” in Android’s tethering settings**.[^2]
- That same article notes two key conditions: Android 13+ and a gadget HAL version ≥ v1.2, checked with `adb shell svc usb getGadgetHalVersion`; if the command errors or returns a lower version, CDC‑NCM is likely not fully supported.[^2]
- A Reddit thread on r/mac reports that **many non‑Pixel phones fail completely**, while a Pixel 7 works “seamlessly” with USB tethering to Mac; the conclusion is that proper CDC‑NCM support on the phone is the hard requirement, not a Mac issue.[^3][^1]

For phones that meet those conditions, the flow is:

1. Install platform‑tools, `adb shell svc usb setFunctions ncm`.[^5][^2]
2. On the phone, an “Ethernet tethering” switch becomes active; turn it on.[^2]
3. macOS then shows a new `enX` interface with a proper DHCP address and routing via the phone’s cellular.[^2]

## Your OnePlus: what the deep guides suggest to try next

Compared to those “happy path” Pixels, your OnePlus 12R already does the hardest part: **when you force `ncm`, macOS sees a CDC‑NCM NIC and creates an `enX` interface**, which is exactly what these guides describe as the Mac‑side success condition. The missing piece is Android actually treating that NCM link as tethered and running DHCP/NAT on it.[^4][^2]

From the Zenn article and related discussions, the next things to check on your OnePlus are:

- Whether `svc usb getGadgetHalVersion` returns a meaningful version ≥ v1.2 instead of “unknown” or an error; if it’s unknown or fails, the deep guide flags that as “likely non‑supported” for CDC‑NCM.[^2]
- Whether there is any **“Ethernet tethering” / “USB Ethernet” toggle** in the tethering UI or developer options that becomes enabled after you switch to `ncm`; on some builds it silently turns on, on others it’s a separate switch.[^2]
- Whether the Android tethering service lists NCM in its **tetherable regexes (e.g. `tetherableNcmRegexs: [ncm\\d]` or `usb\\d`)** and exposes a NCM tether type; when that exists but is not wired to the USB toggle UI, community threads suggest you’re blocked without root or OEM changes.[^6][^3]

In other words: deep sources say **macOS is already “done” once you see the NCM NIC; all remaining problems are on the Android tethering module and vendor build.**[^6][^3][^2]

## Hacky, root‑required workarounds when NCM lacks NAT

For phones that expose a USB interface (`usb0` / `ncm0`) but don’t run DHCP/NAT on it, hackers resort to rooting the phone and manually configuring tethering with `ip`/`iptables`.[^7][^8]

Two representative approaches:

- A static‑IP how‑to: one gist shows configuring the laptop with a **static IP on 192.168.42.0/24** (Android’s hard‑coded tether range) and then, on a rooted phone, running
`ip link set dev usb0 up`, `iptables -t nat -A POSTROUTING -o rmnet0 -j MASQUERADE`, and enabling `net.ipv4.ip_forward`.[^8]
That effectively turns the phone into a manual router on `usb0`, bypassing the broken tethering module.
- A more classic blog post describes building NAT rules around `usb0` on the Linux side and coordinating routing back to Android, again with hand‑written iptables scripts and rc.local hooks.[^7]

The same gist suggests installing `tcpdump` on the phone and sniffing `usb0` to debug whether packets and DHCP are flowing, plus a fallback where you run an HTTP proxy on the phone and just forward a single TCP port with `adb forward`, using the phone as a proxy rather than a full router.[^8]

All of these **require root**, and contributors are explicit that they’re for power users and not officially supported.[^7][^6][^8]

## If stuck on RNDIS: macOS‑side hacks

The deep threads also cover the opposite direction: **keep the phone on RNDIS, and hack macOS instead.** This is the HoRNDIS route and similar.[^9][^10][^4]

- A StackOverflow answer confirms that macOS supports NCM natively but not RNDIS; a third‑party driver like HoRNDIS is needed for RNDIS NICs.[^4]
- The HoRNDIS issue tracker has recent reports of people getting RNDIS tethering working on macOS Sonoma by extracting the kext, disabling SIP in recovery (`csrutil disable`), copying the kext into `/Library/Extensions`, and rebuilding kext caches.[^9]
- Authors in that thread stress that the **“correct fix” is for hardware vendors to add CDC‑NCM support**, and explicitly recommend that users ask vendors to support NCM instead of relying on HoRNDIS.[^3][^9]

This route is messy (SIP off, unsigned kexts) but it’s the main hack path if a vendor refuses to implement CDC‑NCM.

## Automation and scripting tricks

On the automation side, there are a few “geeky” tricks to smooth this out once you know what function works on your device:

- Linux users use udev rules to **auto‑run `adb shell svc usb setFunctions rndis` or `ncm` when an Android device is plugged in**, which is exactly the kind of automation you could adopt on macOS with launchd or a helper script, once you’ve settled on NCM versus RNDIS.[^11][^12]
- Android automation apps like Automate document using privileged shell to run `svc usb setFunctions <rndis|mtp|ptp|ncm>` programmatically, noting that this is unofficial and device‑specific.[^13]

These don’t solve the underlying tethering limitation, but they make it easier to re‑apply the right mode after each plug/unplug once you’ve found one that works.

## Bluetooth: confirmed dead end on modern macOS

The deeper sources match what you already suspected: **Bluetooth PAN tethering is effectively unsupported on current macOS**, even though some Android devices still offer “Bluetooth tethering.”[^14][^15][^4]

- Apple forum threads and Speedify’s documentation state that Bluetooth PAN tethering works only up to Big Sur; Monterey and later no longer support it as a general internet path.[^15][^14]
- Discussions around USB tethering repeatedly dismiss Bluetooth as too slow and now largely broken on newer macOS releases.[^4]

So the geek consensus is: don’t invest engineering effort into Bluetooth for modern macOS; USB (NCM or RNDIS-with-kext) is the realistic path.[^14][^15][^3]

***

In short, the “deepest” sources confirm your current understanding and add three extra levers: **look for an Ethernet‑tethering UI tied to NCM, consider rooted iptables/DHCP workarounds on `usb0` if you’re willing to go full hack mode, or fall back to an RNDIS kext on macOS with SIP disabled if the vendor refuses CDC‑NCM.**[^1][^3][^8][^9][^2]
<span style="display:none">[^16][^17][^18]</span>

<div align="center">⁂</div>

[^1]: https://www.reddit.com/r/mac/comments/1mo2u9l/how_do_you_use_usb_tethering_from_android_to_mac/

[^2]: https://zenn.dev/cyberrex/articles/55cd91137c3a5f

[^3]: https://nothing.community/en/d/57757-migrate-usb-tethering-from-rndis-to-cdc-ncm-for-native-macos-windows-support

[^4]: https://stackoverflow.com/questions/64235706/does-android-use-rndis-by-default-for-tethering-even-to-linux-hosts

[^5]: https://gist.github.com/SongDanielLi/7c34bd61ab0c12f7442645bc5cb96b71

[^6]: https://source.android.com/docs/core/ota/modular-system/tethering

[^7]: http://tbaba-prog.blogspot.com/2011/04/androidusbnat_30.html

[^8]: https://gist.github.com/zonia3000/80674284c27d29f71d37722793c0665b

[^9]: https://github.com/jwise/HoRNDIS/issues/169

[^10]: https://support.speedify.com/article/415-tether-android-mac-usb

[^11]: https://bbs.archlinux.org/viewtopic.php?id=263385

[^12]: https://stackoverflow.com/questions/74798790/how-to-enable-auto-usb-tethering-in-android-programmatically-or-using-shell-for

[^13]: https://llamalab.com/automate/doc/block/usb_function_set.html

[^14]: https://discussions.apple.com/thread/254761525

[^15]: https://support.speedify.com/article/416-tether-android-mac-bluetooth

[^16]: https://gist.github.com/yangfl/199e784cb4d9db127119f020b4987519

[^17]: https://github.com/hathach/tinyusb/issues/3505

[^18]: https://www.reddit.com/r/esp32/comments/1hut6g3/is_there_a_way_to_control_the_dhcp_server_of_esp/

