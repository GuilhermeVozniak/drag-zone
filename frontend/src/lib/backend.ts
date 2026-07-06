// Typed facade over the generated Wails bindings and runtime events.
import * as App from "../../wailsjs/go/main/App"
import { EventsOn } from "../../wailsjs/runtime/runtime"
import type { config, dropbar, main, model } from "../../wailsjs/go/models"

/** UI scale derived from the grid-size setting (mirrors config.Settings.Scale). */
export function uiScale(s: Settings | null): number {
  const pct = Math.min(100, Math.max(0, s?.gridSize ?? 33))
  return 0.8 + (pct / 100) * 0.6
}

export type Settings = config.Settings
export type AddonInfo = main.AddonInfo
export type UpdateInfo = main.UpdateInfo
export type Share = main.Share
export type Target = model.Target
export type ActionSpec = model.ActionSpec
export type OptionField = model.OptionField
export type TaskState = model.TaskState
export type DropBarItem = dropbar.Item

export type PayloadKind = "files" | "text" | "url"

export interface Payload {
  kind: PayloadKind
  paths?: string[]
  text?: string
}

export const backend = {
  settings: {
    get: App.GetSettings,
    set: (s: Settings) => App.SetSettings(s),
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
  window: {
    hide: App.HideWindow,
    quit: App.Quit,
  },
}

// Event subscriptions; each returns an unsubscribe function.
export const events = {
  onGridChanged: (fn: (targets: Target[]) => void) => EventsOn("grid:changed", fn),
  onTasksChanged: (fn: (tasks: TaskState[]) => void) => EventsOn("tasks:changed", fn),
  onDropBarChanged: (fn: (items: DropBarItem[]) => void) => EventsOn("dropbar:changed", fn),
  onOpenSettings: (fn: () => void) => EventsOn("settings:open", fn),
  onSpecsChanged: (fn: (specs: ActionSpec[]) => void) =>
    EventsOn("specs:changed", fn),
  onDropBarPopOut: (fn: (popped: boolean) => void) =>
    EventsOn("dropbar:popout", fn),
  onInputRequest: (
    fn: (req: { id: string; title: string; prompt: string }) => void
  ) => EventsOn("input:request", fn),
  onWindowVisibility: (fn: (visible: boolean) => void) =>
    EventsOn("window:visibility", fn),
  onWindowBeak: (fn: (x: number) => void) => EventsOn("window:beak", fn),
  onSharesChanged: (fn: (shares: Share[]) => void) =>
    EventsOn("shares:changed", fn),
}
