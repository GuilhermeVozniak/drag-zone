// Stateful in-memory stand-in for the generated Wails bindings, driven by
// Playwright against a real (built) React app — no Go process involved.
//
// It implements exactly the surface `src/lib/backend.ts` imports:
//   - every named export of `wailsjs/go/main/App` it calls (see App.d.ts)
//   - `EventsOn`/`EventsEmit`/`EventsOff` and `OnFileDrop`/`OnFileDropOff`
//     from `wailsjs/runtime/runtime` (the latter two are used by
//     `src/lib/dnd.ts`, not backend.ts, but live in the same module)
//
// vite.config.ts aliases both `wailsjs/go/main/App` and
// `wailsjs/runtime/runtime` to THIS file under `--mode e2e`, so both of
// backend.ts's imports resolve to the same module instance — required so
// the App.* mutators and the EventsOn/EventsEmit bus share one event map.
//
// Method calls mutate the in-memory state below and emit the same events
// the real `App` facade does (see apps/desktop/app.go's `Event*` constants
// and app_grid.go/app_dropbar.go's `a.emit(...)` calls), so the real React
// app (hooks in src/hooks/useBackend.ts) re-renders exactly as it would
// against the live backend.

// ---------------------------------------------------------------------------
// Local shapes (mirrors wailsjs/go/models.ts; not imported so this module has
// zero dependency on generated bindings and builds without them on CI).

interface OptionField {
  key: string;
  label: string;
  type: string;
  placeholder?: string;
  choices?: string[];
  default?: string;
  required?: boolean;
}

interface ActionSpec {
  id: string;
  name: string;
  description: string;
  icon: string;
  category: string;
  events: string[];
  accepts: string[];
  options?: OptionField[];
  multi: boolean;
  keyModifier?: string;
}

interface Target {
  id: string;
  actionId: string;
  label: string;
  options?: Record<string, string>;
  position: number;
  shortcut?: string;
}

interface DropBarItem {
  id: string;
  kind: string;
  paths?: string[];
  text?: string;
  label: string;
  locked: boolean;
  addedAt: string;
}

interface TaskState {
  id: string;
  targetId: string;
  title: string;
  detail?: string;
  percent: number;
  status: string;
  error?: string;
  resultUrl?: string;
  startedAt: string;
  finishedAt: string;
}

interface Settings {
  launchAtLogin: boolean;
  globalShortcut: string;
  popOutShortcut: string;
  gridColumns: number;
  gridSize: number;
  theme: string;
  animateGrid: boolean;
  showKeyOverlays: boolean;
  playSounds: boolean;
  dragOverlay: boolean;
  dropBarKeepsItems: boolean;
  notifyOnComplete: boolean;
  autoUpdateCheck: boolean;
  onboardingSeen: boolean;
  lastUpdateNotified?: string;
}

interface Payload {
  kind: string;
  paths?: string[];
  text?: string;
  modifiers?: string[];
}

interface Share {
  title: string;
  url: string;
  at: string;
}

interface AddonInfo {
  name: string;
  installed: boolean;
}

interface UpdateInfo {
  version: string;
  latest: string;
  available: boolean;
  url: string;
  downloadUrl: string;
  publishedAt: string;
}

// ---------------------------------------------------------------------------
// wailsjs/runtime/runtime — event bus + file-drop no-ops.

type Listener = (...args: unknown[]) => void;

const listeners = new Map<string, Set<Listener>>();

export function EventsOn(eventName: string, callback: Listener): () => void {
  let set = listeners.get(eventName);
  if (!set) {
    set = new Set();
    listeners.set(eventName, set);
  }
  set.add(callback);
  return () => {
    listeners.get(eventName)?.delete(callback);
  };
}

export function EventsEmit(eventName: string, ...data: unknown[]): void {
  for (const cb of listeners.get(eventName) ?? []) cb(...data);
}

export function EventsOnce(eventName: string, callback: Listener): () => void {
  const off = EventsOn(eventName, (...data) => {
    off();
    callback(...data);
  });
  return off;
}

