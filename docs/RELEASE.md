# Releasing DragZone

DragZone ships as a **signed and notarized** universal macOS `.app` inside a
DMG, published to GitHub Releases. Two GitHub Actions workflows drive this:

- **`.github/workflows/ci.yml`** — runs on every push to `main` and every PR:
  `gofmt` check, `go test ./...`, the frontend `vitest` suite, and a full
  `wails build` (so a broken build never merges).
- **`.github/workflows/release.yml`** — runs when a `v*` tag is pushed:
  builds `darwin/universal`, bundles the universal `dz` CLI, signs with the
  Developer ID + hardened runtime, builds a DMG, notarizes and staples it,
  and attaches it to a GitHub Release.

## One-time setup

You need an **Apple Developer account** ($99/yr). Gather two things.

### 1. Developer ID Application certificate

1. In Xcode (Settings › Accounts › Manage Certificates) or the Apple
   Developer portal, create a **Developer ID Application** certificate.
2. In Keychain Access, export it (with its private key) as a `.p12` and set
   an export password.
3. Base64-encode it for the secret:
   ```sh
   base64 -i DeveloperID.p12 | pbcopy
   ```
4. Find the identity string (used verbatim as a secret):
   ```sh
   security find-identity -v -p codesigning
   # e.g. "Developer ID Application: Guilherme Vozniak (AB12CD34EF)"
   ```

### 2. App-specific password (for notarization)

1. Sign in at <https://appleid.apple.com> › Sign-In and Security ›
   App-Specific Passwords → generate one (e.g. named "notarytool"). It looks
   like `abcd-efgh-ijkl-mnop`.
2. Note your **Apple ID email** and your 10-character **Team ID** (the code in
   parentheses in the signing identity from step 1, or developer.apple.com ›
   Membership).

> Alternative: an App Store Connect API key (`.p8`) also works and is what
> Apple recommends long-term. To use it, swap the notarize step in
> `release.yml` to `xcrun notarytool submit --key/--key-id/--issuer`.

### 3. Add the repository secrets

Repo › Settings › Secrets and variables › Actions → add:

The names are shared with the option-tab repo, so the same values can be set
on both (secrets are write-only on GitHub — set them from the local sources,
they cannot be copied between repos):

| Secret | Value |
| --- | --- |
| `MACOS_CERT_P12` | base64 of the Developer ID Application `.p12` |
| `MACOS_CERT_PASSWORD` | the `.p12` export password |
| `APPLE_ID` | your Apple ID email |
| `APPLE_APP_PASSWORD` | the app-specific password (`xxxx-xxxx-xxxx-xxxx`) |
| `APPLE_TEAM_ID` | your 10-character Team ID |

The CI keychain password is generated per-run; no keychain secret is needed.
Signing uses the generic `"Developer ID Application"` identity match, so no
identity-name secret is needed either.

## Cutting a release

```sh
git tag v0.3.0
git push origin v0.3.0
```

The tag version (without the leading `v`) is injected into the binary via
`-ldflags "-X main.appVersion=..."` (shown in the Updates tab) and stamped
into the app bundle's `Info.plist`. Watch the run under the repo's Actions
tab; on success a Release with the notarized `DragZone-<version>.dmg` appears.

## Notes & troubleshooting

- **App Intents / Shortcuts extension:** `build/build-appintents.sh`
  compiles `appintents/DragZoneIntents.swift` ("Add to Drop Bar" / "Run
  Dropzone Action") and embeds it as `DragZoneIntents.appex` in
  `dragzone.app/Contents/PlugIns/`. `release.yml` runs it after `wails
  build` and the identity import, before the "Sign app" step. Locally, run
  it by hand after `wails build`:
  ```sh
  cd apps/desktop
  wails build
  bash build/build-appintents.sh          # ad-hoc signs for local testing
  ```
  Pass a Developer ID identity (e.g. `bash build/build-appintents.sh
  "Developer ID Application: ..."`) to sign it for real distribution; the
  release workflow's later `codesign --deep` over the whole app re-seals it
  regardless. Whether the actions actually register in the Shortcuts app is
  a manual, logged-in-session check — it isn't exercised by CI.
- **Universal cross-compile:** the app builds `darwin/universal` on an
  arm64 runner. If cgo cross-compilation to `amd64` ever breaks, fall back to
  `-platform darwin/arm64` in `release.yml` (Apple-silicon only).
- **Entitlements:** hardened-runtime entitlements live in
  `build/darwin/entitlements.plist`. The WKWebView front end needs the JIT
  exceptions; add more only if a capability fails to notarize.
- **`create-dmg` exit code:** it can exit non-zero while still producing the
  DMG on a headless runner, so the workflow guards on the file existing.
- **Notarization failures:** if `notarytool submit --wait` reports `Invalid`,
  run `xcrun notarytool log <submission-id> --key ...` to see which nested
  binary was unsigned or lacked the hardened runtime.
