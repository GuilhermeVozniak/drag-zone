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

### 2. App Store Connect API key (for notarization)

1. At <https://appstoreconnect.apple.com> › Users and Access › Integrations ›
   App Store Connect API, create a key with the **Developer** role.
2. Download the `.p8` (one-time download) and note its **Key ID** and the
   team's **Issuer ID**.
3. Base64-encode the key:
   ```sh
   base64 -i AuthKey_XXXXXXXXXX.p8 | pbcopy
   ```

### 3. Add the repository secrets

Repo › Settings › Secrets and variables › Actions → add:

| Secret | Value |
| --- | --- |
| `MACOS_CERT_P12_BASE64` | base64 of the Developer ID Application `.p12` |
| `MACOS_CERT_PASSWORD` | the `.p12` export password |
| `MACOS_SIGN_IDENTITY` | e.g. `Developer ID Application: Name (TEAMID)` |
| `KEYCHAIN_PASSWORD` | any random string (temporary CI keychain) |
| `NOTARY_KEY_P8_BASE64` | base64 of the App Store Connect `.p8` key |
| `NOTARY_KEY_ID` | the API key's Key ID |
| `NOTARY_ISSUER_ID` | the API key's Issuer ID (UUID) |

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
