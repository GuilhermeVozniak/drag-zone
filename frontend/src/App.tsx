import { useEffect, useState } from "react"
import { backend, events, uiScale } from "@/lib/backend"
import { setUIScale } from "@/lib/dnd"
import { useSettings } from "@/hooks/useBackend"
import { ErrorBoundary } from "@/components/ErrorBoundary"
import { PanelChrome } from "@/components/PanelChrome"
import { TooltipProvider } from "@/components/ui/tooltip"
import { PopoutBar } from "@/features/dropbar/PopoutBar"
import { GridPanel } from "@/features/grid/GridPanel"
import { SettingsDialog } from "@/features/settings/SettingsDialog"
import { InputRequestDialog } from "@/features/tasks/InputRequestDialog"

function App() {
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [poppedOut, setPoppedOut] = useState(false)
  const [settings] = useSettings()

  // Theme: "Always use dark mode" forces dark; otherwise follow the OS.
  // The class lives on <html> so portaled menus/dialogs inherit it.
  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)")
    const apply = () => {
      const dark = settings?.theme === "dark" || media.matches
      document.documentElement.classList.toggle("dark", dark)
    }
    apply()
    media.addEventListener("change", apply)
    return () => media.removeEventListener("change", apply)
  }, [settings?.theme])

  // Grid size scales the whole UI; drop hit-testing needs the same factor.
  const scale = uiScale(settings)
  useEffect(() => {
    setUIScale(scale)
  }, [scale])

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
    <div className="h-screen" style={{ zoom: scale }}>
      <ErrorBoundary>
        <TooltipProvider delayDuration={400}>
          <PanelChrome>
            {poppedOut ? (
              <PopoutBar />
            ) : (
              <GridPanel onOpenSettings={() => setSettingsOpen(true)} />
            )}
          </PanelChrome>
          <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
          <InputRequestDialog />
        </TooltipProvider>
      </ErrorBoundary>
    </div>
  )
}

export default App
