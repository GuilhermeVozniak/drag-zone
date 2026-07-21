import { Plus, X } from "lucide-react";
import { useState } from "react";
import { ActionTileIcon } from "@/components/ActionIcon";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { useFileIcon } from "@/hooks/useFileIcon";
import type { ActionSpec, Target } from "@/lib/backend";
import { DROPBAR_MIME, payloadFromDataTransfer, TARGET_MIME } from "@/lib/dnd";
import { cn } from "@/lib/utils";

// Glyphs for the KeyModifier hint shown over a tile while dragging onto it.
const MODIFIER_GLYPH: Record<string, string> = {
  command: "⌘",
  cmd: "⌘",
  option: "⌥",
  alt: "⌥",
  shift: "⇧",
  control: "⌃",
  ctrl: "⌃",
};

interface TargetTileProps {
  target: Target;
  spec: ActionSpec | undefined;
  /** Tile width and icon-box size in px, column-aware (see GridPanel). */
  tilePx: number;
  iconPx: number;
  showKeyOverlay: boolean;
  optionHeld: boolean;
  onClick: () => void;
  onEdit: () => void;
  onDuplicate: () => void;
  onRemove: () => void;
  /** "Copy and Edit Script" — only wired for bundle (scripted) actions. */
  onCopyEditScript?: () => void;
  onDropBarItemDrop: (itemId: string) => void;
  onTextDrop: (text: string, isUrl: boolean) => void;
  onReorder: (draggedTargetId: string) => void;
}

export function TargetTile({
  target,
  spec,
  tilePx,
  iconPx,
  showKeyOverlay,
  optionHeld,
  onClick,
  onEdit,
  onDuplicate,
  onRemove,
  onCopyEditScript,
  onDropBarItemDrop,
  onTextDrop,
  onReorder,
}: TargetTileProps) {
  const [hover, setHover] = useState(false);
  // Folder and app tiles show the real Finder icon of their configured path.
  const nativeIcon = useFileIcon(target.options?.path || target.options?.app);

  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <button
          data-drop-id={target.id}
          draggable
          onDragStart={(e) => {
            e.dataTransfer.setData(TARGET_MIME, target.id);
            e.dataTransfer.effectAllowed = "move";
          }}
          className={cn(
            "group relative flex flex-col items-center gap-1 rounded-xl p-1.5 outline-none",
            "transition-transform duration-100",
            hover && !optionHeld && "scale-105",
            // Delete mode jiggles the tiles, like Dropzone / iOS edit mode.
            optionHeld && "animate-[dz-jiggle_0.32s_ease-in-out_infinite]",
          )}
          style={
            {
              "--wails-drop-target": "drop",
              width: tilePx,
              ...(optionHeld ? { animationDelay: `${(target.position % 4) * 45}ms` } : {}),
            } as React.CSSProperties
          }
          onClick={onClick}
          onDragOver={(e) => {
            e.preventDefault();
            setHover(true);
          }}
          onDragLeave={() => setHover(false)}
          onDrop={(e) => {
            e.preventDefault();
            setHover(false);
            const draggedTarget = e.dataTransfer.getData(TARGET_MIME);
            if (draggedTarget && draggedTarget !== target.id) {
              onReorder(draggedTarget);
              return;
            }
            const itemId = e.dataTransfer.getData(DROPBAR_MIME);
            if (itemId) {
              onDropBarItemDrop(itemId);
              return;
            }
            const payload = payloadFromDataTransfer(e.dataTransfer);
            if (payload?.text) onTextDrop(payload.text, payload.kind === "url");
          }}
        >
          <div
            className={cn(
              "flex items-center justify-center rounded-xl",
              "transition-all duration-100",
              // A dragged file darkens the hovered icon, like Finder's
              // drop-target folders — no ring or background.
              hover && "brightness-[0.6] saturate-150",
            )}
            style={{ width: iconPx, height: iconPx }}
          >
            {nativeIcon ? (
              <img
                src={`data:image/png;base64,${nativeIcon}`}
                alt=""
                className="size-full object-contain"
                draggable={false}
              />
            ) : (
              <ActionTileIcon actionId={target.actionId} icon={spec?.icon} className="size-[90%]" />
            )}
          </div>
          <span className="line-clamp-2 w-full text-center text-[10px] leading-tight text-neutral-300">
            {target.label}
          </span>
          {hover && spec?.keyModifier && MODIFIER_GLYPH[spec.keyModifier] && (
            <span
              className="absolute right-1 top-1 z-10 flex size-5 items-center justify-center rounded-md bg-black/70 text-[12px] font-semibold text-white shadow"
              title="Hold to change behavior on drop"
            >
              {MODIFIER_GLYPH[spec.keyModifier]}
            </span>
          )}
          {target.shortcut && showKeyOverlay && (
            <span className="absolute left-1/2 top-5 z-10 -translate-x-1/2 rounded-md bg-black/60 px-1.5 py-0.5 font-mono text-[12px] font-semibold text-white">
              {target.shortcut.toUpperCase()}
            </span>
          )}
          {optionHeld && (
            <span
              role="button"
              onClick={(e) => {
                e.stopPropagation();
                onRemove();
              }}
              className="absolute left-0.5 top-0.5 z-10 flex size-5 items-center justify-center rounded-md bg-neutral-600/90 shadow"
              title="Remove from Grid"
            >
              <X className="size-3 text-white" />
            </span>
          )}
          {target.actionId === "folder" && target.options?.mode === "copy" && (
            <span className="absolute bottom-5 right-2 z-10 flex size-4 items-center justify-center rounded-full bg-green-500 shadow">
              <Plus className="size-3 text-white" strokeWidth={3} />
            </span>
          )}
        </button>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onClick={onEdit}>Edit…</ContextMenuItem>
        <ContextMenuItem onClick={onDuplicate}>Duplicate</ContextMenuItem>
        {target.actionId.startsWith("bundle:") && onCopyEditScript && (
          <ContextMenuItem onClick={onCopyEditScript}>Copy and Edit Script</ContextMenuItem>
        )}
        <ContextMenuSeparator />
        <ContextMenuItem variant="destructive" onClick={onRemove}>
          Remove from Grid
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  );
}
