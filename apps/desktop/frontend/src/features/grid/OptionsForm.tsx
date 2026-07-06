import type { OptionField } from "@/lib/backend"
import { backend } from "@/lib/backend"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"

interface OptionsFormProps {
  fields: OptionField[]
  values: Record<string, string>
  onChange: (values: Record<string, string>) => void
}

export function OptionsForm({ fields, values, onChange }: OptionsFormProps) {
  const set = (key: string, value: string) => onChange({ ...values, [key]: value })

  return (
    <div className="flex flex-col gap-3">
      {fields.map((f) => (
        <div key={f.key} className="flex flex-col gap-1.5">
          <Label className="text-xs">{f.label}</Label>
          {f.type === "text" || f.type === "password" ? (
            <Input
              type={f.type}
              value={values[f.key] ?? f.default ?? ""}
              placeholder={f.placeholder}
              onChange={(e) => set(f.key, e.target.value)}
            />
          ) : f.type === "folder" || f.type === "file" || f.type === "app" ? (
            <div className="flex items-center gap-2">
              <span className="min-w-0 flex-1 truncate rounded-md border border-input bg-transparent px-2.5 py-1.5 text-xs text-neutral-400">
                {values[f.key] || "Not selected"}
              </span>
              <Button
                size="sm"
                variant="secondary"
                onClick={async () => {
                  const path =
                    f.type === "app"
                      ? await backend.dialogs.chooseApplication()
                      : await backend.dialogs.chooseFolder()
                  if (path) set(f.key, path)
                }}
              >
                Choose…
              </Button>
            </div>
          ) : f.type === "select" ? (
            <Select
              value={values[f.key] ?? f.default}
              onValueChange={(v) => set(f.key, v)}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(f.choices ?? []).map((c) => (
                  <SelectItem key={c} value={c}>
                    {c}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : f.type === "checkbox" ? (
            <Switch
              checked={(values[f.key] ?? f.default) === "true"}
              onCheckedChange={(v) => set(f.key, String(v))}
            />
          ) : null}
        </div>
      ))}
    </div>
  )
}
