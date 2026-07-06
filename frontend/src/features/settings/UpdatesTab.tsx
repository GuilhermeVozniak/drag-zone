import { useEffect, useState } from "react"
import { backend, type Settings, type UpdateInfo } from "@/lib/backend"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { SettingRow } from "./SettingRow"

interface UpdatesTabProps {
  settings: Settings
  update: (s: Settings) => void
}

/** Mirrors Dropzone 4's Updates tab (backed by the GitHub repository). */
export function UpdatesTab({ settings, update }: UpdatesTabProps) {
  const [info, setInfo] = useState<UpdateInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [checking, setChecking] = useState(false)

  const check = async () => {
    setChecking(true)
    setError(null)
    try {
      setInfo(await backend.updates.check())
    } catch (e) {
      setError(String(e))
    } finally {
      setChecking(false)
    }
  }

  return (
    <div className="flex flex-col gap-3.5">
      <SettingRow label="Automatically check for updates">
        <Switch
          checked={settings.autoUpdateCheck}
          onCheckedChange={(v) => update({ ...settings, autoUpdateCheck: v })}
        />
      </SettingRow>
      <div className="flex justify-center">
        <Button size="sm" variant="secondary" disabled={checking} onClick={check}>
          {checking ? "Checking…" : "Check Now"}
        </Button>
      </div>
      {info && (
        <p className="text-center text-[11px] text-neutral-400">
          Latest revision: {info.latestSha}
          {info.latestAt ? ` (${new Date(info.latestAt).toLocaleString()})` : ""}
          {" — "}
          <button
            className="text-sky-400 hover:underline"
            onClick={() => window.open(info.url)}
          >
            open repository
          </button>
        </p>
      )}
      {error && <p className="text-center text-[11px] text-red-400">{error}</p>}
      <p className="pt-2 text-center text-[11px] text-neutral-500">
        Version <VersionLabel />
      </p>
    </div>
  )
}

function VersionLabel() {
  const [version, setVersion] = useState("")
  useEffect(() => {
    backend.updates.version().then(setVersion)
  }, [])
  return <>{version || "…"}</>
}
