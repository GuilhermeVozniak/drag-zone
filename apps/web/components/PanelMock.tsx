import {
  ArrowDownToLine,
  ClipboardCopy,
  FileArchive,
  FileText,
  Folder,
  Plus,
  Trash2,
  Wifi,
} from "lucide-react";
import type { ReactNode } from "react";

function TrayGlyph({ className = "size-3.5" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M12 3v9" />
      <path d="m8 10 4 4 4-4" />
      <path d="M4 14v3a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-3" />
    </svg>
  );
}

function Tile({
  label,
  gradient,
  icon,
  badge,
  highlight,
}: {
  label: string;
  gradient: string;
  icon: ReactNode;
  badge?: ReactNode;
  highlight?: boolean;
}) {
  return (
    <div className="flex w-16 flex-col items-center gap-1.5">
      <div
        className={`relative flex size-12 items-center justify-center rounded-[15px] border border-white/20 bg-gradient-to-br ${gradient} text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.35)] ${
          highlight ? "ring-2 ring-sky-300 ring-offset-2 ring-offset-[#2c2c30] brightness-110" : ""
        }`}
      >
        {icon}
        {badge}
      </div>
      <span className="w-full truncate text-center text-[10px] text-white/55">{label}</span>
    </div>
  );
}

function Section({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="border-t border-white/[0.08]">
      <p className="px-4 pb-1.5 pt-2.5 text-[9px] font-semibold tracking-[0.14em] text-white/40">
        {label}
      </p>
      <div className="flex gap-1 px-3 pb-2">{children}</div>
    </div>
  );
}

/**
 * The product itself, rendered as it appears in real life: a borderless panel
 * dropping from the menu bar's tray glyph, mid-interaction with a file being
 * dragged onto the Zip action. Purely decorative.
 */
export function PanelMock() {
  return (
    <div aria-hidden="true" className="relative mx-auto w-[344px] max-w-full">
      {/* menu bar with the highlighted tray glyph the panel hangs from */}
      <div className="mb-2.5 flex h-7 items-center justify-end gap-3 rounded-lg border border-white/10 bg-white/[0.05] px-3 text-[11px] text-white/45 backdrop-blur-md">
        <span>Fri 9:41</span>
        <span className="flex size-5 items-center justify-center rounded-[6px] bg-white/15 text-white">
          <TrayGlyph />
        </span>
      </div>

      {/* upward beak pointing at the tray glyph */}
      <div className="absolute right-[15px] top-[25px] size-3 rotate-45 rounded-[3px] border-l border-t border-white/10 bg-[#303030]" />

      <div className="dz-drop relative overflow-hidden rounded-[18px] border border-white/10 bg-[#303030] shadow-[0_44px_120px_-28px_rgba(0,0,0,0.92)]">
        {/* header */}
        <div className="flex items-center justify-between px-3 py-2">
          <span className="flex size-6 items-center justify-center rounded-md text-white/55">
            <Plus className="size-4" />
          </span>
          <span className="rounded-full border border-white/12 bg-white/[0.06] px-2.5 py-0.5 text-[10px] text-white/55">
            Recently Shared ▾
          </span>
          <span className="flex gap-0.5 text-white/40">
            <span className="size-1 rounded-full bg-current" />
            <span className="size-1 rounded-full bg-current" />
            <span className="size-1 rounded-full bg-current" />
          </span>
        </div>

        {/* drop bar row */}
        <div className="flex items-start gap-1 px-3 pb-2">
          <div className="flex w-16 flex-col items-center gap-1.5">
            <span className="flex size-12 items-center justify-center rounded-[15px] border-2 border-dashed border-white/25 text-white/45">
              <ArrowDownToLine className="size-5" />
            </span>
            <span className="text-[10px] text-white/45">Drop Bar</span>
          </div>
          <div className="flex w-16 flex-col items-center gap-1.5">
            <span className="relative size-12">
              <span className="absolute inset-0 m-auto size-9 -rotate-[12deg] rounded-[4px] border-2 border-white bg-neutral-200 shadow" />
              <span className="absolute inset-0 m-auto size-9 rotate-[9deg] rounded-[4px] border-2 border-white bg-neutral-100 shadow" />
              <span className="absolute inset-0 m-auto flex size-9 items-center justify-center rounded-[4px] border-2 border-white bg-gradient-to-br from-sky-200 to-indigo-200 text-[9px] font-semibold text-indigo-900 shadow">
                IMG
              </span>
            </span>
            <span className="text-[10px] text-white/55">3 Items</span>
          </div>
        </div>

        <Section label="FOLDERS / APPS">
          <Tile
            label="Desktop"
            gradient="from-sky-400 to-blue-500"
            icon={<Folder className="size-6" />}
            badge={
              <span className="absolute -bottom-1 -right-1 flex size-4 items-center justify-center rounded-full border-2 border-[#303030] bg-emerald-500">
                <Plus className="size-2.5" strokeWidth={3} />
              </span>
            }
          />
          <Tile
            label="Downloads"
            gradient="from-sky-400 to-blue-500"
            icon={<Folder className="size-6" />}
          />
        </Section>

        <Section label="ACTIONS">
          <Tile
            label="AirDrop"
            gradient="from-cyan-300 to-sky-500"
            icon={<Wifi className="size-6" />}
          />
          <Tile
            label="Zip Files"
            gradient="from-amber-300 to-orange-500"
            icon={<FileArchive className="size-6" />}
            highlight
          />
          <Tile
            label="Clipboard"
            gradient="from-indigo-400 to-violet-600"
            icon={<ClipboardCopy className="size-6" />}
          />
          <Tile
            label="Trash"
            gradient="from-rose-400 to-red-600"
            icon={<Trash2 className="size-6" />}
          />
        </Section>
      </div>

      {/* the file being dragged onto the Zip action */}
      <div className="dz-float absolute -right-3 bottom-8 flex items-center gap-2 rounded-xl border border-white/25 bg-white/95 px-3 py-2 text-neutral-800 shadow-[0_20px_50px_-12px_rgba(0,0,0,0.7)] sm:-right-8">
        <FileText className="size-5 text-indigo-500" />
        <span className="text-[13px] font-medium">report.pdf</span>
        <span className="absolute -bottom-2 -left-2 flex size-5 items-center justify-center rounded-full bg-indigo-500 text-[10px] font-bold text-white shadow">
          1
        </span>
      </div>
    </div>
  );
}
