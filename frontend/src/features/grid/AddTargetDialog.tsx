import { useEffect, useMemo, useState } from "react"
import type { ActionSpec, Target } from "@/lib/backend"
import { backend } from "@/lib/backend"
import { ActionIcon } from "@/components/ActionIcon"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { OptionsForm } from "./OptionsForm"

interface AddTargetDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  specs: ActionSpec[]
  /** When set, the dialog edits this target instead of adding a new one. */
  editing?: Target | null
}

export function AddTargetDialog({
  open,
  onOpenChange,
  specs,
  editing,
}: AddTargetDialogProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [label, setLabel] = useState("")
  const [shortcut, setShortcut] = useState("")
  const [values, setValues] = useState<Record<string, string>>({})

  useEffect(() => {
    if (open) {
      setSelectedId(editing?.actionId ?? null)
      setLabel(editing?.label ?? "")
      setShortcut(editing?.shortcut ?? "")
      setValues(editing?.options ?? {})
    }
  }, [open, editing])

  const spec = useMemo(
    () => specs.find((s) => s.id === selectedId),
    [specs, selectedId]
  )

  const missingRequired = (spec?.options ?? []).some(
    (f) => f.required && !(values[f.key] ?? f.default)
  )

  const submit = async () => {
    if (!spec) return
    const finalValues = { ...values }
    for (const f of spec.options ?? []) {
      if (finalValues[f.key] === undefined && f.default) finalValues[f.key] = f.default
    }
    const finalLabel = label.trim() || spec.name
    if (editing) {
      await backend.grid.update({
        ...editing,
        label: finalLabel,
        options: finalValues,
        shortcut,
      })
    } else {
      const created = await backend.grid.add(spec.id, finalLabel, finalValues)
      if (shortcut) {
        await backend.grid.update({ ...created, shortcut })
      }
    }
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[480px] overflow-y-auto dark border-white/10 bg-neutral-900 text-neutral-100 sm:max-w-[380px]">
        <DialogHeader>
          <DialogTitle className="text-sm">
            {editing ? "Edit " + editing.label : spec ? "Add " + spec.name : "Add to Grid"}
          </DialogTitle>
        </DialogHeader>

        {!spec ? (
          <div className="grid grid-cols-1 gap-1">
            {specs.map((s) => {
              return (
                <button
                  key={s.id}
                  onClick={() => {
                    setSelectedId(s.id)
                    setLabel(s.name)
                  }}
                  className="flex items-center gap-3 rounded-lg px-2.5 py-2 text-left hover:bg-white/[0.08]"
                >
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-full border border-white/10 bg-white/[0.07]">
                    <ActionIcon
                      icon={s.icon}
                      className={s.icon.startsWith("data:") ? "size-6" : "size-4.5"}
                    />
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs font-medium">{s.name}</p>
                    <p className="truncate text-[11px] text-neutral-500">
                      {s.description}
                    </p>
                  </div>
                </button>
              )
            })}
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Name in grid</Label>
              <Input value={label} onChange={(e) => setLabel(e.target.value)} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Shortcut key (while grid is open)</Label>
              <Input
                value={shortcut}
                maxLength={1}
                placeholder="e.g. D"
                className="w-16 text-center font-mono uppercase"
                onChange={(e) => setShortcut(e.target.value.slice(-1))}
              />
            </div>
            <OptionsForm
              fields={spec.options ?? []}
              values={values}
              onChange={setValues}
            />
            <DialogFooter>
              {!editing && (
                <Button variant="ghost" size="sm" onClick={() => setSelectedId(null)}>
                  Back
                </Button>
              )}
              <Button size="sm" disabled={missingRequired} onClick={submit}>
                {editing ? "Save" : "Add to Grid"}
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
