// Typed facade over the generated Wails bindings and runtime events.
import * as App from "../../wailsjs/go/main/App"
import { EventsOn } from "../../wailsjs/runtime/runtime"
import type { config, dropbar, model } from "../../wailsjs/go/models"

export type Settings = config.Settings
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
  },
  dropBar: {
    list: App.DropBarItems,
    add: (payload: Payload) => App.DropBarAdd(payload as model.Payload),
    remove: App.DropBarRemove,
    clear: App.DropBarClear,
    consume: App.DropBarConsume,
    setLocked: App.DropBarSetLocked,
    rename: App.DropBarRename,
    setPopOut: App.SetDropBarPopOut,
  },
  quickLook: App.QuickLook,
  answerInput: App.AnswerInputRequest,
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
}
