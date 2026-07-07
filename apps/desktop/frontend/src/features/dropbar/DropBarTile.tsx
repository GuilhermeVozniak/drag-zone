import { File, Files, Link, Lock, Type, X } from "lucide-react";
import { useRef, useState } from "react";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { useFileIcon } from "@/hooks/useFileIcon";
import { backend, type DropBarItem } from "@/lib/backend";
import { DROPBAR_MIME } from "@/lib/dnd";
import { RenameItemDialog } from "./RenameItemDialog";

function itemIcon(item: DropBarItem) {
  if (item.kind === "files") return (item.paths?.length ?? 0) > 1 ? Files : File;
  if (item.kind === "url") return Link;
  return Type;
}

/**
 * Fanned, photo-bordered thumbnails for a stack, like Dropzone's stacks.
 * Hovering the tile spreads the fan and lifts + highlights the front image, so
 * it's clear which stack you're about to grab. paths[0] is drawn on top.
 */
function StackFan({ paths }: { paths: string[] }) {
  const first = useFileIcon(paths[0]);
  const second = useFileIcon(paths[1]);
  const third = useFileIcon(paths[2]);
  const layers = [
    {
      icon: third,
      base: "-rotate-[10deg] -translate-x-2",
      hover: "group-hover:-rotate-[18deg] group-hover:-translate-x-4 group-hover:-translate-y-0.5",
    },
    {
      icon: second,
      base: "rotate-[8deg] translate-x-2",
      hover: "group-hover:rotate-[18deg] group-hover:translate-x-4 group-hover:-translate-y-0.5",
    },
    {
      icon: first,
      base: "rotate-0",
      hover:
        "group-hover:-translate-y-1 group-hover:scale-[1.12] group-hover:ring-2 group-hover:ring-sky-400/80",
    },
  ].filter((l) => l.icon);
  if (layers.length === 0) {
    return <Files className="size-7 text-neutral-300" strokeWidth={1.5} />;
  }
  return (
    <div className="relative size-[48px]">
      {layers.map((l, i) => (
        <img
          key={i}
          src={`data:image/png;base64,${l.icon}`}
          alt=""
          draggable={false}
          className={`absolute inset-0 m-auto max-h-[40px] max-w-[40px] rounded-[3px] border-2 border-white bg-white object-contain shadow-sm transition-transform duration-150 ease-out ${l.base} ${l.hover}`}
        />
      ))}
    </div>
  );
}

interface DropBarTileProps {
  item: DropBarItem;
  onRemove: (id: string) => void;
}

/**
 * One stashed item (or stack) in the Drop Bar. File items start a native
 * drag session on drag-out so they can land in Finder and other apps;
 * text/URL items use HTML5 drag for in-window drops onto grid tiles.
 */
export function DropBarTile({ item, onRemove }: DropBarTileProps) {
  const Icon = itemIcon(item);
  const count = item.paths?.length ?? 0;
  const nativeIcon = useFileIcon(item.paths?.[0]);
  const dragStart = useRef<{ x: number; y: number } | null>(null);
  const isFiles = item.kind === "files";
  const [renaming, setRenaming] = useState<string | null>(null);

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>
        <div
          draggable={!isFiles}
          onDragStart={(e) => {
            e.dataTransfer.setData(DROPBAR_MIME, item.id);
            e.dataTransfer.effectAllowed = "copyMove";
          }}
          onMouseDown={(e) => {
            if (isFiles && e.button === 0) {
              dragStart.current = { x: e.clientX, y: e.clientY };
            }
          }}
          onMouseMove={(e) => {
            const start = dragStart.current;
            if (!start) return;
            if (Math.hypot(e.clientX - start.x, e.clientY - start.y) > 5) {
              dragStart.current = null;
              backend.dragOut(item.id);
            }
          }}
          onMouseUp={() => {
            dragStart.current = null;
          }}
          onDoubleClick={() => {
            if (isFiles) backend.quickLook(item.paths ?? []);
          }}
          className="group relative flex w-[64px] cursor-grab flex-col items-center gap-1 rounded-lg p-1.5 hover:bg-white/[0.08]"
        >
          <div className="relative flex size-[52px] items-center justify-center">
            {count > 1 ? (
              <StackFan paths={item.paths ?? []} />
            ) : nativeIcon ? (
              <img
                src={`data:image/png;base64,${nativeIcon}`}
                alt=""
                className="max-h-[46px] max-w-[46px] rounded-[3px] object-contain"
                draggable={false}
              />
            ) : (
              <Icon className="size-7 text-neutral-300" strokeWidth={1.5} />
            )}
            {item.locked && (
              <span className="absolute -bottom-0.5 -right-0.5 z-10 rounded-full bg-neutral-700 p-0.5">
                <Lock className="size-2.5 text-amber-400" />
              </span>
            )}
            {count > 1 && (
              <span className="pointer-events-none absolute -right-1.5 -top-1.5 z-20 min-w-[17px] rounded-full bg-sky-500 px-1 text-center text-[9px] font-semibold leading-[17px] text-white shadow-sm ring-1 ring-white/70">
                {count}
              </span>
            )}
          </div>
          <span className="w-full truncate text-center text-[10px] text-neutral-400">
            {item.label}
          </span>
          <button
            onClick={() => onRemove(item.id)}
            className="absolute -left-1 -top-1 hidden rounded-full bg-neutral-700 p-0.5 group-hover:block"
          >
            <X className="size-2.5 text-white" />
          </button>
        </div>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onClick={() => backend.dropBar.setLocked(item.id, !item.locked)}>
          {item.locked ? "Unlock Items" : "Lock Items"}
        </ContextMenuItem>
        {count > 1 && (
          <ContextMenuItem onClick={() => backend.dropBar.separate(item.id)}>
            Separate Items
          </ContextMenuItem>
        )}
        <ContextMenuItem onClick={() => setRenaming(item.label)}>
          {count > 1 ? "Name Stack…" : "Rename…"}
        </ContextMenuItem>
        <ContextMenuSeparator />
        {isFiles && (
          <ContextMenuItem onClick={() => backend.quickLook(item.paths ?? [])}>
            Quick Look
          </ContextMenuItem>
        )}
        {isFiles && (
          <ContextMenuItem onClick={() => backend.dropBar.reveal(item.id)}>
            Show in Finder
          </ContextMenuItem>
        )}
        <ContextMenuSeparator />
        <ContextMenuItem onClick={() => backend.dropBar.copyToClipboard(item.id)}>
          Copy to Clipboard
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem variant="destructive" onClick={() => onRemove(item.id)}>
          Remove
        </ContextMenuItem>
      </ContextMenuContent>
      <RenameItemDialog item={item} value={renaming} onValueChange={setRenaming} />
    </ContextMenu>
  );
}
