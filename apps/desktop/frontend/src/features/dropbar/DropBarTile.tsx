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
import { DROPBAR_MIME, setDraggingDropBarItem } from "@/lib/dnd";
import { RenameItemDialog } from "./RenameItemDialog";

function itemIcon(item: DropBarItem) {
  if (item.kind === "files") return (item.paths?.length ?? 0) > 1 ? Files : File;
  if (item.kind === "url") return Link;
  return Type;
}

// A hover combines only over a file tile's center (the outer 30% edges
// reorder/stash instead, mirroring useNativeFileDrop's routing) and never
// for internal text/URL drags, which can't join a stack.
function isCombineHover(e: React.DragEvent, isFiles: boolean) {
  if (!isFiles || e.dataTransfer.types.includes(DROPBAR_MIME)) return false;
  const rect = e.currentTarget.getBoundingClientRect();
  if (rect.width <= 0) return true;
  const rel = (e.clientX - rect.left) / rect.width;
  return rel >= 0.3 && rel <= 0.7;
}

// Stacks fan out at most this many thumbnails; the count badge covers the
// rest.
const STACK_FAN_MAX = 7;

/**
 * Fanned, photo-bordered thumbnails for a stack, like Dropzone's stacks.
 * Hovering the fan spreads it; the thumbnail under the cursor lifts to the
 * front and highlights, so a click opens exactly that file (Quick Look of
 * the whole stack stays on the tile's own click), and so a click must not
 * bubble up into the tile's drag-out or Quick Look handlers. paths[0] is
 * drawn on top when nothing is hovered.
 */
function StackFan({ paths }: { paths: string[] }) {
  const [spread, setSpread] = useState(false);
  const [focused, setFocused] = useState<number | null>(null);
  const shown = paths.slice(0, STACK_FAN_MAX);
  // k is a layer's stack position: 0 is the front item; deeper layers
  // alternate right/left with growing depth (1,1,2,2,3,3).
  const transform = (k: number) => {
    if (k === 0) {
      return spread || focused === k ? "translateY(-4px)" : "";
    }
    const side = k % 2 === 1 ? 1 : -1;
    const depth = Math.ceil(k / 2);
    const open = spread || focused != null;
    const deg = side * depth * (open ? 8 : 3);
    const tx = side * depth * (open ? 9 : 4);
    const ty = open ? -2 : 0;
    return `rotate(${deg}deg) translate(${tx}px, ${ty}px)`;
  };
  return (
    <div
      className="relative size-[60px]"
      onMouseEnter={() => setSpread(true)}
      onMouseLeave={() => {
        setSpread(false);
        setFocused(null);
      }}
    >
      {/* Placeholder while icons load (and fallback if none resolve). */}
      <Files className="absolute inset-0 m-auto size-7 text-neutral-300" strokeWidth={1.5} />
      {/* Back-most layer first so paths[0] ends up last in DOM (on top). */}
      {[...shown.keys()].reverse().map((k) => (
        <StackThumb
          key={shown[k]}
          path={shown[k]}
          transform={`${transform(k)}${focused === k ? " scale(1.12)" : ""}`}
          focused={focused === k}
          onFocus={() => setFocused(k)}
          onBlur={() => setFocused((f) => (f === k ? null : f))}
        />
      ))}
    </div>
  );
}

/** One thumbnail in a stack fan; resolves its own Finder icon. */
function StackThumb({
  path,
  transform,
  focused,
  onFocus,
  onBlur,
}: {
  path: string;
  transform: string;
  focused: boolean;
  onFocus: () => void;
  onBlur: () => void;
}) {
  const icon = useFileIcon(path);
  if (!icon) return null;
  return (
    <img
      src={`data:image/png;base64,${icon}`}
      alt=""
      draggable={false}
      onClick={(e) => {
        e.stopPropagation();
        backend.openPath(path);
      }}
      onMouseEnter={onFocus}
      onMouseLeave={onBlur}
      style={{ transform }}
      className={`pointer-events-auto absolute inset-0 m-auto max-h-[50px] max-w-[50px] cursor-pointer rounded-[3px] border-2 border-white bg-white object-contain shadow-sm transition-transform duration-150 ease-out ${
        focused ? "z-10 ring-2 ring-sky-400/80" : ""
      }`}
    />
  );
}

