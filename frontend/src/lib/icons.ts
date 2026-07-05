// Maps backend ActionSpec.icon names to lucide components.
import {
  AppWindow,
  Archive,
  ClipboardCopy,
  File,
  Files,
  Folder,
  Globe,
  Link,
  Printer,
  Trash2,
  Type,
  Upload,
  Wifi,
  type LucideIcon,
} from "lucide-react"

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
}

export function iconFor(name: string): LucideIcon {
  return byName[name] ?? File
}
