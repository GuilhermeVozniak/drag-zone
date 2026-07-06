import { useEffect, useState, useSyncExternalStore } from "react"
import {
  backend,
  events,
  type ActionSpec,
  type DropBarItem,
  type Settings,
  type Target,
  type TaskState,
} from "@/lib/backend"

// arr coerces a possibly-null binding result into an array. Go marshals an
// empty slice as null unless the backend guards against it; this keeps the
// UI resilient regardless.
const arr = <T,>(v: T[] | null | undefined): T[] => v ?? []

/** Live grid targets, updated via backend events. */
export function useTargets(): Target[] {
  const [targets, setTargets] = useState<Target[]>([])
  useEffect(() => {
    backend.grid.list().then((v) => setTargets(arr(v)))
    return events.onGridChanged((v) => setTargets(arr(v)))
  }, [])
  return targets
}

/** Live task list, most recent first. */
export function useTasks(): TaskState[] {
  const [tasks, setTasks] = useState<TaskState[]>([])
  useEffect(() => {
    backend.tasks.list().then((v) => setTasks(arr(v)))
    return events.onTasksChanged((v) => setTasks(arr(v)))
  }, [])
  return tasks
}

/** Live drop bar items. */
export function useDropBar(): DropBarItem[] {
  const [items, setItems] = useState<DropBarItem[]>([])
  useEffect(() => {
    backend.dropBar.list().then((v) => setItems(arr(v)))
    return events.onDropBarChanged((v) => setItems(arr(v)))
  }, [])
  return items
}

/** Installable action specs, refreshed when bundles are installed. */
export function useActionSpecs(): ActionSpec[] {
  const [specs, setSpecs] = useState<ActionSpec[]>([])
  useEffect(() => {
    backend.actions.specs().then((v) => setSpecs(arr(v)))
    return events.onSpecsChanged((v) => setSpecs(arr(v)))
  }, [])
  return specs
}

// Settings live in a tiny module store so every consumer re-renders when any
// component saves a change.
let settingsState: Settings | null = null
let settingsLoading = false
const settingsListeners = new Set<() => void>()

function publishSettings(s: Settings) {
  settingsState = s
  settingsListeners.forEach((l) => l())
}

/** Settings with an updater that persists and notifies all consumers. */
export function useSettings(): [Settings | null, (s: Settings) => Promise<void>] {
  const settings = useSyncExternalStore(
    (cb) => {
      settingsListeners.add(cb)
      return () => settingsListeners.delete(cb)
    },
    () => settingsState
  )
  useEffect(() => {
    if (settingsState === null && !settingsLoading) {
      settingsLoading = true
      backend.settings.get().then(publishSettings)
    }
  }, [])
  const update = async (s: Settings) => {
    publishSettings(s)
    await backend.settings.set(s)
  }
  return [settings, update]
}
