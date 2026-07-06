import { useEffect, useState } from "react"
import { backend } from "@/lib/backend"
import { Button } from "@/components/ui/button"

/** Mirrors Dropzone 4's Command Line tab: install the dz tool and usage. */
export function CommandLineTab() {
  const [installed, setInstalled] = useState<boolean | null>(null)
  const [error, setError] = useState<string | null>(null)

  const refresh = () => backend.cli.installed().then(setInstalled)
  useEffect(() => {
    refresh()
  }, [])

  const install = async () => {
    setError(null)
    try {
      await backend.cli.install()
      await refresh()
    } catch (e) {
      setError(String(e))
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-neutral-300">
        The <code className="font-mono">dz</code> command line tool controls
        DragZone from the terminal.
      </p>
      <div className="flex items-center gap-2">
        <span className="text-xs text-neutral-400">
          {installed === null
            ? "Checking…"
            : installed
              ? "Installed at /usr/local/bin/dz"
              : "Not installed"}
        </span>
        {installed === false && (
          <Button size="sm" variant="secondary" onClick={install}>
            Install Command Line Tool
          </Button>
        )}
      </div>
      {error && <p className="text-[11px] text-red-400">{error}</p>}
      <pre className="overflow-x-auto rounded-lg bg-white/[0.07] p-2.5 font-mono text-[10px] leading-relaxed text-neutral-300">
        {`dz list                              list grid targets
dz run NAME dragged|clicked [FILES]  run an action
dz list-items [--json]               list Drop Bar items
dz add [--stack] FILES               stash files in Drop Bar
dz rename INDEX NAME|--reset         rename an item
dz remove|lock|unlock INDEX          manage items
dz clear                             clear the Drop Bar
dz open | close                      show / hide the grid
dz open-dropbar | close-dropbar      pop out / dock Drop Bar`}
      </pre>
    </div>
  )
}
