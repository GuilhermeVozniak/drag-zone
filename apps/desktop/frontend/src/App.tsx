import { useEffect, useState } from "react";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { PanelChrome } from "@/components/PanelChrome";
import { TooltipProvider } from "@/components/ui/tooltip";
import { PopoutBar } from "@/features/dropbar/PopoutBar";
import { GridPanel } from "@/features/grid/GridPanel";
import { Onboarding } from "@/features/onboarding/Onboarding";
import { SettingsView } from "@/features/settings/SettingsView";
import { InputRequestDialog } from "@/features/tasks/InputRequestDialog";
import { useSettings } from "@/hooks/useBackend";
import { backend, events, uiScale } from "@/lib/backend";
import { setUIScale } from "@/lib/dnd";

function App() {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsTab, setSettingsTab] = useState("general");
  const [poppedOut, setPoppedOut] = useState(false);

  // Opening settings is a backend round-trip: the native layer flips the
  // shared window into settings mode (titled, centered, Dock icon visible)
  // and emits settings:open back, which is what actually mounts the view.
  const openSettings = (tab?: string) => {
    void backend.settings.open(tab);
  };
  const closeSettings = () => {
    void backend.settings.close();
  };
  const [settings, setSettings] = useSettings();

  // Show the first-run carousel until the user finishes or skips it.
  const showOnboarding = settings != null && !settings.onboardingSeen && !poppedOut;
  const dismissOnboarding = () => {
    if (settings) setSettings({ ...settings, onboardingSeen: true });
  };

  // Theme: "Always use dark mode" forces dark; otherwise follow the OS.
  // The class lives on <html> so portaled menus/dialogs inherit it.
  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      const dark = settings?.theme === "dark" || media.matches;
      document.documentElement.classList.toggle("dark", dark);
    };
    apply();
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [settings?.theme]);

  // Grid size scales the whole UI; drop hit-testing needs the same factor.
  const scale = uiScale(settings);
  useEffect(() => {
    setUIScale(scale);
  }, [scale]);

  useEffect(() => {
    const offOpen = events.onOpenSettings((tab) => {
      setSettingsTab(tab || "general");
      setSettingsOpen(true);
    });
    const offClose = events.onCloseSettings(() => setSettingsOpen(false));
    const offPopout = events.onDropBarPopOut(setPoppedOut);
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      // In settings mode Escape closes settings; otherwise it hides the grid.
      if (settingsOpen) {
        closeSettings();
      } else {
        backend.window.hide();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => {
      offOpen();
      offClose();
      offPopout();
      window.removeEventListener("keydown", onKey);
    };
  }, [settingsOpen]);

  return (
    <ErrorBoundary>
      <TooltipProvider delayDuration={400}>
        {/* The grid stays mounted under the settings view so its state
            survives a settings round-trip; settings is unscaled (it owns the
            window size in settings mode, not the grid's zoom). */}
        <div style={{ zoom: scale }}>
          <PanelChrome resizeEnabled={!settingsOpen}>
            {showOnboarding ? (
              <Onboarding onDone={dismissOnboarding} />
            ) : poppedOut ? (
              <PopoutBar />
            ) : (
              <GridPanel onOpenSettings={openSettings} />
            )}
          </PanelChrome>
        </div>
        {settingsOpen && <SettingsView tab={settingsTab} />}
        <InputRequestDialog />
      </TooltipProvider>
    </ErrorBoundary>
  );
}

export default App;
