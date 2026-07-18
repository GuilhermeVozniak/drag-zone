import AppIntents
import Foundation

// DragZone's App Intents extension: exposes "Add to Drop Bar" and "Run
// Dropzone Action" to the macOS Shortcuts app, Spotlight, and Siri.
//
// This extension has no logic of its own beyond locating and exec'ing the
// `dz` CLI (apps/desktop/cmd/dz), which already talks to the running
// DragZone.app over its local IPC socket (see internal/ipc). Building this
// file is documented in build/build-appintents.sh.

/// Locates and runs the bundled `dz` CLI.
enum DzCLI {
    /// Thrown when no `dz` binary can be located anywhere we look.
    struct NotFoundError: Error, CustomLocalizedStringResourceConvertible {
        var localizedStringResource: LocalizedStringResource {
            "Could not find the \u{201C}dz\u{201D} CLI. Is DragZone.app installed?"
        }
    }

    /// Thrown when `dz` runs but exits non-zero.
    struct RunError: Error, CustomLocalizedStringResourceConvertible {
        let message: String
        var localizedStringResource: LocalizedStringResource {
            LocalizedStringResource(stringLiteral: message)
        }
    }

    /// Resolution order: the copy shipped inside the host app's Resources
    /// (this extension bundle lives at
    /// DragZone.app/Contents/PlugIns/DragZoneIntents.appex, so
    /// `../../Resources/dz` from the extension's own bundle path reaches
    /// DragZone.app/Contents/Resources/dz — see the "Bundle universal dz
    /// CLI into Resources" step in release.yml), then `/usr/local/bin/dz`,
    /// then PATH.
    static func locate() -> String? {
        let fm = FileManager.default

        let bundledCandidate = Bundle.main.bundleURL
            .deletingLastPathComponent() // PlugIns
            .deletingLastPathComponent() // Contents
            .appendingPathComponent("Resources")
            .appendingPathComponent("dz")
        if fm.isExecutableFile(atPath: bundledCandidate.path) {
            return bundledCandidate.path
        }

        let systemCandidate = "/usr/local/bin/dz"
        if fm.isExecutableFile(atPath: systemCandidate) {
            return systemCandidate
        }

        if let pathEnv = ProcessInfo.processInfo.environment["PATH"] {
            for dir in pathEnv.split(separator: ":") {
                let candidate = "\(dir)/dz"
                if fm.isExecutableFile(atPath: candidate) {
                    return candidate
                }
            }
        }

        return nil
    }

    /// Runs `dz <args>` synchronously and returns trimmed stdout.
    static func run(_ args: [String]) throws -> String {
        guard let dzPath = locate() else {
            throw NotFoundError()
        }

        let process = Process()
        process.executableURL = URL(fileURLWithPath: dzPath)
        process.arguments = args

        let stdoutPipe = Pipe()
        let stderrPipe = Pipe()
        process.standardOutput = stdoutPipe
        process.standardError = stderrPipe

        try process.run()
        process.waitUntilExit()

        let outData = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
        let errData = stderrPipe.fileHandleForReading.readDataToEndOfFile()
        let outString = String(data: outData, encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let errString = String(data: errData, encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""

        guard process.terminationStatus == 0 else {
            let message = errString.isEmpty
                ? "dz exited with status \(process.terminationStatus)"
                : errString
            throw RunError(message: message)
        }

        return outString
    }
}

/// Resolves an `IntentFile` (as handed to us by Shortcuts/Siri) to an
/// on-disk absolute path suitable for passing to `dz`.
func resolveFilePath(_ file: IntentFile) throws -> String {
    guard let url = file.fileURL else {
        throw DzCLI.RunError(message: "One of the provided files has no on-disk location.")
    }
    return url.path
}

/// "Add to Drop Bar" — adds one or more files to DragZone's Drop Bar, the
/// same effect as dragging them onto it. Shells out to `dz add`.
struct AddToDropBarIntent: AppIntent {
    static var title: LocalizedStringResource = "Add to Drop Bar"
    static var description = IntentDescription(
        "Adds files to DragZone's Drop Bar, the same as dragging them onto it."
    )

    @Parameter(title: "Files")
    var files: [IntentFile]

    static var parameterSummary: some ParameterSummary {
        Summary("Add \(\.$files) to the Drop Bar")
    }

    func perform() async throws -> some IntentResult & ReturnsValue<String> {
        let paths = try files.map(resolveFilePath)
        let output = try DzCLI.run(["add"] + paths)
        let message = output.isEmpty
            ? "Added \(paths.count) item(s) to the Drop Bar."
            : output
        return .result(value: message)
    }
}

/// "Run Dropzone Action" — runs a named grid action (by its label, as shown
/// in the DragZone grid) against the given files. Shells out to `dz run
/// <name> dragged <paths>`.
struct RunDropzoneActionIntent: AppIntent {
    static var title: LocalizedStringResource = "Run Dropzone Action"
    static var description = IntentDescription(
        "Runs a DragZone grid action, by its label, against the given files."
    )

    @Parameter(title: "Action Name")
    var action: String

    @Parameter(title: "Files")
    var files: [IntentFile]

    static var parameterSummary: some ParameterSummary {
        Summary("Run \(\.$action) on \(\.$files)")
    }

    func perform() async throws -> some IntentResult & ReturnsValue<String> {
        let paths = try files.map(resolveFilePath)
        let output = try DzCLI.run(["run", action, "dragged"] + paths)
        let message = output.isEmpty
            ? "Ran \u{201C}\(action)\u{201D} on \(paths.count) item(s)."
            : output
        return .result(value: message)
    }
}

/// Publishes both intents to the Shortcuts app / Spotlight / Siri as App
/// Shortcuts, so they show up without the user having to hand-build a
/// Shortcut first.
struct DragZoneShortcuts: AppShortcutsProvider {
    static var appShortcuts: [AppShortcut] {
        AppShortcut(
            intent: AddToDropBarIntent(),
            phrases: [
                "Add to Drop Bar in \(.applicationName)",
                "Add files to \(.applicationName)"
            ],
            shortTitle: "Add to Drop Bar",
            systemImageName: "tray.and.arrow.down"
        )
        AppShortcut(
            intent: RunDropzoneActionIntent(),
            phrases: [
                "Run a Dropzone action in \(.applicationName)",
                "Run \(.applicationName) action"
            ],
            shortTitle: "Run Dropzone Action",
            systemImageName: "bolt"
        )
    }
}
