import {
  ClipboardCopy,
  Contrast,
  FileArchive,
  FolderInput,
  Image as ImageIcon,
  Keyboard,
  Layers,
  Link2,
  type LucideIcon,
  Puzzle,
  ShieldCheck,
  Terminal,
  Trash2,
  UploadCloud,
  Wifi,
} from "lucide-react";
import { Card, CardDescription, CardTitle } from "@/components/ui/card";

const ACTIONS: { icon: LucideIcon; tint: string; title: string; body: string }[] = [
  {
    icon: Wifi,
    tint: "from-cyan-300 to-sky-500",
    title: "AirDrop",
    body: "Share a drop to any nearby device instantly.",
  },
  {
    icon: FileArchive,
    tint: "from-amber-300 to-orange-500",
    title: "Zip Files",
    body: "Compress whatever you drop into a single archive.",
  },
  {
    icon: FolderInput,
    tint: "from-sky-400 to-blue-500",
    title: "Move or Copy",
    body: "Route files to any folder — hold ⌥ to flip copy and move.",
  },
  {
    icon: UploadCloud,
    tint: "from-indigo-400 to-violet-600",
    title: "Upload anywhere",
    body: "S3, FTP/SFTP, Google Drive, Imgur — drop to upload and get a link back.",
  },
  {
    icon: ClipboardCopy,
    tint: "from-blue-400 to-indigo-600",
    title: "Copy to Clipboard",
    body: "Put file contents or paths straight on the clipboard.",
  },
  {
    icon: Link2,
    tint: "from-teal-300 to-cyan-500",
    title: "Shorten Links",
    body: "Drop a URL to get a short link back.",
  },
  {
    icon: ImageIcon,
    tint: "from-fuchsia-400 to-violet-600",
    title: "Convert & clean images",
    body: "Change formats and strip EXIF and location metadata.",
  },
  {
    icon: Trash2,
    tint: "from-rose-400 to-red-600",
    title: "Move to Trash",
    body: "Send files to the Trash from anywhere on screen.",
  },
];

const FEATURES: { icon: LucideIcon; title: string; body: string }[] = [
  {
    icon: Layers,
    title: "The Drop Bar shelf",
    body: "Stash files from anywhere, then drag them all back out together — perfect for gathering attachments.",
  },
  {
    icon: Keyboard,
    title: "Menu bar & hotkeys",
    body: "F3 drops the grid, F4 pops out the Drop Bar, and any tile can take a single-key shortcut.",
  },
  {
    icon: Terminal,
    title: "The dz command line",
    body: "Drive DragZone from the terminal: list actions, run them, and manage the Drop Bar.",
  },
  {
    icon: Puzzle,
    title: "Add-on actions",
    body: "Install community actions from aptonic/dropzone4-actions, or script your own.",
  },
  {
    icon: ShieldCheck,
    title: "Signed & notarized",
    body: "A universal build, Developer ID-signed and notarized by Apple — it opens with no warnings.",
  },
  {
    icon: Contrast,
    title: "Light & dark",
    body: "Follows the macOS appearance automatically, with an always-dark override.",
  },
];

function Chip({ icon: Icon, tint }: { icon: LucideIcon; tint?: string }) {
  if (tint) {
    return (
      <span
        className={`flex size-10 shrink-0 items-center justify-center rounded-xl border border-white/20 bg-gradient-to-br ${tint} text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.35)]`}
      >
        <Icon className="size-5" />
      </span>
    );
  }
  return (
    <span className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-white/12 bg-white/[0.06] text-sky-300">
      <Icon className="size-5" />
    </span>
  );
}

const H2 = "m-0 mb-3 text-center text-[clamp(28px,4vw,40px)] font-bold tracking-tight";

export function Features() {
  return (
    <>
      <section className="border-t border-white/10 py-16" id="actions">
        <h2 className={H2}>Every built-in action, ready to drop</h2>
        <p className="mx-auto mb-10 max-w-[560px] text-center text-muted-foreground">
          Drag a file onto a tile and it runs — the actions Dropzone 4 ships, plus a few more.
        </p>
        <div className="grid grid-cols-[repeat(auto-fit,minmax(250px,1fr))] gap-4">
          {ACTIONS.map((a) => (
            <Card className="flex gap-3.5 p-5" key={a.title}>
              <Chip icon={a.icon} tint={a.tint} />
              <div className="min-w-0">
                <CardTitle className="mb-1 text-base">{a.title}</CardTitle>
                <CardDescription className="text-sm">{a.body}</CardDescription>
              </div>
            </Card>
          ))}
        </div>
      </section>

      <section className="border-t border-white/10 py-16" id="features">
        <h2 className={H2}>Built for the menu bar</h2>
        <p className="mx-auto mb-10 max-w-[560px] text-center text-muted-foreground">
          Everything you'd expect from a native macOS menu bar app — and nothing you have to pay
          for.
        </p>
        <div className="grid grid-cols-[repeat(auto-fit,minmax(260px,1fr))] gap-4">
          {FEATURES.map((f) => (
            <Card className="p-6" key={f.title}>
              <Chip icon={f.icon} />
              <CardTitle className="mb-2 mt-4 text-lg">{f.title}</CardTitle>
              <CardDescription>{f.body}</CardDescription>
            </Card>
          ))}
        </div>
      </section>
    </>
  );
}