interface DropBarTileProps {
  item: DropBarItem;
  onRemove: (id: string) => void;
  /** Another item was dropped on this tile's left/right half: reorder. */
  onReorderRequest?: (sourceId: string, targetId: string, after: boolean) => void;
}

/**
 * One stashed item (or stack) in the Drop Bar. File items start a native
 * drag session on drag-out so they can land in Finder and other apps;
 * text/URL items use HTML5 drag for in-window drops onto grid tiles. A
 * single click on a file item (or stack) Quick Looks its contents.
 */
export function DropBarTile({ item, onRemove, onReorderRequest }: DropBarTileProps) {
  const Icon = itemIcon(item);
  const count = item.paths?.length ?? 0;
  const nativeIcon = useFileIcon(item.paths?.[0]);
  const dragStart = useRef<{ x: number; y: number } | null>(null);
  // Set when a press turns into a drag-out, so the click that lands on
  // mouse-up doesn't also fire a Quick Look.
  const didDrag = useRef(false);
  const isFiles = item.kind === "files";
  const [renaming, setRenaming] = useState<string | null>(null);
  // Highlighted while another Drop Bar item's drag-out hovers this tile,
  // signalling that releasing here combines the two into a stack.
  const [combineHover, setCombineHover] = useState(false);

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>
        <div
          data-drop-id={item.id}
          draggable={!isFiles}
          onDragStart={(e) => {
            e.dataTransfer.setData(DROPBAR_MIME, item.id);
            e.dataTransfer.effectAllowed = "copyMove";
          }}
          onMouseDown={(e) => {
            didDrag.current = false;
            if (isFiles && e.button === 0) {
              dragStart.current = { x: e.clientX, y: e.clientY };
            }
          }}
          onMouseMove={(e) => {
            const start = dragStart.current;
            if (!start) return;
            if (Math.hypot(e.clientX - start.x, e.clientY - start.y) > 5) {
              dragStart.current = null;
              didDrag.current = true;
              // Mark this item as the in-flight drag-out source so a drop
              // that lands back on a sibling tile (see useNativeFileDrop)
              // combines the two instead of stashing a duplicate item.
              setDraggingDropBarItem(item.id);
              backend.dragOut(item.id);
            }
          }}
          onMouseUp={() => {
            dragStart.current = null;
          }}
          onClick={() => {
            if (!isFiles) return;
            if (didDrag.current) {
              didDrag.current = false;
              return;
            }
            backend.quickLook(item.paths ?? []);
          }}
          // Best-effort visual hint for the combine drop target: WebKit
          // forwards a native drag hovering the window as ordinary drag
          // events. The actual combine happens through the native file-drop
          // path (see useNativeFileDrop); this only drives the highlight,
          // which mirrors that routing: center of a file tile combines,
          // the outer edges (and text/URL drags) never do.
          onDragOver={(e) => {
            if (isFiles || e.dataTransfer.types.includes(DROPBAR_MIME)) e.preventDefault();
            setCombineHover(isCombineHover(e, isFiles));
          }}
          onDragEnter={(e) => {
            if (isFiles) e.preventDefault();
          }}
          onDragLeave={() => setCombineHover(false)}
          onDrop={(e) => {
            e.preventDefault();
            setCombineHover(false);
            const sourceId = e.dataTransfer.getData(DROPBAR_MIME);
            if (sourceId && sourceId !== item.id && onReorderRequest) {
              const rect = e.currentTarget.getBoundingClientRect();
              const after = (e.clientX - rect.left) / Math.max(rect.width, 1) > 0.5;
              onReorderRequest(sourceId, item.id, after);
            }
          }}
          className={`group relative flex w-[72px] cursor-grab flex-col items-center gap-1 rounded-lg p-1.5 hover:bg-white/[0.08] ${
            combineHover ? "bg-sky-500/20 ring-2 ring-sky-400/80" : ""
          }`}
        >
          <div className="relative flex size-[64px] items-center justify-center">
            {count > 1 ? (
              <StackFan paths={item.paths ?? []} />
            ) : nativeIcon ? (
              <img
                src={`data:image/png;base64,${nativeIcon}`}
                alt=""
                className="max-h-[58px] max-w-[58px] rounded-[3px] object-contain"
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
            onClick={(e) => {
              // Don't let the remove click bubble into the tile's Quick Look.
              e.stopPropagation();
              onRemove(item.id);
            }}
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
