import { buttonVariants } from "@/components/ui/button";
import { APP_VERSION, downloadUrl } from "../lib/download";

// DragZone is macOS-only, so the primary call to action is always the mac
// universal build — no platform detection needed.
export function PrimaryDownload() {
  return (
    <a
      data-testid="primary-download"
      data-platform="darwin"
      className={buttonVariants({ variant: "default", size: "lg" })}
      href={downloadUrl("darwin", "universal", APP_VERSION)}
    >
      <svg viewBox="0 0 16 16" className="size-4" fill="currentColor" aria-hidden="true">
        <path d="M11.2 8.4c0-1.4 1.1-2.1 1.2-2.1-.6-1-1.7-1.1-2-1.1-.9-.1-1.7.5-2.1.5s-1.1-.5-1.8-.5c-.9 0-1.8.5-2.3 1.4-1 1.7-.3 4.2.7 5.6.5.7 1 1.4 1.8 1.4.7 0 1-.5 1.9-.5s1.1.5 1.8.5 1.3-.7 1.8-1.4c.6-.8.8-1.6.8-1.6s-1.6-.6-1.6-2.4zM9.8 4.2c.4-.5.7-1.2.6-1.9-.6 0-1.3.4-1.7.9-.4.4-.7 1.1-.6 1.8.7.1 1.3-.3 1.7-.8z" />
      </svg>
      Download for macOS
    </a>
  );
}
