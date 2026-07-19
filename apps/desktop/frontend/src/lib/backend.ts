// Typed facade over the generated Wails bindings and runtime events.
import * as App from "../../wailsjs/go/main/App";
import type { config, dropbar, main, model } from "../../wailsjs/go/models";
import { EventsOn } from "../../wailsjs/runtime/runtime";

/** UI scale derived from the grid-size setting (mirrors config.Settings.Scale). */
export function uiScale(s: Settings | null): number {
  const pct = Math.min(100, Math.max(0, s?.gridSize ?? 33));
  return 0.8 + (pct / 100) * 0.6;
}

export type Settings = config.Settings;
export type AddonInfo = main.AddonInfo;
export type UpdateInfo = main.UpdateInfo;
export type Share = main.Share;
export type Target = model.Target;
export type ActionSpec = model.ActionSpec;
export type OptionField = model.OptionField;
export type TaskState = model.TaskState;
export type DropBarItem = dropbar.Item;

export type PayloadKind = "files" | "text" | "url";

export interface Payload {
  kind: PayloadKind;
  paths?: string[];
  text?: string;
}

export const backend = {
  settings: {
    get: App.GetSettings,
    set: (s: Settings) => App.SetSettings(s),
    /** Enter settings mode: the shared window becomes a regular titled app
     * window (Dock icon visible) hosting the settings UI on the given tab. */
    open: (tab?: string) => App.OpenSettings(tab ?? "general"),
    /** Leave settings mode and return the window to the popover grid. */
    close: () => App.CloseSettings(),
  },
  actions: {
    specs: App.ActionSpecs,
    installBundle: App.InstallBundle,
    openFolder: App.OpenActionsFolder,
    develop: App.DevelopAction,
  },
  grid: {
    list: App.Targets,
    add: App.AddTarget,
    addFromPaths: App.AddTargetsFromPaths,
    update: (t: Target) => App.UpdateTarget(t),
    duplicate: App.DuplicateTarget,
    remove: App.RemoveTarget,
    move: App.MoveTarget,
  },
  drop: (targetId: string, payload: Payload) =>
    App.DropOnTarget(targetId, payload as model.Payload),
  click: App.ClickTarget,
  tasks: {
    list: App.Tasks,
    dismiss: App.DismissTask,
    cancel: App.CancelTask,
  },
  shares: {
    list: App.RecentShares,
    clear: App.ClearRecentShares,
    open: App.OpenURL,
  },
  playDropSound: App.PlayDropSound,
  dropBar: {
    list: App.DropBarItems,
    add: (payload: Payload) => App.DropBarAdd(payload as model.Payload),
    remove: App.DropBarRemove,
    clear: App.DropBarClear,
    consume: App.DropBarConsume,
    setLocked: App.DropBarSetLocked,
    rename: App.DropBarRename,
    setPopOut: App.SetDropBarPopOut,
    separate: App.DropBarSeparate,
    combineAll: App.DropBarCombineAll,
    combine: App.DropBarCombine,
    copyToClipboard: App.DropBarCopyToClipboard,
    reveal: App.DropBarReveal,
    paste: App.DropBarPaste,
  },
  quickLook: App.QuickLook,
  answerInput: App.AnswerInputRequest,
  addons: {
    list: App.ListAddons,
    install: App.InstallAddon,
  },
  cli: {
    installed: App.CLIInstalled,
    install: App.InstallCLI,
  },
  updates: {
    check: App.CheckForUpdates,
    version: App.GetVersion,
  },
  dialogs: {
    chooseFolder: App.ChooseFolder,
    chooseApplication: App.ChooseApplication,
  },
  dragOut: App.StartDragOut,
  fileIcon: App.FileIcon,
  openURL: App.OpenURL,
  openPath: App.OpenPath,
  window: {
    hide: App.HideWindow,
    quit: App.Quit,
    about: App.About,
    resize: App.ResizeWindow,
  },
};

// Event subscriptions; each returns an unsubscribe function.
export const events = {
  onGridChanged: (fn: (targets: Target[]) => void) => EventsOn("grid:changed", fn),
  onTasksChanged: (fn: (tasks: TaskState[]) => void) => EventsOn("tasks:changed", fn),
  onDropBarChanged: (fn: (items: DropBarItem[]) => void) => EventsOn("dropbar:changed", fn),
  onOpenSettings: (fn: (tab: string) => void) => EventsOn("settings:open", fn),
  onCloseSettings: (fn: () => void) => EventsOn("settings:close", fn),
  onSpecsChanged: (fn: (specs: ActionSpec[]) => void) => EventsOn("specs:changed", fn),
  onDropBarPopOut: (fn: (popped: boolean) => void) => EventsOn("dropbar:popout", fn),
  onInputRequest: (
    fn: (req: { id: string; title: string; prompt: string; choices?: string[] }) => void,
  ) => EventsOn("input:request", fn),
  onWindowVisibility: (fn: (visible: boolean) => void) => EventsOn("window:visibility", fn),
  onWindowBeak: (fn: (x: number) => void) => EventsOn("window:beak", fn),
  onSharesChanged: (fn: (shares: Share[]) => void) => EventsOn("shares:changed", fn),
  /** Whether a native (Finder) file drag is currently over the open grid
   * window; drives the drop-target overlay. Emitted by the native drag
   * monitor in bridge_darwin.m, not HTML5 dragenter/dragover — those don't
   * reliably fire for native file drags in a Wails WKWebView. */
  onDragActive: (fn: (active: boolean) => void) => EventsOn("drag:active", fn),
  /** A Drop Bar item's native drag-out session finished, whatever the
   * outcome (dropped on a sibling tile, dropped outside the window, or
   * cancelled). useNativeFileDrop clears its in-flight drag-source tracker
   * on this signal so a stale id can never leak into a later, unrelated
   * drop. */
  onDropBarDragEnded: (fn: () => void) => EventsOn("dropbar:dragended", fn),
};