export function EventsOff(eventName: string, ...additionalEventNames: string[]): void {
  listeners.delete(eventName);
  for (const name of additionalEventNames) listeners.delete(name);
}

export function EventsOffAll(): void {
  listeners.clear();
}

// Native file drops never happen in a browser-driven e2e run (there is no
// Wails webview delivering OS drag events); `src/lib/dnd.ts` only registers
// the callback, so a no-op is a faithful stand-in.
export function OnFileDrop(
  _callback: (x: number, y: number, paths: string[]) => void,
  _useDropTarget: boolean,
): void {}
export function OnFileDropOff(): void {}

// ---------------------------------------------------------------------------
// In-memory state + seed data.

let nextId = 1;
const genId = (prefix: string) => `${prefix}-${nextId++}`;

// Test-only hook: `?onboarding=1` seeds a fresh first run (onboardingSeen
// false) so dropbar-settings.spec.ts can drive the onboarding carousel
// (src/features/onboarding/Onboarding.tsx, gated on this flag in
// src/App.tsx's `showOnboarding`) without it popping up in every other spec,
// which needs the grid to render immediately. Evaluated once at module init,
// so it only takes effect on the navigation that requested it.
const forceOnboarding =
  typeof window !== "undefined" &&
  new URLSearchParams(window.location.search).get("onboarding") === "1";

let settings: Settings = {
  launchAtLogin: false,
  globalShortcut: "F3",
  popOutShortcut: "F4",
  gridColumns: 4,
  gridSize: 33,
  theme: "system",
  animateGrid: true,
  showKeyOverlays: true,
  playSounds: true,
  dragOverlay: true,
  dropBarKeepsItems: false,
  notifyOnComplete: true,
  autoUpdateCheck: false,
  // Seeded true so the e2e app opens straight into the grid instead of the
  // first-run onboarding carousel (see src/App.tsx's `showOnboarding`).
  onboardingSeen: !forceOnboarding,
};

// Mirrors the specs the real Go registry exposes for a handful of built-in
// actions (see apps/desktop/internal/actions/builtin/*.go `Spec()`), enough
// to exercise the "+" add menu and tile rendering realistically.
const specs: ActionSpec[] = [
  {
    id: "folder",
    name: "Folder",
    description: "Copy or move dropped files to a folder. Click to open it in Finder.",
    icon: "folder",
    category: "File Management",
    events: ["dragged", "clicked"],
    accepts: ["files", "text", "url"],
    multi: true,
    keyModifier: "option",
    options: [
      { key: "path", label: "Folder", type: "folder", required: true },
      { key: "mode", label: "On drop", type: "select", choices: ["copy", "move"], default: "copy" },
    ],
  },
  {
    id: "zip",
    name: "Zip Files",
    description: "Compress dropped files into a .zip archive.",
    icon: "archive",
    category: "Utilities",
    events: ["dragged"],
    accepts: ["files"],
    multi: false,
    options: [
      {
        key: "dest",
        label: "Save archive in",
        type: "select",
        choices: ["same folder", "Desktop"],
        default: "same folder",
      },
    ],
  },
  {
    id: "trash",
    name: "Move to Trash",
    description: "Move dropped files to the Trash.",
    icon: "trash-2",
    category: "File Management",
    events: ["dragged"],
    accepts: ["files"],
    multi: false,
  },
  {
    id: "airdrop",
    name: "AirDrop",
    description: "Share dropped files with nearby devices via AirDrop.",
    icon: "wifi",
    category: "Sharing",
    events: ["dragged"],
    accepts: ["files"],
    multi: false,
  },
  {
    id: "copy-to-clipboard",
    name: "Copy to Clipboard",
    description: "Copy dropped text, URLs, or file paths to the clipboard.",
    icon: "clipboard-copy",
    category: "Utilities",
    events: ["dragged"],
    accepts: ["files", "text", "url"],
    multi: false,
  },
  {
    id: "shorten-url",
    name: "Shorten URL",
    description:
      "Shorten a dropped URL with TinyURL and copy it. Click to shorten the clipboard URL.",
    icon: "link",
    category: "Utilities",
    events: ["dragged", "clicked"],
    accepts: ["url", "text"],
    multi: false,
  },
];

