// Icon and color treatment for action tiles. Dropzone 4 shows large
// borderless, branded/colorful icons; we approximate with a colored shape
// (circle or rounded square) behind a white glyph.
import {
  AppWindow,
  Archive,
  ClipboardCopy,
  File,
  Files,
  Folder,
  Globe,
  HardDriveUpload,
  Image,
  ImageOff,
  Link,
  type LucideIcon,
  Printer,
  Trash2,
  Type,
  Upload,
  Wifi,
} from "lucide-react";

const byName: Record<string, LucideIcon> = {
  "app-window": AppWindow,
  archive: Archive,
  "clipboard-copy": ClipboardCopy,
  file: File,
  files: Files,
  folder: Folder,
  globe: Globe,
  link: Link,
  printer: Printer,
  "trash-2": Trash2,
  type: Type,
  upload: Upload,
  wifi: Wifi,
};

export function iconFor(name: string): LucideIcon {
  return byName[name] ?? File;
}

export interface TileStyle {
  glyph: LucideIcon;
  /** Tailwind classes for the colored shape behind the glyph. */
  shape: string;
}

// Per-action tile treatment, keyed by action spec ID.
const tileStyles: Record<string, TileStyle> = {
  airdrop: { glyph: Wifi, shape: "rounded-full bg-gradient-to-b from-sky-400 to-blue-600" },
  zip: { glyph: Archive, shape: "rounded-[14px] bg-gradient-to-b from-amber-400 to-orange-600" },
  "copy-to-clipboard": {
    glyph: ClipboardCopy,
    shape: "rounded-[14px] bg-gradient-to-b from-neutral-500 to-neutral-700",
  },
  trash: { glyph: Trash2, shape: "rounded-full bg-gradient-to-b from-rose-400 to-red-600" },
  "install-app": {
    glyph: AppWindow,
    shape: "rounded-[14px] bg-gradient-to-b from-violet-400 to-purple-600",
  },
  "save-text": { glyph: Type, shape: "rounded-[14px] bg-gradient-to-b from-cyan-400 to-sky-600" },
  print: { glyph: Printer, shape: "rounded-[14px] bg-gradient-to-b from-slate-400 to-slate-600" },
  "shorten-url": { glyph: Link, shape: "rounded-full bg-gradient-to-b from-blue-400 to-blue-600" },
  imgur: { glyph: Image, shape: "rounded-[14px] bg-gradient-to-b from-emerald-400 to-green-600" },
  "ftp-upload": {
    glyph: HardDriveUpload,
    shape: "rounded-[14px] bg-gradient-to-b from-indigo-400 to-indigo-600",
  },
  "s3-upload": {
    glyph: Upload,
    shape: "rounded-[14px] bg-gradient-to-b from-orange-400 to-amber-600",
  },
  "google-drive": {
    glyph: HardDriveUpload,
    shape: "rounded-full bg-gradient-to-b from-lime-400 to-green-600",
  },
  "convert-images": {
    glyph: Image,
    shape: "rounded-[14px] bg-gradient-to-b from-teal-400 to-teal-600",
  },
  "remove-metadata": {
    glyph: ImageOff,
    shape: "rounded-[14px] bg-gradient-to-b from-fuchsia-400 to-pink-600",
  },
};

export function tileStyleFor(actionId: string, iconName: string): TileStyle {
  return (
    tileStyles[actionId] ?? {
      glyph: iconFor(iconName),
      shape: "rounded-[14px] bg-gradient-to-b from-neutral-500 to-neutral-700",
    }
  );
}
