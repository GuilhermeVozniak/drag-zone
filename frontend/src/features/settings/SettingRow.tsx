import { Label } from "@/components/ui/label"

/** One labeled settings row with the control on the right. */
export function SettingRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <Label className="text-xs text-neutral-300">{label}</Label>
      {children}
    </div>
  )
}

/** A small grey group heading (System / Behaviour / …). */
export function SettingGroup({ title }: { title: string }) {
  return (
    <p className="pt-1 text-[11px] font-semibold text-neutral-500">{title}</p>
  )
}

export const SHORTCUTS = [
  "Off",
  "F1",
  "F2",
  "F3",
  "F4",
  "F5",
  "F6",
  "F7",
  "F8",
  "F9",
  "F10",
  "F11",
  "F12",
]
