import { useState } from "react"
import { useSettings } from "@/hooks/useBackend"
import { backend } from "@/lib/backend"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"

const SHORTCUTS = ["Off", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"]

interface SettingsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  const [settings, update] = useSettings()
  if (!settings) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="dark border-white/10 bg-neutral-900 text-neutral-100 sm:max-w-[340px]">
        <DialogHeader>
          <DialogTitle className="text-sm">Settings</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          <SettingRow label="Toggle grid shortcut">
            <Select
              value={settings.globalShortcut || "Off"}
              onValueChange={(v) =>
                update({ ...settings, globalShortcut: v === "Off" ? "" : v })
              }
            >
              <SelectTrigger size="sm" className="w-[88px]">
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
          </SettingRow>
          <SettingRow label="Grid columns">
            <Select
              value={String(settings.gridColumns || 4)}
              onValueChange={(v) =>
                update({ ...settings, gridColumns: Number(v) })
              }
            >
              <SelectTrigger size="sm" className="w-[88px]">
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
          <SettingRow label="Launch at login">
            <Switch
              checked={settings.launchAtLogin}
              onCheckedChange={(v) => update({ ...settings, launchAtLogin: v })}
            />
          </SettingRow>
          <SettingRow label="Notify when tasks complete">
            <Switch
              checked={settings.notifyOnComplete}
              onCheckedChange={(v) => update({ ...settings, notifyOnComplete: v })}
            />
          </SettingRow>
          <SettingRow label="Keep Drop Bar items after drag out">
            <Switch
              checked={settings.dropBarKeepsItems}
              onCheckedChange={(v) =>
                update({ ...settings, dropBarKeepsItems: v })
              }
            />
          </SettingRow>
          <div className="flex flex-col gap-2 border-t border-white/10 pt-3">
            <DevelopActionRow />
            <Button
              variant="secondary"
              size="sm"
              className="w-full"
              onClick={() => backend.actions.openFolder()}
            >
              Open Add-on Actions Folder
            </Button>
            <Button
              variant="destructive"
              size="sm"
              className="w-full"
              onClick={() => backend.window.quit()}
            >
              Quit DragZone
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function DevelopActionRow() {
  const [name, setName] = useState("")
  const [lang, setLang] = useState("ruby")
  return (
    <div className="flex items-center gap-1.5">
      <Input
        value={name}
        placeholder="New action name"
        className="h-8 flex-1 text-xs"
        onChange={(e) => setName(e.target.value)}
      />
      <Select value={lang} onValueChange={setLang}>
        <SelectTrigger size="sm" className="w-[86px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="ruby">Ruby</SelectItem>
          <SelectItem value="python">Python</SelectItem>
        </SelectContent>
      </Select>
      <Button
        size="sm"
        variant="secondary"
        disabled={!name.trim()}
        onClick={async () => {
          await backend.actions.develop(name.trim(), lang)
          setName("")
        }}
      >
        Develop
      </Button>
    </div>
  )
}

function SettingRow({
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
