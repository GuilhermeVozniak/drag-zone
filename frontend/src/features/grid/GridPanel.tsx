import { useState } from "react"
import { backend, type Target } from "@/lib/backend"
import {
  useActionSpecs,
  useDropBar,
  useSettings,
  useTargets,
  useTasks,
} from "@/hooks/useBackend"
import { useNativeFileDrop } from "@/hooks/useNativeFileDrop"
import { useTargetShortcuts } from "@/hooks/useTargetShortcuts"
import {
  ChevronsUp,
  Copy,
  FolderCog,
  Plus,
  Power,
  Settings as SettingsIcon,
  Wrench,
} from "lucide-react"
import { ActionTileIcon } from "@/components/ActionIcon"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { DropBar } from "@/features/dropbar/DropBar"
import { EmptyTopTiles } from "@/features/dropbar/EmptyTopTiles"
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
  const [addingSpecId, setAddingSpecId] = useState<string | null>(null)

  // "+" menu selection: actions without options are added straight to the
  // grid (Dropzone's SkipConfig behavior); the rest open the config dialog.
  const chooseSpec = (specId: string) => {
    const spec = specs.find((s) => s.id === specId)
    if (!spec) return
    if (!spec.options || spec.options.length === 0) {
      backend.grid.add(spec.id, spec.name, {})
      return
    }
    setEditing(null)
    setAddingSpecId(specId)
    setAddOpen(true)
  }

  useNativeFileDrop()
  useTargetShortcuts(targets)

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
    <div className="flex h-full flex-col overflow-hidden">
      <header
        className="flex items-center justify-between px-3 py-2"
        style={{ "--wails-draggable": "drag" } as React.CSSProperties}
      >
        <div className="flex items-center gap-1">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className="flex size-7 items-center justify-center rounded-md hover:bg-white/10"
                title="Add to Grid"
              >
                <Plus className="size-4 text-neutral-200" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="dark max-h-[380px] overflow-y-auto">
              {specs.map((s) => (
                <DropdownMenuItem key={s.id} onClick={() => chooseSpec(s.id)}>
                  <ActionTileIcon actionId={s.id} icon={s.icon} className="size-5" />
                  {s.name}
                </DropdownMenuItem>
              ))}
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={onOpenSettings}>
                <Wrench className="size-3.5" /> Develop Action…
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <HeaderButton title="Hide grid" onClick={() => backend.window.hide()}>
            <ChevronsUp className="size-4 text-neutral-200" />
          </HeaderButton>
        </div>
        <div className="flex items-center">
          <HeaderButton
            title="Pop out Drop Bar"
            onClick={() => backend.dropBar.setPopOut(true)}
          >
            <Copy className="size-4 text-neutral-200" />
          </HeaderButton>
          <div className="mx-1.5 h-4 w-px bg-white/15" />
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className="flex size-7 items-center justify-center rounded-md hover:bg-white/10"
                title="Settings"
              >
                <SettingsIcon className="size-4 text-neutral-200" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="dark">
              <DropdownMenuItem onClick={onOpenSettings}>
                <SettingsIcon className="size-3.5" /> Settings…
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => backend.actions.openFolder()}>
                <FolderCog className="size-3.5" /> Open Add-on Actions Folder
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                variant="destructive"
                onClick={() => backend.window.quit()}
              >
                <Power className="size-3.5" /> Quit DragZone
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      {dropBarItems.length === 0 ? (
        <EmptyTopTiles
          onAddClick={() => {
            setEditing(null)
            setAddingSpecId(null)
            setAddOpen(true)
          }}
        />
      ) : (
        <DropBar
          items={dropBarItems}
          onRemove={(id) => backend.dropBar.remove(id)}
          onClear={() => backend.dropBar.clear()}
        />
      )}

      <div className="flex-1 overflow-y-auto pb-2">
        {folderApps.length > 0 && (
          <Section label="FOLDERS / APPS">{renderTiles(folderApps)}</Section>
        )}
        {actionTargets.length > 0 && (
          <Section label="ACTIONS">{renderTiles(actionTargets)}</Section>
        )}
      </div>

      <div className="pb-2">
        <TaskList tasks={tasks} />
      </div>

      <AddTargetDialog
        open={addOpen}
        onOpenChange={(open) => {
          setAddOpen(open)
          if (!open) setAddingSpecId(null)
        }}
        specs={specs}
        editing={editing}
        initialSpecId={addingSpecId}
      />
    </div>
  )
}

function HeaderButton({
  title,
  onClick,
  children,
}: {
  title: string
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className="flex size-7 items-center justify-center rounded-md hover:bg-white/10"
      title={title}
    >
      {children}
    </button>
  )
}

function Section({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="border-t border-white/10">
      <p className="px-4 pb-1.5 pt-2 text-[10px] font-semibold tracking-wider text-neutral-500">
        {label}
      </p>
      {children}
    </div>
  )
}
