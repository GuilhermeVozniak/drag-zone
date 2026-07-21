import {
  ChevronsDown,
  ChevronsUp,
  CircleHelp,
  Copy,
  Download,
  FolderCog,
  Info,
  Plus,
  Power,
  Settings as SettingsIcon,
  TerminalSquare,
  Wrench,
} from "lucide-react";
import { useEffect, useState } from "react";
import { ActionTileIcon } from "@/components/ActionIcon";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { TopSection } from "@/features/dropbar/TopSection";
import { ConsoleDrawer } from "@/features/tasks/ConsoleDrawer";
import { TaskList } from "@/features/tasks/TaskList";
import {
  useActionSpecs,
  useDragActive,
  useDropBar,
  useSettings,
  useTargets,
  useTasks,
} from "@/hooks/useBackend";
import { useNativeFileDrop } from "@/hooks/useNativeFileDrop";
import { useTargetShortcuts } from "@/hooks/useTargetShortcuts";
import { backend, events, type Target } from "@/lib/backend";
import { reportError } from "@/lib/report";
import { gridInputBlocked } from "@/lib/uistate";
import { AddTargetDialog } from "./AddTargetDialog";
import { clickBehavior } from "./clickBehavior";
import { DropTargetOverlay } from "./DropTargetOverlay";
import { RecentSharesPill } from "./RecentSharesPill";
import { TargetTile } from "./TargetTile";

const FOLDER_APP_ACTIONS = new Set(["folder", "open-app"]);

