import { useState } from "react"
import { backend } from "@/lib/backend"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

/**
 * The "Develop Action…" workflow: creates a template .dzbundle in the
 * actions folder, registers it, and opens it for editing.
 */
export function DevelopActionRow() {
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