// A couple of folders plus a few actions, like a freshly set-up grid.
let targets: Target[] = [
  {
    id: genId("target"),
    actionId: "folder",
    label: "Downloads",
    options: { path: "/Users/demo/Downloads", mode: "copy" },
    position: 0,
  },
  {
    id: genId("target"),
    actionId: "folder",
    label: "Desktop",
    options: { path: "/Users/demo/Desktop", mode: "move" },
    position: 1,
  },
  {
    id: genId("target"),
    actionId: "zip",
    label: "Zip Files",
    options: { dest: "same folder" },
    position: 2,
  },
  { id: genId("target"), actionId: "trash", label: "Move to Trash", options: {}, position: 3 },
  { id: genId("target"), actionId: "airdrop", label: "AirDrop", options: {}, position: 4 },
];

let dropBarItems: DropBarItem[] = [
  {
    id: genId("item"),
    kind: "url",
    text: "https://example.com/shared-link",
    label: "Shared link",
    locked: false,
    addedAt: new Date().toISOString(),
  },
];

let tasks: TaskState[] = [];
let shares: Share[] = [];
const addons: AddonInfo[] = [
  { name: "google-drive", installed: false },
  { name: "s3-upload", installed: false },
];

const emitGrid = () =>
  EventsEmit(
    "grid:changed",
    targets.slice().sort((a, b) => a.position - b.position),
  );
const emitDropBar = () => EventsEmit("dropbar:changed", dropBarItems.slice());
const emitTasks = () => EventsEmit("tasks:changed", tasks.slice());
const emitShares = () => EventsEmit("shares:changed", shares.slice());

// ---------------------------------------------------------------------------
// wailsjs/go/main/App

export async function GetSettings(): Promise<Settings> {
  return { ...settings };
}

export async function SetSettings(s: Settings): Promise<void> {
  settings = { ...s };
}

export async function ActionSpecs(): Promise<ActionSpec[]> {
  return specs.slice();
}

export async function InstallBundle(_path: string): Promise<void> {}

export async function OpenActionsFolder(): Promise<void> {}

export async function DevelopAction(_name: string, _template: string): Promise<void> {}

export async function Targets(): Promise<Target[]> {
  return targets.slice().sort((a, b) => a.position - b.position);
}

export async function AddTarget(
  actionId: string,
  label: string,
  options: Record<string, string>,
): Promise<Target> {
  const t: Target = {
    id: genId("target"),
    actionId,
    label,
    options: { ...options },
    position: targets.length,
  };
  targets.push(t);
  emitGrid();
  return t;
}

export async function AddTargetsFromPaths(paths: string[]): Promise<void> {
  for (const p of paths) {
    const isApp = p.endsWith(".app");
    const name = (p.split("/").filter(Boolean).pop() ?? p).replace(/\.app$/, "");
    targets.push({
      id: genId("target"),
      actionId: isApp ? "open-app" : "folder",
      label: name,
      options: isApp ? { app: p } : { path: p, mode: "copy" },
      position: targets.length,
    });
  }
  emitGrid();
}

export async function UpdateTarget(t: Target): Promise<void> {
  const idx = targets.findIndex((x) => x.id === t.id);
  if (idx >= 0) targets[idx] = { ...t };
  emitGrid();
}

export async function DuplicateTarget(id: string): Promise<Target> {
  const src = targets.find((t) => t.id === id);
  if (!src) throw new Error(`target not found: ${id}`);
  const copy: Target = {
    ...src,
    id: genId("target"),
    position: targets.length,
    options: { ...src.options },
  };
  targets.push(copy);
  emitGrid();
  return copy;
}

