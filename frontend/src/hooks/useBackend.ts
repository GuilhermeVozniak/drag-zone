import { useEffect, useState } from "react"
import {
  backend,
  events,
  type ActionSpec,
  type DropBarItem,
  type Settings,
  type Target,
  type TaskState,
} from "@/lib/backend"

/** Live grid targets, updated via backend events. */
export function useTargets(): Target[] {
  const [targets, setTargets] = useState<Target[]>([])
  useEffect(() => {
    backend.grid.list().then(setTargets)
    return events.onGridChanged(setTargets)
  }, [])
  return targets
}

/** Live task list, most recent first. */
export function useTasks(): TaskState[] {
  const [tasks, setTasks] = useState<TaskState[]>([])
  useEffect(() => {
    backend.tasks.list().then(setTasks)
    return events.onTasksChanged(setTasks)
  }, [])
  return tasks
}

/** Live drop bar items. */
export function useDropBar(): DropBarItem[] {
  const [items, setItems] = useState<DropBarItem[]>([])
  useEffect(() => {
    backend.dropBar.list().then(setItems)
    return events.onDropBarChanged(setItems)
  }, [])
  return items
}

/** Installable action specs, refreshed when bundles are installed. */
export function useActionSpecs(): ActionSpec[] {
  const [specs, setSpecs] = useState<ActionSpec[]>([])
  useEffect(() => {
    backend.actions.specs().then(setSpecs)
    return events.onSpecsChanged(setSpecs)
  }, [])
  return specs
}

/** Settings with an updater that persists. */
export function useSettings(): [Settings | null, (s: Settings) => Promise<void>] {
  const [settings, setSettings] = useState<Settings | null>(null)
  useEffect(() => {
    backend.settings.get().then(setSettings)
  }, [])
  const update = async (s: Settings) => {
    setSettings(s)
    await backend.settings.set(s)
  }
  return [settings, update]
}
