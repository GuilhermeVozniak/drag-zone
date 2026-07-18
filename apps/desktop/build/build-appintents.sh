#!/usr/bin/env bash
# Compiles the DragZone App Intents extension ("Add to Drop Bar" / "Run
# Dropzone Action", apps/desktop/appintents/), generates its
# Metadata.appintents bundle, and embeds both as DragZoneIntents.appex inside
# build/bin/dragzone.app/Contents/PlugIns/.
#
# Metadata.appintents is what makes the extension's intents discoverable by
# Shortcuts/Siri/Spotlight: those surfaces read this STATIC bundle at
# discovery time, they do not inspect the extension binary at runtime. It is
# produced by `appintentsmetadataprocessor` from two compiler-emitted inputs
# per architecture: the object file and a `.swiftconstvalues` sidecar (the
# compile-time-constant literals - titles, phrases, parameter shapes - that
# `-emit-const-values-path` extracts). Getting the sidecar written at all
# requires two things a plain `swiftc -o binary` build does not do:
#   1. Compiling with `-c` (object-only) rather than compile+link in one
#      invocation - the driver only honors a supplementary output path when
#      there is a single discrete compile job; in one-shot compile+link mode
#      it silently writes the sidecar to a temp dir and never copies it out.
#      Linking is then a separate `swiftc <object>.o -o <binary>` step.
#   2. Passing `-Xfrontend -const-gather-protocols-file -Xfrontend <file>`
#      listing the AppIntents protocols to extract conformances for -
#      without it the frontend has nothing to gather and writes no sidecar
#      at all, even with `-emit-const-values-path` present.
# Note the driver also ignores the exact filename passed to
# `-emit-const-values-path` in `-c` mode: it always names the sidecar after
# the object file's basename in the same directory, so this script always
# reads it back from `<scratch>/<arch>/DragZoneIntents.swiftconstvalues`
# rather than trusting the path it requested.
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
MODULE_NAME="DragZoneIntents"

if [ ! -d "$APP_BUNDLE" ]; then
  echo "error: $APP_BUNDLE not found - run 'wails build' first" >&2
  exit 1
fi

rm -rf "$SCRATCH_DIR"
mkdir -p "$SCRATCH_DIR"

# Protocols whose conformances the compiler should extract compile-time
# constant values for. This is the AppIntents vocabulary Xcode's own
# extension template feeds `-const-gather-protocols-file`; harmless to list
# ones this extension doesn't use.
CONST_PROTOCOLS_FILE="$SCRATCH_DIR/const-gather-protocols.json"
cat >"$CONST_PROTOCOLS_FILE" <<'EOF'
["AppIntent","AppEntity","AppEnum","EntityQuery","EntityPropertyQuery","AppShortcutsProvider","TransientAppEntity","AssistantIntent","AssistantEntity"]
EOF

# Compiles + links the extension source for one arch. `set -e` does not
# abort the script when a failing command runs inside a tested context
# (`if ... ; then`), which is how the x86_64 best-effort attempt below
# relies on this function being allowed to fail.
compile_arch() {
  local arch="$1"
  local arch_dir="$SCRATCH_DIR/$arch"
  mkdir -p "$arch_dir"
  # -c (object-only): the only mode in which the driver reliably emits a
  # discoverable .swiftconstvalues sidecar (see header comment).
  swiftc -c \
    -target "$arch-apple-macosx$MIN_MACOS" \
    -module-name "$MODULE_NAME" \
    -framework AppIntents \
    -framework Foundation \
    -O \
    -emit-const-values-path "$arch_dir/DragZoneIntents.swiftconstvalues" \
    -Xfrontend -const-gather-protocols-file -Xfrontend "$CONST_PROTOCOLS_FILE" \
    -o "$arch_dir/DragZoneIntents.o" \
    "$APPINTENTS_DIR/DragZoneIntents.swift"
  # Separate link step: swiftc happily links a pre-compiled .o directly.
  swiftc \
    -target "$arch-apple-macosx$MIN_MACOS" \
    -framework AppIntents \
    -framework Foundation \
    -O \
    -o "$arch_dir/DragZoneIntents" \
    "$arch_dir/DragZoneIntents.o"
  printf '%s' "$arch_dir/DragZoneIntents"
}

echo "==> compiling arm64 slice"
ARM64_BIN="$(compile_arch arm64)"
ARCH_BINS=("$ARM64_BIN")
ARCHS=("arm64")

echo "==> compiling x86_64 slice (best-effort for a universal .appex)"
if X86_64_BIN="$(compile_arch x86_64 2>&1)"; then
  ARCH_BINS+=("$X86_64_BIN")
  ARCHS+=("x86_64")
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
mkdir -p "$APPEX_STAGE/Contents/MacOS" "$APPEX_STAGE/Contents/Resources"
cp "$APPINTENTS_DIR/Info.plist" "$APPEX_STAGE/Contents/Info.plist"
cp "$FINAL_BIN" "$APPEX_STAGE/Contents/MacOS/DragZoneIntents"
chmod +x "$APPEX_STAGE/Contents/MacOS/DragZoneIntents"

