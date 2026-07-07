import { Package } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { type AddonInfo, backend } from "@/lib/backend";

type InstallState = "installing" | "done" | string; // string = error message

/**
 * Mirrors Dropzone 4's Add-on Actions tab: the live catalogue from the
 * official aptonic/dropzone4-actions repository with one-click install.
 */
export function AddonsTab() {
  const [addons, setAddons] = useState<AddonInfo[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [installState, setInstallState] = useState<Record<string, InstallState>>({});

  useEffect(() => {
    backend.addons
      .list()
      .then(setAddons)
      .catch((e) => setError(String(e)));
  }, []);

  const install = async (name: string) => {
    setInstallState((s) => ({ ...s, [name]: "installing" }));
    try {
      await backend.addons.install(name);
      setInstallState((s) => ({ ...s, [name]: "done" }));
    } catch (e) {
      setInstallState((s) => ({ ...s, [name]: String(e) }));
    }
  };

  if (error) {
    return <p className="py-6 text-center text-xs text-red-400">{error}</p>;
  }
  if (!addons) {
    return <p className="py-6 text-center text-xs text-neutral-500">Loading add-on actions…</p>;
  }

  return (
    <div className="flex flex-col">
      <p className="pb-2 text-[11px] text-neutral-500">
        Community actions from aptonic/dropzone4-actions. Installed actions appear in the “+” menu.
      </p>
      <div className="flex max-h-[300px] flex-col gap-0.5 overflow-y-auto pr-1">
        {addons.map((a) => {
          const state = installState[a.name];
          const installed = a.installed || state === "done";
          return (
            <div
              key={a.name}
              className="flex items-center gap-2.5 rounded-lg px-2 py-1.5 hover:bg-white/[0.07]"
            >
              <Package className="size-4 shrink-0 text-neutral-400" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs text-neutral-200">{a.name}</p>
                {state && state !== "installing" && state !== "done" && (
                  <p className="truncate text-[10px] text-red-400">{state}</p>
                )}
              </div>
              <Button
                size="sm"
                variant="secondary"
                className="h-6 px-2 text-[11px]"
                disabled={installed || state === "installing"}
                onClick={() => install(a.name)}
              >
                {installed ? "Installed" : state === "installing" ? "Installing…" : "Install"}
              </Button>
            </div>
          );
        })}
      </div>
      <Button
        variant="ghost"
        size="sm"
        className="mt-2 self-start text-xs"
        onClick={() => backend.actions.openFolder()}
      >
        Open Add-on Actions Folder
      </Button>
    </div>
  );
}
