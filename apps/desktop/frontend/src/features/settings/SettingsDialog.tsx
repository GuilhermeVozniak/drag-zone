import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useSettings } from "@/hooks/useBackend";
import { backend } from "@/lib/backend";
import { AddonsTab } from "./AddonsTab";
import { CommandLineTab } from "./CommandLineTab";
import { GeneralTab } from "./GeneralTab";
import { UpdatesTab } from "./UpdatesTab";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Tab to show when the dialog opens (e.g. "addons" from "Get More Actions…"). */
  tab?: string;
}

/**
 * Settings, organized in Dropzone 4's tabs (General, Add-on Actions,
 * Command Line, Updates). Rendered as a dialog since the app is a single
 * always-on-top window rather than a multi-window process.
 */
export function SettingsDialog({ open, onOpenChange, tab = "general" }: SettingsDialogProps) {
  const [settings, update] = useSettings();
  const [active, setActive] = useState(tab);
  // Jump to the requested tab each time the dialog opens (or the request
  // changes) — e.g. "Get More Actions…" opens straight to Add-on Actions.
  useEffect(() => {
    if (open) setActive(tab);
  }, [open, tab]);
  if (!settings) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[520px] overflow-y-auto border-white/10 bg-neutral-900 text-neutral-100 sm:max-w-[380px]">
        <DialogHeader>
          <DialogTitle className="text-sm">Settings</DialogTitle>
        </DialogHeader>
        <Tabs value={active} onValueChange={setActive}>
          <TabsList className="w-full text-xs">
            <TabsTrigger value="general">General</TabsTrigger>
            <TabsTrigger value="addons">Add-on Actions</TabsTrigger>
            <TabsTrigger value="cli">Command Line</TabsTrigger>
            <TabsTrigger value="updates">Updates</TabsTrigger>
          </TabsList>
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
        </Tabs>
        <div className="border-t border-white/10 pt-3">
          <Button
            variant="destructive"
            size="sm"
            className="w-full"
            onClick={() => backend.window.quit()}
          >
            Quit DragZone
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
