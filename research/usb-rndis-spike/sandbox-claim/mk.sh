#!/bin/bash
# mk.sh — wrap sbclaim in three signed .app bundles (unsandboxed / sandbox+usb / sandbox-no-usb).
set -e; cd "$(dirname "$0")"
mk() { local A=$1.app; rm -rf $A; mkdir -p $A/Contents/MacOS; cp sbclaim $A/Contents/MacOS/sbclaim
  /usr/libexec/PlistBuddy -c "Add :CFBundleIdentifier string dev.gemina.sbclaim.$1" -c "Add :CFBundleExecutable string sbclaim" -c "Add :CFBundlePackageType string APPL" -c "Add :LSMinimumSystemVersion string 14.0" $A/Contents/Info.plist >/dev/null
  if [ -n "$2" ]; then codesign -f -s "Apple Development" --options runtime --entitlements "$2" $A 2>/dev/null; else codesign -f -s "Apple Development" --options runtime $A 2>/dev/null; fi
  codesign --verify --strict $A && echo "$1 signed+verified"; }
mk unsandboxed ""; mk sandboxed_usb sb_usb.entitlements; mk sandboxed_nousb sb_nousb.entitlements
shasum -a 256 sbclaim */Contents/MacOS/sbclaim | awk '{print substr($1,1,16), $2}'