export function GridPanel({ onOpenSettings }: { onOpenSettings: (tab?: string) => void }) {
  const targets = useTargets();
  const tasks = useTasks();
  const dropBarItems = useDropBar();
  const specs = useActionSpecs();
  const [settings] = useSettings();
  const dragActive = useDragActive();
  // Dropzone's "Show drag target overlay when dragging items" setting;
  // defaults to on when settings haven't loaded yet.
  const showDragOverlay = dragActive && (settings?.dragOverlay ?? true);

  const [addOpen, setAddOpen] = useState(false);
  const [editing, setEditing] = useState<Target | null>(null);
  const [addingSpecId, setAddingSpecId] = useState<string | null>(null);
  const [topCollapsed, setTopCollapsed] = useState(false);
  const [optionHeld, setOptionHeld] = useState(false);
  const [consoleOpen, setConsoleOpen] = useState(false);

  // A failed script run auto-opens the debug console, like Dropzone.
  useEffect(() => events.onConsoleError(() => setConsoleOpen(true)), []);

  // Option puts the grid in delete mode (X badges on tiles), like Dropzone.
  useEffect(() => {
    const down = (e: KeyboardEvent) => e.altKey && setOptionHeld(true);
    const up = (e: KeyboardEvent) => !e.altKey && setOptionHeld(false);
    const blur = () => setOptionHeld(false);
    window.addEventListener("keydown", down);
    window.addEventListener("keyup", up);
    window.addEventListener("blur", blur);
    return () => {
      window.removeEventListener("keydown", down);
      window.removeEventListener("keyup", up);
      window.removeEventListener("blur", blur);
    };
  }, []);

  // Cmd-V stashes the clipboard into the Drop Bar.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement;
      if (el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.isContentEditable) return;
      if (gridInputBlocked()) return;
      if (e.metaKey && e.key.toLowerCase() === "v") {
        e.preventDefault();
        backend.dropBar.paste().catch((err) => reportError("Couldn't paste", err));
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // "+" menu selection: actions without options are added straight to the
  // grid (Dropzone's SkipConfig behavior); the rest open the config dialog.
  const chooseSpec = (specId: string) => {
    const spec = specs.find((s) => s.id === specId);
    if (!spec) return;
    if (!spec.options || spec.options.length === 0) {
      backend.grid.add(spec.id, spec.name, {}).catch((err) => reportError("Couldn't add", err));
      return;
    }
    setEditing(null);
    setAddingSpecId(specId);
    setAddOpen(true);
  };

  const openConfig = (t: Target) => {
    setEditing(t);
    setAddOpen(true);
  };

  // Clicking a tile runs the action's click handler, opens its config, or does
  // nothing, per clickBehavior() (see its tests for the exact rules).
  const handleClick = (t: Target) => {
    switch (clickBehavior(specFor(t))) {
      case "config":
        openConfig(t);
        break;
      case "run":
        backend.click(t.id).catch((err) => reportError(`Couldn't run ${t.label}`, err));
        break;
      // "none": drag-only action with nothing to configure.
    }
  };

  useNativeFileDrop(dropBarItems);
  useTargetShortcuts(targets);

  const specFor = (t: Target) => specs.find((s) => s.id === t.actionId);
  const folderApps = targets.filter((t) => FOLDER_APP_ACTIONS.has(t.actionId));
  const actionTargets = targets.filter((t) => !FOLDER_APP_ACTIONS.has(t.actionId));

  const dropBarItemOnTarget = async (targetId: string, itemId: string) => {
    const item = dropBarItems.find((i) => i.id === itemId);
    if (!item) return;
    try {
      await backend.drop(targetId, {
        kind: item.kind as "files" | "text" | "url",
        paths: item.paths,
        text: item.text,
      });
      await backend.dropBar.consume(itemId); // leaves the bar unless locked
    } catch (err) {
      reportError("Drop failed", err);
    }
  };

  const cols = settings?.gridColumns ?? 4;
  const colsClass =
    { 3: "grid-cols-3", 4: "grid-cols-4", 5: "grid-cols-5", 6: "grid-cols-6" }[cols] ??
    "grid-cols-4";

  // Tile/icon size is column-aware so the fixed-width tiles always fit their
  // grid track inside the 360px window: at 3-4 columns icons render at the
  // Dropzone spec's ~64px large-icon default; denser 5-6 column layouts shrink
  // to fit (track width ~344/cols). Driven by a prop, not CSS %, because each
  // tile's grid item is a Radix ContextMenuTrigger wrapper that breaks
  // percentage-width resolution against the track.
  const tileSize =
    cols >= 6 ? { w: 54, icon: 44 } : cols === 5 ? { w: 66, icon: 52 } : { w: 80, icon: 64 };

  const renderTiles = (list: Target[]) => (
    <div className={`grid ${colsClass} justify-items-center gap-y-0.5 px-2`}>
      {list.map((t) => (
        <TargetTile
          key={t.id}
          target={t}
          spec={specFor(t)}
          tilePx={tileSize.w}
          iconPx={tileSize.icon}
          showKeyOverlay={settings?.showKeyOverlays ?? true}
          optionHeld={optionHeld}
          onClick={() => handleClick(t)}
          onEdit={() => openConfig(t)}
          onDuplicate={() =>
            backend.grid.duplicate(t.id).catch((err) => reportError("Couldn't duplicate", err))
          }
          onRemove={() =>
            backend.grid.remove(t.id).catch((err) => reportError("Couldn't remove", err))
          }
          onCopyEditScript={() =>
            backend.actions
              .copyEditScript(t.id)
              .catch((err) => reportError("Couldn't copy script", err))
          }
          onDropBarItemDrop={(itemId) => dropBarItemOnTarget(t.id, itemId)}
          onTextDrop={(text, isUrl) =>
            backend
              .drop(t.id, { kind: isUrl ? "url" : "text", text })
              .catch((err) => reportError("Drop failed", err))
          }
          onReorder={(draggedId) =>
            backend.grid
              .move(draggedId, t.position)
              .catch((err) => reportError("Couldn't move", err))
          }
        />
      ))}
    </div>
  );

  return (
    <div className="relative flex min-h-0 flex-1 flex-col overflow-hidden">
      <DropTargetOverlay active={showDragOverlay} />
      <header
        className="flex items-center justify-between px-3 py-2"
        style={{ "--wails-draggable": "drag" } as React.CSSProperties}
      >
        <div className="flex items-center gap-1">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className="flex size-7 items-center justify-center rounded-md hover:bg-white/10"
                title="Add to Grid (⌥-click to Develop Action)"
                onPointerDown={(e) => {
                  // Dropzone's Option+plus shortcut: skip the catalogue menu
                  // and go straight to the Develop Action workflow.
                  if (e.altKey) {
                    e.preventDefault();
                    e.stopPropagation();
                    onOpenSettings("general");
                  }
                }}
              >
                <Plus className="size-4 text-neutral-200" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="max-h-[380px] overflow-y-auto">
              {specs.map((s) => (
                <DropdownMenuItem key={s.id} onClick={() => chooseSpec(s.id)}>
                  <ActionTileIcon actionId={s.id} icon={s.icon} className="size-5" />
                  {s.name}
                </DropdownMenuItem>
              ))}
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => onOpenSettings("addons")}>
                <Download className="size-3.5" /> Get More Actions…
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onOpenSettings()}>
                <Wrench className="size-3.5" /> Develop Action…
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <div className="mx-1.5 h-4 w-px bg-white/15" />
          <HeaderButton
            title={topCollapsed ? "Show Drop Bar" : "Hide Drop Bar"}
            onClick={() => setTopCollapsed((c) => !c)}
          >
            {topCollapsed ? (
              <ChevronsDown className="size-4 text-neutral-200" />
            ) : (
              <ChevronsUp className="size-4 text-neutral-200" />
            )}
          </HeaderButton>
        </div>
        <RecentSharesPill />
        <div className="flex items-center">
          <HeaderButton title="Pop out Drop Bar" onClick={() => backend.dropBar.setPopOut(true)}>
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
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onOpenSettings()}>
                <SettingsIcon className="size-3.5" /> Settings…
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => backend.actions.openFolder()}>
                <FolderCog className="size-3.5" /> Open Add-on Actions Folder
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setConsoleOpen((o) => !o)}>
                <TerminalSquare className="size-3.5" /> Debug Console
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => backend.openURL("https://github.com/GuilhermeVozniak/drag-zone")}
              >
                <CircleHelp className="size-3.5" /> Help
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => backend.window.about()}>
                <Info className="size-3.5" /> About DragZone
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" onClick={() => backend.window.quit()}>
                <Power className="size-3.5" /> Quit DragZone
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      {!topCollapsed && (
        <TopSection
          items={dropBarItems}
          onAddClick={() => {
            setEditing(null);
            setAddingSpecId(null);
            setAddOpen(true);
          }}
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

      {tasks.length > 0 && (
        <Section label="TASK PROGRESS">
          <TaskList
            tasks={tasks}
            specFor={(id) => specs.find((s) => s.id === id)}
            targets={targets}
          />
        </Section>
      )}

      {consoleOpen && <ConsoleDrawer onClose={() => setConsoleOpen(false)} />}

      <AddTargetDialog
        open={addOpen}
        onOpenChange={(open) => {
          setAddOpen(open);
          if (!open) setAddingSpecId(null);
        }}
        specs={specs}
        editing={editing}
        initialSpecId={addingSpecId}
      />
    </div>
  );
}

function HeaderButton({
  title,
  onClick,
  children,
}: {
  title: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className="flex size-7 items-center justify-center rounded-md hover:bg-white/10"
      title={title}
    >
      {children}
    </button>
  );
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="border-t border-white/10">
      <p className="px-4 pb-1 pt-1.5 text-[10px] font-semibold tracking-wider text-neutral-500">
        {label}
      </p>
      {children}
    </div>
  );
}