plutil -lint "$APPEX_STAGE/Contents/Info.plist"

# Metadata.appintents generation is BEST-EFFORT: appintentsmetadataprocessor
# shells out to `swift-reflection-test` from the *active* toolchain, and on
# at least one combination of GitHub's macos-14 runner image and an older
# Xcode this fails with "swift-reflection-test: No such file or directory"
# even though compiling, linking, and assembling the .appex all succeed. That
# failure has nothing to do with whether the extension itself works - it only
# means Shortcuts/Siri/Spotlight can't discover its intents ahead of time.
# Losing discovery metadata on a bad CI toolchain is far better than aborting
# the whole release, so this step (and its const-values-list prerequisite,
# which only matters for this step) is wrapped in an explicit guard: `set -e`
# does not abort the script for a failing command tested by `if ... ; then`,
# so run_appintents_metadata's internal failures can't trip errexit here,
# while every step outside this guard (compile/lipo/assemble/embed/codesign)
# still hard-fails as before.
#
# APPINTENTS_METADATA_TOOL overrides which processor binary is invoked - unset
# it (or leave it as the default) for normal use; it exists so this
# best-effort path can be exercised on demand, e.g.
# `APPINTENTS_METADATA_TOOL=/bin/false bash build/build-appintents.sh` to
# simulate a toolchain that rejects the processor.
APPINTENTS_METADATA_TOOL="${APPINTENTS_METADATA_TOOL:-appintentsmetadataprocessor}"

run_metadata_processor() {
  if [[ "$APPINTENTS_METADATA_TOOL" = /* ]]; then
    "$APPINTENTS_METADATA_TOOL" "$@"
  else
    xcrun "$APPINTENTS_METADATA_TOOL" "$@"
  fi
}

run_appintents_metadata() {
  local source_file_list="$SCRATCH_DIR/source-files.txt"
  printf '%s\n' "$APPINTENTS_DIR/DragZoneIntents.swift" >"$source_file_list"

  local const_vals_list="$SCRATCH_DIR/const-vals.txt"
  : >"$const_vals_list"
  local arch
  for arch in "${ARCHS[@]}"; do
    printf '%s\n' "$SCRATCH_DIR/$arch/DragZoneIntents.swiftconstvalues" >>"$const_vals_list"
  done

  local target_triple_args=()
  for arch in "${ARCHS[@]}"; do
    target_triple_args+=(--target-triple "$arch-apple-macos$MIN_MACOS")
  done

  local xcode_developer_dir toolchain_dir sdk_root xcode_version bundle_id
  xcode_developer_dir="$(xcode-select -p)"
  toolchain_dir="$xcode_developer_dir/Toolchains/XcodeDefault.xctoolchain"
  sdk_root="$(xcrun --show-sdk-path)"
  xcode_version="$(defaults read "$(dirname "$xcode_developer_dir")/version.plist" ProductBuildVersion)"
  bundle_id="$(plutil -extract CFBundleIdentifier raw "$APPINTENTS_DIR/Info.plist")"

  run_metadata_processor \
    --toolchain-dir "$toolchain_dir" \
    --module-name "$MODULE_NAME" \
    --sdk-root "$sdk_root" \
    --xcode-version "$xcode_version" \
    --platform-family macOS \
    --deployment-target "$MIN_MACOS" \
    "${target_triple_args[@]}" \
    --source-file-list "$source_file_list" \
    --swift-const-vals-list "$const_vals_list" \
    --binary-file "$APPEX_STAGE/Contents/MacOS/DragZoneIntents" \
    --bundle-identifier "$bundle_id" \
    --output "$APPEX_STAGE/Contents/Resources"

  [ -d "$APPEX_STAGE/Contents/Resources/Metadata.appintents" ]
}

echo "==> generating Metadata.appintents"
METADATA_OK=1
if run_appintents_metadata; then
  echo "==> Metadata.appintents OK"
else
  METADATA_OK=0
  echo "WARNING: Metadata.appintents generation failed on this toolchain (swift-reflection-test/appintentsmetadataprocessor); embedding extension WITHOUT discovery metadata - regenerate on a compatible Xcode for Shortcuts discovery" >&2
fi

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
if [ "$METADATA_OK" -eq 1 ]; then
  echo "==> done: $PLUGINS_DIR/$APPEX_NAME (with Metadata.appintents)"
else
  echo "==> done: $PLUGINS_DIR/$APPEX_NAME (WITHOUT Metadata.appintents - see warning above)"
fi
