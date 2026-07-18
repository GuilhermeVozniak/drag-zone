#!/usr/bin/env bash
# Builds the DragZone App Intents extension ("Add to Drop Bar" / "Run
# Dropzone Action", apps/desktop/appintents/) as a real Xcode App Extension
# target via `xcodegen` + `xcodebuild`, then embeds the result as
# DragZoneIntents.appex inside build/bin/dragzone.app/Contents/PlugIns/.
#
# Metadata.appintents is what makes the extension's intents discoverable by
# Shortcuts/Siri/Spotlight: those surfaces read this STATIC bundle at
# discovery time, they do not inspect the extension binary at runtime.
#
# WHY xcodebuild instead of raw swiftc: producing Metadata.appintents
# requires several compiler-emitted inputs per architecture (the object
# file, a `.swiftconstvalues` sidecar, dependency files, stringsdata) to be
# fed to `appintentsmetadataprocessor` in exactly the shape it expects. A
# real Xcode target of type app-extension gets this wired into its build
# graph automatically (Xcode inserts an `ExtractAppIntentsMetadata` build
# phase and drives the processor itself); a prior version of this script
# hand-rolled the equivalent by compiling with `swiftc -c`, then invoking
# `appintentsmetadataprocessor` standalone with manually-assembled file
# lists. That worked on a local Xcode 26.6 but failed on GitHub's macOS
# runner with `swift-reflection-test: No such file or directory`, so
# CI-built releases embedded the extension WITHOUT discovery metadata and
# the Shortcuts actions never appeared. Building through xcodebuild against
# a real Xcode project (spec'd in appintents/project.yml, materialized by
# `xcodegen generate`) takes the same path any App-Intents-using macOS app
# takes, and is confirmed locally (Xcode 26.6) to produce Metadata.appintents
# reliably as part of the normal build - no separate best-effort step needed.
#
# Run AFTER `wails build` (which must have already produced
# build/bin/dragzone.app), and BEFORE the app bundle is codesigned for
# release: this script's own codesign call only signs the .appex itself; the
# release workflow's later `codesign --deep` over the whole app re-seals
# everything, including this extension, under the same Developer ID.
#
# Usage (from apps/desktop):
#   bash build/build-appintents.sh                    # ad-hoc sign (local dev)
#   bash build/build-appintents.sh "$IDENTITY_HASH"   # sign with a real identity (CI)
#
# Idempotent: wipes the generated .xcodeproj and its own scratch dir
# (DerivedData) plus any previously embedded .appex before rebuilding, so
# re-running is safe.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DESKTOP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
APPINTENTS_DIR="$DESKTOP_DIR/appintents"
APP_BUNDLE="$DESKTOP_DIR/build/bin/dragzone.app"
PLUGINS_DIR="$APP_BUNDLE/Contents/PlugIns"
APPEX_NAME="DragZoneIntents.appex"
SCRATCH_DIR="$DESKTOP_DIR/build/appintents-out"
DERIVED_DATA_DIR="$SCRATCH_DIR/DerivedData"
XCODEPROJ="$APPINTENTS_DIR/DragZoneIntents.xcodeproj"
SCHEME="DragZoneIntents"
IDENTITY="${1:--}"

if [ ! -d "$APP_BUNDLE" ]; then
  echo "error: $APP_BUNDLE not found - run 'wails build' first" >&2
  exit 1
fi

echo "==> ensuring xcodegen is available"
command -v xcodegen >/dev/null 2>&1 || brew install xcodegen

rm -rf "$SCRATCH_DIR" "$XCODEPROJ"
mkdir -p "$SCRATCH_DIR"

echo "==> generating Xcode project (xcodegen)"
(cd "$APPINTENTS_DIR" && xcodegen generate)

echo "==> building $SCHEME (xcodebuild, Release, universal, unsigned)"
xcodebuild \
  -project "$XCODEPROJ" \
  -scheme "$SCHEME" \
  -configuration Release \
  -derivedDataPath "$DERIVED_DATA_DIR" \
  -destination 'generic/platform=macOS' \
  CODE_SIGNING_ALLOWED=NO \
  build

# The exact "Release" vs "Release-<variant>" products subdirectory name can
# vary by Xcode version/destination, so search rather than hardcode it.
APPEX_BUILD_PATH="$(find "$DERIVED_DATA_DIR/Build/Products" -maxdepth 2 -type d -name "$APPEX_NAME" 2>/dev/null | head -n1)"
if [ -z "$APPEX_BUILD_PATH" ] || [ ! -d "$APPEX_BUILD_PATH" ]; then
  echo "error: xcodebuild reported success but did not produce $APPEX_NAME under $DERIVED_DATA_DIR/Build/Products/Release*/" >&2
  exit 1
fi
echo "==> built: $APPEX_BUILD_PATH"
file "$APPEX_BUILD_PATH/Contents/MacOS/DragZoneIntents"

# This is the whole point of building via xcodebuild rather than raw swiftc:
# confirm Xcode's own build graph produced the discovery metadata, hard-fail
# if it didn't rather than silently shipping an extension Shortcuts can't see.
METADATA_DIR="$APPEX_BUILD_PATH/Contents/Resources/Metadata.appintents"
if [ ! -d "$METADATA_DIR" ]; then
  echo "error: $APPEX_NAME was built but is missing Metadata.appintents ($METADATA_DIR) - Shortcuts/Siri/Spotlight would not discover its intents; this is the exact failure this script exists to prevent" >&2
  exit 1
fi
echo "==> Metadata.appintents present: $METADATA_DIR"

echo "==> embedding into $PLUGINS_DIR"
mkdir -p "$PLUGINS_DIR"
rm -rf "$PLUGINS_DIR/$APPEX_NAME"
cp -R "$APPEX_BUILD_PATH" "$PLUGINS_DIR/$APPEX_NAME"

echo "==> codesigning (identity: $IDENTITY)"
if [ "$IDENTITY" = "-" ]; then
  # Ad-hoc signing doesn't support a secure timestamp or the hardened
  # runtime flag combination the release identity uses.
  codesign --force --sign - "$PLUGINS_DIR/$APPEX_NAME"
else
  codesign --force --options runtime --timestamp --sign "$IDENTITY" "$PLUGINS_DIR/$APPEX_NAME"
fi

codesign -v "$PLUGINS_DIR/$APPEX_NAME"
echo "==> done: $PLUGINS_DIR/$APPEX_NAME (with Metadata.appintents)"
