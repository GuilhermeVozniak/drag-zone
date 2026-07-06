import { useEffect, useState } from "react"
import { backend, events } from "@/lib/backend"
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

interface InputRequest {
  id: string
  title: string
  prompt: string
}

/** Answers dz.inputbox requests from running action scripts. */
export function InputRequestDialog() {
  const [queue, setQueue] = useState<InputRequest[]>([])
  const [value, setValue] = useState("")
  const current = queue[0]

  useEffect(
    () =>
      events.onInputRequest((req) => {
        setQueue((q) => [...q, req])
      }),
    []
  )

  const answer = (ok: boolean) => {
    if (!current) return
    backend.answerInput(current.id, ok ? value : "", ok)
    setQueue((q) => q.slice(1))
    setValue("")
  }

  return (
    <Dialog open={!!current} onOpenChange={(open) => !open && answer(false)}>
      <DialogContent className="border-white/10 bg-neutral-900 text-neutral-100 sm:max-w-[340px]">
        <DialogHeader>
          <DialogTitle className="text-sm">{current?.title}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          <Label className="text-xs text-neutral-300">{current?.prompt}</Label>
          <Input
            autoFocus
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") answer(true)
            }}
          />
        </div>
        <DialogFooter>
          <Button variant="ghost" size="sm" onClick={() => answer(false)}>
            Cancel
          </Button>
          <Button size="sm" onClick={() => answer(true)}>
            OK
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
