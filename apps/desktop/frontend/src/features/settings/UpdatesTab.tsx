import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Switch } from "@/components/ui/switch";
import {
  backend,
  events,
  type Settings,
  type UpdateInfo,
  type UpdateProgress,
} from "@/lib/backend";
import { SettingRow } from "./SettingRow";

interface UpdatesTabProps {
  settings: Settings;
  update: (s: Settings) => void;
}

/** Mirrors Dropzone 4's Updates tab (backed by GitHub Releases). */
export function UpdatesTab({ settings, update }: UpdatesTabProps) {
  const [info, setInfo] = useState<UpdateInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [checking, setChecking] = useState(false);
  const [install, setInstall] = useState<UpdateProgress | null>(null);

  const check = async () => {
    setChecking(true);
    setError(null);
    try {
      setInfo(await backend.updates.check());
    } catch (e) {
      setError(String(e));
    } finally {
      setChecking(false);
    }
  };

  // Check when the tab opens so the answer is visible without a click.
  useEffect(() => {
    check();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Track the in-place install's progress stream; errors leave the stream.
  useEffect(
    () =>
      events.onUpdateProgress((p) => {
        setInstall(p.stage === "error" ? null : p);
        if (p.stage === "error") setError(p.error ?? "Update failed");
      }),
    [],
  );

  const installing = install != null && install.stage !== "done";

  return (
    <div className="flex flex-col gap-3.5">
      <SettingRow label="Automatically check for updates">
        <Switch
          checked={settings.autoUpdateCheck}
          onCheckedChange={(v) => update({ ...settings, autoUpdateCheck: v })}
        />
      </SettingRow>
      <div className="flex justify-center">
        <Button size="sm" variant="secondary" disabled={checking || installing} onClick={check}>
          {checking ? "Checking…" : "Check Now"}
        </Button>
      </div>
      {info && info.available && !install && (
        <div className="flex flex-col items-center gap-2">
          <p className="text-center text-[12px] font-medium text-neutral-100">
            Version {info.latest} is available
            {info.publishedAt ? ` (${new Date(info.publishedAt).toLocaleDateString()})` : ""}
          </p>
          <div className="flex items-center gap-2">
            {info.downloadUrl ? (
              <Button size="sm" onClick={() => backend.updates.install()}>
                Update to {info.latest}
              </Button>
            ) : (
              <Button size="sm" onClick={() => backend.openURL(info.url)}>
                Download {info.latest}
              </Button>
            )}
            <button
              className="text-[11px] text-sky-400 hover:underline"
              onClick={() => backend.openURL(info.url)}
            >
              release notes
            </button>
          </div>
        </div>
      )}
      {install && install.stage !== "done" && (
        <div className="flex flex-col items-center gap-1.5">
          <p className="text-[11px] text-neutral-300">
            {install.stage === "downloading"
              ? `Downloading ${install.version}… ${install.percent}%`
              : install.stage === "verifying"
                ? "Verifying signature…"
                : install.stage === "installing"
                  ? "Installing…"
                  : "Preparing update…"}
          </p>
          <Progress
            value={install.stage === "downloading" ? install.percent : null}
            className="h-1.5 w-56 bg-black/40 [&>div]:bg-sky-500"
          />
        </div>
      )}
      {install?.stage === "done" && (
        <p className="text-center text-[12px] font-medium text-green-400">
          Updated to {install.version} — relaunching…
        </p>
      )}
      {info && !info.available && !install && (
        <p className="text-center text-[11px] text-neutral-400">You're up to date.</p>
      )}
      {error && <p className="text-center text-[11px] text-red-400">{error}</p>}
      <p className="pt-2 text-center text-[11px] text-neutral-500">
        Version <VersionLabel />
      </p>
    </div>
  );
}

function VersionLabel() {
  const [version, setVersion] = useState("");
  useEffect(() => {
    backend.updates.version().then(setVersion);
  }, []);
  return <>{version || "…"}</>;
}
