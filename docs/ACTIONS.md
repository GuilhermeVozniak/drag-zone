# Writing DragZone Actions (.dzbundle)

DragZone runs Dropzone 4-compatible scriptable actions. An action is a
directory named `Something.dzbundle` containing:

```
MyAction.dzbundle/
├─ action.rb     # or action.py — the script (required)
├─ icon.png      # grid icon (optional)
└─ … any resources your script needs
```

Install by dropping the bundle on Settings → "Open Add-on Actions Folder"
(`~/Library/Application Support/DragZone/Actions/`), calling the
`InstallBundle` binding, or generate a template from Settings → Develop.

## Metadata header

The first comment block of the script declares the action:

```ruby
# Dropzone Action Info
# Name: My Action
# Description: Does something useful
# Handles: Files              # Files | Text | Files, Text
# Events: Dragged, Clicked    # optional; defaults to both
# OptionsNIB: Login           # optional config panel, see below
# OptionsTitle: Service Login # optional heading for the panel
# SkipConfig: No              # Yes = add to grid with no config step
# KeyModifiers: Command, Option
# RunsSandboxed: Yes
# Version: 1.0
# UniqueID: 1234567890        # used for the action's identity
# MinDropzoneVersion: 4.0
```

`OptionsNIB` values map to config fields whose values reach the script as
environment variables:

| OptionsNIB | Env variables |
|---|---|
| `Login` | `username`, `password` |
| `ExtendedLogin` | `server`, `port`, `username`, `password`, `remote_path` |
| `APIKey` | `api_key` |
| `UsernameAPIKey` | `username`, `api_key` |
| `ChooseFolder` | `path` |
| `ChooseApplication` | `app` |

## Entry points and inputs

Define `dragged` and/or `clicked` (matching `Events`). Inputs:

- Ruby: `$items` array / Python: `items` list — file paths, or the dropped
  text as the single element.
- `ENV['dragged_type']` — `files` or `text`.
- `ENV['KEY_MODIFIERS']` — comma-separated modifiers held during the drop.
- Saved values and option-panel fields — environment variables named by key.

## The `$dz` / `dz` API

| Method | Effect |
|---|---|
| `begin(msg)` | create/update the task's status row in the grid |
| `determinate(bool)` | switch between percent and indeterminate progress |
| `percent(0-100)` | update the progress bar |
| `finish(msg)` | set the completion message (shown as a notification) |
| `url(url)` / `url(false)` | put a URL on the clipboard (or nothing) |
| `text(text)` | put raw text on the clipboard |
| `fail(msg)` | fail the task and exit |
| `error(title, msg)` | fail with a titled error and exit |
| `alert(title, msg)` | show a notification, keep running |
| `inputbox(title, prompt)` | ask the user for a line of text (blocks) |
| `save_value(name, value)` | persist a value on this grid target (returned as `ENV[name]` next run) |
| `remove_value(name)` | delete a persisted value |
| `read_clipboard` | current clipboard text |
| `temp_folder` | a writable scratch directory for this run |
| `add_dropbar(paths)` | push result files into the Drop Bar |
| `pashua(config)` | unsupported — fails with a clear message |

Example:

```ruby
def dragged
  $dz.begin("Processing #{$items.length} item(s)...")
  $dz.determinate(true)
  $items.each_with_index do |path, i|
    # work…
    $dz.percent(((i + 1) * 100.0 / $items.length).to_i)
  end
  $dz.finish("Done")
  $dz.url(false)
end
```

Under the hood the shims speak a line protocol on stdout (see
`internal/bundles/shims.go`); plain `puts`/`print` output is ignored, so
debug freely.

## Built-in actions

Folder (copy/move, Option inverts), Open Application, AirDrop, Zip Files,
Copy to Clipboard, Move to Trash, Install Application (.dmg/.zip), Save Text,
Print, Shorten URL (TinyURL), Imgur, Amazon S3, FTP/SFTP, Google Drive
(bring-your-own OAuth Desktop-app credential), Convert Images, Remove Image
Metadata. Sources: `internal/actions/builtin/` — one file per action; they
are also the best reference for writing new built-ins in Go.

## The `dz` command line

Build once with `go build -o build/bin/dz ./cmd/dz` (talks to the running app):

```
dz list                              # grid targets
dz run NAME dragged|clicked [FILES]  # run an action by grid label
dz list-items [--json]               # Drop Bar contents
dz add [--stack] FILES               # stash files (one item each, or one stack)
dz rename INDEX NEW_NAME|--reset     # rename item (1-based index)
dz remove INDEX | lock | unlock | clear
dz open | close                      # show/hide the grid
dz open-dropbar | close-dropbar      # pop out / dock the Drop Bar
```
