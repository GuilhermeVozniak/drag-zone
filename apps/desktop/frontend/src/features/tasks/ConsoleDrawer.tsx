import { Trash2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { backend, type ConsoleLine, events } from "@/lib/backend";

/**
 * The debug console, like Dropzone's: raw stdout/stderr from running action
 * scripts. It auto-opens when a script run fails (console:error) and lives
 * as a drawer at the bottom of the grid.
 */
export function ConsoleDrawer({ onClose }: { onClose: () => void }) {
  const [lines, setLines] = useState<ConsoleLine[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    backend.console
      .lines()
      .then((v) => setLines(v ?? []))
      .catch(() => setLines([]));
    return events.onConsoleChanged((v) => setLines(v ?? []));
  }, []);

  // Follow the tail as new output streams in.
  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [lines]);

  return (
    <div className="flex h-[140px] flex-col border-t border-white/10 bg-black/40">
      <div className="flex items-center justify-between px-3 py-1">
        <p className="text-[10px] font-semibold tracking-wider text-neutral-500">DEBUG CONSOLE</p>
        <div className="flex items-center gap-1">
          <button
            title="Clear console"
            onClick={() => backend.console.clear()}
            className="flex size-5 items-center justify-center rounded hover:bg-white/10"
          >
            <Trash2 className="size-3 text-neutral-400" />
          </button>
          <button
            title="Close console"
            onClick={onClose}
            className="flex size-5 items-center justify-center rounded hover:bg-white/10"
          >
            <X className="size-3 text-neutral-400" />
          </button>
        </div>
      </div>
      <div
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-y-auto px-3 pb-2 font-mono text-[10px] leading-relaxed text-neutral-300"
      >
        {lines.length === 0 ? (
          <p className="text-neutral-600">No script output yet.</p>
        ) : (
          lines.map((l, i) => (
            <p key={i} className="whitespace-pre-wrap break-all">
              {l.line}
            </p>
          ))
        )}
      </div>
    </div>
  );
}
