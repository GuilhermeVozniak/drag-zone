import { Features } from "../components/Features";
import { PanelMock } from "../components/PanelMock";
import { PrimaryDownload } from "../components/PrimaryDownload";
import { buttonVariants } from "../components/ui/button";
import { APP_VERSION, latestReleaseUrl } from "../lib/download";

export default function Home() {
  return (
    <main className="mx-auto max-w-[1120px] px-6">
      <section className="grid items-center gap-12 pb-20 pt-20 lg:grid-cols-[1.02fr_0.98fr] lg:pt-28">
        <div className="text-center lg:text-left">
          <span className="inline-block rounded-full border border-white/15 bg-white/[0.06] px-4 py-1.5 text-sm text-muted-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.12)] backdrop-blur-md">
            A free, open-source Dropzone 4 for macOS
          </span>
          <h1 className="mx-auto mb-4 mt-6 max-w-[560px] text-[clamp(44px,7vw,74px)] font-bold leading-[1.02] tracking-tight lg:mx-0">
            Every file, one{" "}
            <span className="bg-gradient-to-r from-sky-300 via-indigo-300 to-violet-300 bg-clip-text text-transparent">
              drop
            </span>{" "}
            away.
          </h1>
          <p className="mx-auto mb-8 max-w-[520px] text-xl text-muted-foreground lg:mx-0">
            DragZone turns your menu bar into a drop shelf. Drag files up, drop them on an action —
            zip, AirDrop, upload, move — and it just runs.
          </p>
          <div className="mb-5 flex flex-wrap items-center justify-center gap-3 lg:justify-start">
            <PrimaryDownload />
            <a
              className={buttonVariants({ variant: "glass", size: "lg" })}
              href={latestReleaseUrl()}
            >
              All releases
            </a>
          </div>
          <p className="m-0 text-sm text-muted-foreground">
            Version {APP_VERSION} · macOS 12+ · universal (Apple Silicon &amp; Intel) · signed &amp;
            notarized
          </p>
        </div>

        <div className="flex justify-center lg:justify-end">
          <PanelMock />
        </div>
      </section>

      <Features />

      <footer className="border-t border-white/10 py-12 text-center text-muted-foreground">
        <p className="m-0 mb-2">
          Free and open-source.{" "}
          <a
            className="text-sky-300 no-underline hover:underline"
            href="https://github.com/GuilhermeVozniak/drag-zone"
          >
            Source on GitHub
          </a>
          .
        </p>
        <p className="m-0 text-sm opacity-80">
          A Dropzone 4–style menu bar app, rebuilt in Go &amp; Wails — every feature, free.
        </p>
      </footer>
    </main>
  );
}