export async function RemoveTarget(id: string): Promise<void> {
  targets = targets.filter((t) => t.id !== id);
  emitGrid();
}

export async function MoveTarget(id: string, position: number): Promise<void> {
  const ordered = targets.slice().sort((a, b) => a.position - b.position);
  const idx = ordered.findIndex((x) => x.id === id);
  if (idx < 0) return;
  const [t] = ordered.splice(idx, 1);
  const clamped = Math.max(0, Math.min(position, ordered.length));
  ordered.splice(clamped, 0, t);
  ordered.forEach((x, i) => {
    x.position = i;
  });
  targets = ordered;
  emitGrid();
}

/** Starts a fake task that finishes shortly after, like the real runner. */
function startTask(
  target: Target | undefined,
  spec: ActionSpec | undefined,
  detail?: string,
): TaskState {
  const id = genId("task");
  const now = new Date().toISOString();
  const task: TaskState = {
    id,
    targetId: target?.id ?? "",
    title: target?.label ?? spec?.name ?? "Action",
    detail,
    percent: 0,
    status: "running",
    startedAt: now,
    finishedAt: "",
  };
  tasks = [task, ...tasks];
  emitTasks();
  setTimeout(() => {
    const t = tasks.find((x) => x.id === id);
    if (!t) return;
    t.percent = 100;
    t.status = "done";
    t.finishedAt = new Date().toISOString();
    // Mirrors the real backend (internal/tasks/runner.go's OnResultURL ->
    // app.go's addRecentShare): an action producing a shareable URL — here,
    // shorten-url — sets the task's resultUrl and surfaces it in Recently
    // Shared, instead of just finishing silently.
    if (spec?.id === "shorten-url") {
      t.resultUrl = "https://tinyurl.com/e2e-demo";
      shares = [{ title: t.title, url: t.resultUrl, at: t.finishedAt }, ...shares].slice(0, 10);
      emitShares();
    }
    emitTasks();
  }, 50);
  return task;
}

export async function DropOnTarget(targetId: string, payload: Payload): Promise<string> {
  const target = targets.find((t) => t.id === targetId);
  const spec = specs.find((s) => s.id === target?.actionId);
  return startTask(target, spec, payload.paths?.[0] ?? payload.text).id;
}

export async function ClickTarget(targetId: string): Promise<string> {
  const target = targets.find((t) => t.id === targetId);
  const spec = specs.find((s) => s.id === target?.actionId);
  return startTask(target, spec).id;
}

export async function Tasks(): Promise<TaskState[]> {
  return tasks.slice();
}

export async function DismissTask(id: string): Promise<void> {
  tasks = tasks.filter((t) => t.id !== id);
  emitTasks();
}

export async function CancelTask(id: string): Promise<void> {
  tasks = tasks.filter((t) => t.id !== id);
  emitTasks();
}

export async function PlayDropSound(): Promise<void> {}

export async function DropBarItems(): Promise<DropBarItem[]> {
  return dropBarItems.slice();
}

function labelForPayload(payload: Payload): string {
  if (payload.kind === "files") return payload.paths?.[0]?.split("/").pop() ?? "Files";
  if (payload.kind === "url") return payload.text ?? "Link";
  return payload.text?.slice(0, 40) ?? "Text";
}

export async function DropBarAdd(payload: Payload): Promise<DropBarItem> {
  const item: DropBarItem = {
    id: genId("item"),
    kind: payload.kind,
    paths: payload.paths,
    text: payload.text,
    label: labelForPayload(payload),
    locked: false,
    addedAt: new Date().toISOString(),
  };
  dropBarItems = [item, ...dropBarItems];
  emitDropBar();
  return item;
}

export async function DropBarRemove(id: string): Promise<void> {
  dropBarItems = dropBarItems.filter((i) => i.id !== id);
  emitDropBar();
}

export async function DropBarClear(): Promise<void> {
  dropBarItems = [];
  emitDropBar();
}

