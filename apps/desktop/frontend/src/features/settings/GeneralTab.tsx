import type { Settings } from "@/lib/backend"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Slider } from "@/components/ui/slider"
import { Switch } from "@/components/ui/switch"
import { DevelopActionRow } from "./DevelopActionRow"
import { SettingGroup, SettingRow, SHORTCUTS } from "./SettingRow"

interface GeneralTabProps {
  settings: Settings
  update: (s: Settings) => void
}

/** Mirrors Dropzone 4's General settings tab. */
export function GeneralTab({ settings, update }: GeneralTabProps) {
  const shortcutSelect = (
    value: string,
    onChange: (v: string) => void
  ) => (
    <Select value={value || "Off"} onValueChange={(v) => onChange(v === "Off" ? "" : v)}>
      <SelectTrigger size="sm" className="w-[84px]">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {SHORTCUTS.map((s) => (
          <SelectItem key={s} value={s}>
            {s}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )

  return (
    <div className="flex flex-col gap-3.5">
      <SettingRow label="Grid size">
        <Slider
          className="w-[130px]"
          value={[settings.gridSize ?? 33]}
          min={0}
          max={100}
          step={1}
          onValueChange={([v]) => update({ ...settings, gridSize: v })}
        />
      </SettingRow>
      <SettingRow label="Open grid shortcut">
        {shortcutSelect(settings.globalShortcut, (v) =>
          update({ ...settings, globalShortcut: v })
        )}
      </SettingRow>
      <SettingRow label="Pop out Drop Bar shortcut">
        {shortcutSelect(settings.popOutShortcut, (v) =>
          update({ ...settings, popOutShortcut: v })
        )}
      </SettingRow>
      <SettingRow label="Grid columns">
        <Select
          value={String(settings.gridColumns || 4)}
          onValueChange={(v) => update({ ...settings, gridColumns: Number(v) })}
        >
          <SelectTrigger size="sm" className="w-[84px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {[3, 4, 5, 6].map((n) => (
              <SelectItem key={n} value={String(n)}>
                {n}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </SettingRow>
      <SettingRow label="Always use dark mode">
        <Switch
          checked={settings.theme === "dark"}
          onCheckedChange={(v) =>
            update({ ...settings, theme: v ? "dark" : "system" })
          }
        />
      </SettingRow>
      <SettingRow label="Animate grid opening and closing">
        <Switch
          checked={settings.animateGrid}
          onCheckedChange={(v) => update({ ...settings, animateGrid: v })}
        />
      </SettingRow>
      <SettingRow label="Show service key overlays">
        <Switch
          checked={settings.showKeyOverlays}
          onCheckedChange={(v) => update({ ...settings, showKeyOverlays: v })}
        />
      </SettingRow>

      <SettingGroup title="System" />
      <SettingRow label="Launch at login">
        <Switch
          checked={settings.launchAtLogin}
          onCheckedChange={(v) => update({ ...settings, launchAtLogin: v })}
        />
      </SettingRow>
      <SettingRow label="Notifications">
        <Switch
          checked={settings.notifyOnComplete}
          onCheckedChange={(v) => update({ ...settings, notifyOnComplete: v })}
        />
      </SettingRow>
      <SettingRow label="Play sounds">
        <Switch
          checked={settings.playSounds}
          onCheckedChange={(v) => update({ ...settings, playSounds: v })}
        />
      </SettingRow>

      <SettingGroup title="Behaviour" />
      <SettingRow label="Show drag target overlay when dragging items">
        <Switch
          checked={settings.dragOverlay}
          onCheckedChange={(v) => update({ ...settings, dragOverlay: v })}
        />
      </SettingRow>
      <SettingRow label="Keep Drop Bar items after drag out">
        <Switch
          checked={settings.dropBarKeepsItems}
          onCheckedChange={(v) => update({ ...settings, dropBarKeepsItems: v })}
        />
      </SettingRow>

      <SettingGroup title="Developer" />
      <DevelopActionRow />
    </div>
  )
}
