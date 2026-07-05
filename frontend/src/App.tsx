import { useEffect, useState } from "react"
import { backend, events } from "@/lib/backend"
import { ErrorBoundary } from "@/components/ErrorBoundary"
import { TooltipProvider } from "@/components/ui/tooltip"
import { PopoutBar } from "@/features/dropbar/PopoutBar"
import { GridPanel } from "@/features/grid/GridPanel"
import { SettingsDialog } from "@/features/settings/SettingsDialog"
import { InputRequestDialog } from "@/features/tasks/InputRequestDialog"

function App() {
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [poppedOut, setPoppedOut] = useState(false)

  useEffect(() => {
    const offSettings = events.onOpenSettings(() => setSettingsOpen(true))
    const offPopout = events.onDropBarPopOut(setPoppedOut)
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") backend.window.hide()
    }
    window.addEventListener("keydown", onKey)
    return () => {
      offSettings()
      offPopout()
      window.removeEventListener("keydown", onKey)
    }
  }, [])

  return (
    <div className="dark h-screen">
      <ErrorBoundary>
        <TooltipProvider delayDuration={400}>
          {poppedOut ? (
            <PopoutBar />
          ) : (
            <GridPanel onOpenSettings={() => setSettingsOpen(true)} />
          )}
          <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
          <InputRequestDialog />
        </TooltipProvider>
      </ErrorBoundary>
    </div>
  )
}

export default App