export async function DropBarConsume(id: string): Promise<void> {
  const item = dropBarItems.find((i) => i.id === id);
  if (!item || item.locked || settings.dropBarKeepsItems) return;
  dropBarItems = dropBarItems.filter((i) => i.id !== id);
  emitDropBar();
}

export async function DropBarSetLocked(id: string, locked: boolean): Promise<void> {
  const item = dropBarItems.find((i) => i.id === id);
  if (item) item.locked = locked;
  emitDropBar();
}

export async function DropBarRename(id: string, name: string): Promise<void> {
  const item = dropBarItems.find((i) => i.id === id);
  if (item)
    item.label = name || labelForPayload({ kind: item.kind, paths: item.paths, text: item.text });
  emitDropBar();
}

export async function DropBarSeparate(id: string): Promise<void> {
  const item = dropBarItems.find((i) => i.id === id);
  if (!item || !item.paths || item.paths.length < 2) return;
  const separated = item.paths.map((p) => ({
    id: genId("item"),
    kind: item.kind,
    paths: [p],
    label: p.split("/").pop() ?? p,
    locked: false,
    addedAt: new Date().toISOString(),
  }));
  const idx = dropBarItems.findIndex((i) => i.id === id);
  dropBarItems = [...dropBarItems.slice(0, idx), ...separated, ...dropBarItems.slice(idx + 1)];
  emitDropBar();
}

export async function DropBarCombineAll(): Promise<void> {
  const fileItems = dropBarItems.filter((i) => i.kind === "files");
  if (fileItems.length < 2) return;
  const rest = dropBarItems.filter((i) => i.kind !== "files");
  const combined: DropBarItem = {
    id: genId("item"),
    kind: "files",
    paths: fileItems.flatMap((i) => i.paths ?? []),
    label: `${fileItems.length} items`,
    locked: false,
    addedAt: new Date().toISOString(),
  };
  dropBarItems = [combined, ...rest];
  emitDropBar();
}

export async function DropBarCopyToClipboard(_id: string): Promise<void> {}

export async function DropBarReveal(_id: string): Promise<void> {}

export async function DropBarPaste(): Promise<void> {}

export async function SetDropBarPopOut(popped: boolean): Promise<void> {
  EventsEmit("dropbar:popout", popped);
}

export async function QuickLook(_paths: string[]): Promise<void> {}

export async function AnswerInputRequest(
  _id: string,
  _value: string,
  _ok: boolean,
): Promise<void> {}

export async function ListAddons(): Promise<AddonInfo[]> {
  return addons.slice();
}

export async function InstallAddon(name: string): Promise<void> {
  const addon = addons.find((a) => a.name === name);
  if (addon) addon.installed = true;
}

export async function CLIInstalled(): Promise<boolean> {
  return false;
}

export async function InstallCLI(): Promise<void> {}

export async function CheckForUpdates(): Promise<UpdateInfo> {
  return {
    version: "0.1.0",
    latest: "0.1.0",
    available: false,
    url: "https://github.com/GuilhermeVozniak/drag-zone/releases",
    downloadUrl: "",
    publishedAt: "",
  };
}

export async function GetVersion(): Promise<string> {
  return "0.1.0-e2e";
}

export async function ChooseFolder(): Promise<string> {
  return "/Users/demo/Chosen Folder";
}

export async function ChooseApplication(): Promise<string> {
  return "/Applications/TextEdit.app";
}

export async function StartDragOut(_itemId: string): Promise<void> {}

export async function FileIcon(_path: string): Promise<string> {
  return "";
}

export async function RecentShares(): Promise<Share[]> {
  return shares.slice();
}

export async function ClearRecentShares(): Promise<void> {
  shares = [];
  emitShares();
}

export async function OpenURL(_url: string): Promise<void> {}

export async function HideWindow(): Promise<void> {}

export async function ShowWindow(): Promise<void> {}

export async function Quit(): Promise<void> {}

export async function About(): Promise<void> {}
