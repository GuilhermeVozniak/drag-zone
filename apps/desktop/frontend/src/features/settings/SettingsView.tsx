import { X } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useSettings } from "@/hooks/useBackend";
import { backend } from "@/lib/backend";
import { AddonsTab } from "./AddonsTab";
import { CommandLineTab } from "./CommandLineTab";
import { GeneralTab } from "./GeneralTab";
import { UpdatesTab } from "./UpdatesTab";

interface SettingsViewProps {
  /** Tab to show when the view opens (e.g. "addons" from "Get More Actions…"). */
  tab?: string;
}

/**
 * Settings as a full-window view, organized in Dropzone 4's tabs (General,
 * Add-on Actions, Command Line, Updates). In settings mode the shared
 * window is a regular titled app window with a Dock icon (see
 * dz_set_settings_mode in bridge_darwin.m), so this covers the whole window
 * with an opaque surface over the (still mounted) grid below. Closed via
 * the header button, Escape (see App.tsx), or the title bar's close button —
 * all routed to backend.settings.close().
 */
export function SettingsView({ tab = "general" }: SettingsViewProps) {
  const [settings, update] = useSettings();
  const [active, setActive] = useState(tab);
  // Jump to the requested tab each time the view opens — e.g. "Get More
  // Actions…" opens straight to Add-on Actions.
  useEffect(() => setActive(tab), [tab]);
  if (!settings) return null;

  return (
    <div
      role="dialog"
      aria-label="Settings"
      className="fixed inset-0 z-50 flex flex-col bg-neutral-900 text-neutral-100"
    >
      <header className="flex items-center justify-between border-b border-white/10 px-4 py-3">
        <h1 className="text-sm font-semibold">Settings</h1>
        <button
          onClick={() => backend.settings.close()}
          className="flex size-7 items-center justify-center rounded-md hover:bg-white/10"
          title="Close Settings"
        >
          <X className="size-4" />
        </button>
      </header>
      <Tabs value={active} onValueChange={setActive} className="flex min-h-0 flex-1 flex-col">
        <div className="border-b border-white/10 px-4 pt-2">
          <TabsList className="w-full text-xs">
            <TabsTrigger value="general">General</TabsTrigger>
            <TabsTrigger value="addons">Add-on Actions</TabsTrigger>
            <TabsTrigger value="cli">Command Line</TabsTrigger>
            <TabsTrigger value="updates">Updates</TabsTrigger>
          </TabsList>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
          <TabsContent value="general" className="pt-2">
            <GeneralTab settings={settings} update={update} />
          </TabsContent>
          <TabsContent value="addons" className="pt-2">
            <AddonsTab />
          </TabsContent>
          <TabsContent value="cli" className="pt-2">
            <CommandLineTab />
          </TabsContent>
          <TabsContent value="updates" className="pt-2">
            <UpdatesTab settings={settings} update={update} />
          </TabsContent>
        </div>
      </Tabs>
      <div className="border-t border-white/10 px-4 py-3">
        <Button variant="destructive" size="sm" onClick={() => backend.window.quit()}>
          Quit DragZone
        </Button>
      </div>
    </div>
  );
}
