import { toast } from "sonner";

/**
 * Surfaces a backend failure to the user. Wails binding rejections carry the
 * Go error string; anything else is stringified. Use for fire-and-forget
 * backend calls whose failure the user must see.
 */
export function reportError(context: string, err: unknown) {
  const message = err instanceof Error ? err.message : String(err);
  toast.error(context, { description: message });
}
