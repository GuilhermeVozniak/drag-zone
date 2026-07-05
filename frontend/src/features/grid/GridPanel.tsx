import { useEffect, useState } from "react"
import { backend, type Target } from "@/lib/backend"
import { initNativeFileDrop } from "@/lib/dnd"
import {
  useActionSpecs,
  useDropBar,
  useSettings,
  useTargets,
  useTasks,
} from "@/hooks/useBackend"
import { PanelTopOpen, Plus, Settings as SettingsIcon } from "lucide-react"
import { DropBar } from "@/features/dropbar/DropBar"
import { TaskList } from "@/features/tasks/TaskList"
import { AddTargetDialog } from "./AddTargetDialog"
import { TargetTile } from "./TargetTile"

const FOLDER_APP_ACTIONS = new Set(["folder", "open-app"])

export function GridPanel({ onOpenSettings }: { onOpenSettings: () => void }) {
  const targets = useTargets()
  const tasks = useTasks()
  const dropBarItems = useDropBar()
  const specs = useActionSpecs()
  const [settings] = useSettings()

  const [addOpen, setAddOpen] = useState(false)
  const [editing, setEditing] = useState<Target | null>(null)

  // Native file drops: route to the tile / drop bar under the cursor.
  useEffect(() => {
    initNativeFileDrop({
      onFiles(dropId, paths) {
        if (!dropId || paths.length === 0) return
        if (dropId === "dropbar") {
          backend.dropBar.add({ kind: "files", paths })
        } else {
          backend.drop(dropId, { kind: "files", paths })
        }
      },
    })
  }, [])

  // Single-key shortcuts launch targets while the grid is open.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey) return
      const el = e.target as HTMLElement
      if (el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.isContentEditable) return
      const key = e.key.length === 1 ? e.key.toUpperCase() : ""
      if (!key) return
      const match = targets.find((t) => t.shortcut?.toUpperCase() === key)
      if (match) {
        e.preventDefault()
        backend.click(match.id)
      }
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [targets])

  const specFor = (t: Target) => specs.find((s) => s.id === t.actionId)
  const folderApps = targets.filter((t) => FOLDER_APP_ACTIONS.has(t.actionId))
  const actionTargets = targets.filter((t) => !FOLDER_APP_ACTIONS.has(t.actionId))

  const dropBarItemOnTarget = async (targetId: string, itemId: string) => {
    const item = dropBarItems.find((i) => i.id === itemId)
    if (!item) return
    await backend.drop(targetId, {
      kind: item.kind as "files" | "text" | "url",
      paths: item.paths,
      text: item.text,
    })
    await backend.dropBar.consume(itemId) // leaves the bar unless locked
  }

  const colsClass =
    { 3: "grid-cols-3", 4: "grid-cols-4", 5: "grid-cols-5", 6: "grid-cols-6" }[
      settings?.gridColumns ?? 4
    ] ?? "grid-cols-4"

  const renderTiles = (list: Target[]) => (
    <div className={`grid ${colsClass} justify-items-center gap-y-1 px-3`}>
      {list.map((t) => (
        <TargetTile
          key={t.id}
          target={t}
          spec={specFor(t)}
          onClick={() => backend.click(t.id)}
          onEdit={() => {
            setEditing(t)
            setAddOpen(true)
          }}
          onRemove={() => backend.grid.remove(t.id)}
          onDropBarItemDrop={(itemId) => dropBarItemOnTarget(t.id, itemId)}
          onTextDrop={(text, isUrl) =>
            backend.drop(t.id, { kind: isUrl ? "url" : "text", text })
          }
          onReorder={(draggedId) => backend.grid.move(draggedId, t.position)}
        />
      ))}
    </div>
  )

  return (
    <div className="flex h-screen flex-col overflow-hidden rounded-2xl border border-white/10 bg-neutral-900/95 shadow-2xl">
      <header
        className="flex items-center justify-between px-4 py-2.5"
        style={{ "--wails-draggable": "drag" } as React.CSSProperties}
      >
        <button
          onClick={() => {
            setEditing(null)
            setAddOpen(true)
          }}
          className="flex size-6 items-center justify-center rounded-full bg-white/10 hover:bg-white/20"
          title="Add to Grid"
        >
          <Plus className="size-3.5 text-white" />
        </button>
        <span className="text-xs font-semibold tracking-wide text-neutral-400">
          DragZone
        </span>
        <div className="flex items-center gap-1">
          <button
            onClick={() => backend.dropBar.setPopOut(true)}
            className="flex size-6 items-center justify-center rounded-full hover:bg-white/10"
            title="Pop out Drop Bar"
          >
            <PanelTopOpen className="size-3.5 text-neutral-400" />
          </button>
          <button
            onClick={onOpenSettings}
            className="flex size-6 items-center justify-center rounded-full hover:bg-white/10"
            title="Settings"
          >
            <SettingsIcon className="size-3.5 text-neutral-400" />
          </button>
        </div>
      </header>

      <DropBar
        items={dropBarItems}
        onRemove={(id) => backend.dropBar.remove(id)}
        onClear={() => backend.dropBar.clear()}
      />

      <div className="mt-2 flex-1 overflow-y-auto pb-2">
        {folderApps.length > 0 && (
          <>
            <SectionLabel>Folders & Apps</SectionLabel>
            {renderTiles(folderApps)}
          </>
        )}
        {actionTargets.length > 0 && (
          <>
            <SectionLabel>Actions</SectionLabel>
            {renderTiles(actionTargets)}
          </>
        )}
      </div>

      <div className="pb-3">
        <TaskList tasks={tasks} />
      </div>

      <AddTargetDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        specs={specs}
        editing={editing}
      />
    </div>
  )
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <p className="px-4 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-wider text-neutral-500">
      {children}
    </p>
  )
}
