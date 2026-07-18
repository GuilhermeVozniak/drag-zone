#!/usr/bin/env bash
# Compiles the DragZone App Intents extension ("Add to Drop Bar" / "Run
# Dropzone Action", apps/desktop/appintents/) and embeds it as
# DragZoneIntents.appex inside build/bin/dragzone.app/Contents/PlugIns/.
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
# Idempotent: wipes its own scratch dir and any previously embedded .appex
# before rebuilding, so re-running is safe.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DESKTOP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
APPINTENTS_DIR="$DESKTOP_DIR/appintents"
APP_BUNDLE="$DESKTOP_DIR/build/bin/dragzone.app"
PLUGINS_DIR="$APP_BUNDLE/Contents/PlugIns"
APPEX_NAME="DragZoneIntents.appex"
SCRATCH_DIR="$DESKTOP_DIR/build/appintents-out"
IDENTITY="${1:--}"
MIN_MACOS="13.0"

if [ ! -d "$APP_BUNDLE" ]; then
  echo "error: $APP_BUNDLE not found - run 'wails build' first" >&2
  exit 1
fi

rm -rf "$SCRATCH_DIR"
mkdir -p "$SCRATCH_DIR"

# Compiles the extension source for one arch; prints the output binary path
# on success. `set -e` does not abort the script when a failing command runs
# inside a tested context (`if ... ; then`), which is how the x86_64
# best-effort attempt below relies on this function being allowed to fail.
compile_arch() {
  local arch="$1"
  local out="$SCRATCH_DIR/DragZoneIntents-$arch"
  swiftc \
    -target "$arch-apple-macosx$MIN_MACOS" \
    -framework AppIntents \
    -framework Foundation \
    -O \
    -o "$out" \
    "$APPINTENTS_DIR/DragZoneIntents.swift"
  printf '%s' "$out"
}

echo "==> compiling arm64 slice"
ARM64_BIN="$(compile_arch arm64)"
ARCH_BINS=("$ARM64_BIN")

echo "==> compiling x86_64 slice (best-effort for a universal .appex)"
if X86_64_BIN="$(compile_arch x86_64 2>&1)"; then
  ARCH_BINS+=("$X86_64_BIN")
else
  echo "note: x86_64 slice failed to compile on this toolchain; shipping arm64-only extension:" >&2
  echo "$X86_64_BIN" >&2
fi

FINAL_BIN="$SCRATCH_DIR/DragZoneIntents"
if [ "${#ARCH_BINS[@]}" -gt 1 ]; then
  echo "==> lipo: assembling universal binary (${#ARCH_BINS[@]} slices)"
  lipo -create -output "$FINAL_BIN" "${ARCH_BINS[@]}"
else
  cp "${ARCH_BINS[0]}" "$FINAL_BIN"
fi
file "$FINAL_BIN"

echo "==> assembling $APPEX_NAME"
APPEX_STAGE="$SCRATCH_DIR/$APPEX_NAME"
rm -rf "$APPEX_STAGE"
mkdir -p "$APPEX_STAGE/Contents/MacOS"
cp "$APPINTENTS_DIR/Info.plist" "$APPEX_STAGE/Contents/Info.plist"
cp "$FINAL_BIN" "$APPEX_STAGE/Contents/MacOS/DragZoneIntents"
chmod +x "$APPEX_STAGE/Contents/MacOS/DragZoneIntents"

plutil -lint "$APPEX_STAGE/Contents/Info.plist"

echo "==> embedding into $PLUGINS_DIR"
mkdir -p "$PLUGINS_DIR"
rm -rf "$PLUGINS_DIR/$APPEX_NAME"
cp -R "$APPEX_STAGE" "$PLUGINS_DIR/$APPEX_NAME"

echo "==> codesigning (identity: $IDENTITY)"
if [ "$IDENTITY" = "-" ]; then
  # Ad-hoc signing doesn't support a secure timestamp or the hardened
  # runtime flag combination the release identity uses.
  codesign --force --sign - "$PLUGINS_DIR/$APPEX_NAME"
else
  codesign --force --options runtime --timestamp --sign "$IDENTITY" "$PLUGINS_DIR/$APPEX_NAME"
fi

codesign -v "$PLUGINS_DIR/$APPEX_NAME"
echo "==> done: $PLUGINS_DIR/$APPEX_NAME"
