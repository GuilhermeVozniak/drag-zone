import { useEffect, useState } from "react"
import { backend } from "@/lib/backend"

const cache = new Map<string, string>()

/** Finder icon for a path as a base64 PNG, fetched once and cached. */
export function useFileIcon(path: string | undefined): string | null {
  const [icon, setIcon] = useState<string | null>(() =>
    path ? (cache.get(path) ?? null) : null
  )
  useEffect(() => {
    if (!path) {
      setIcon(null)
      return
    }
    const cached = cache.get(path)
    if (cached !== undefined) {
      setIcon(cached || null)
      return
    }
    let stale = false
    backend.fileIcon(path).then((b64) => {
      cache.set(path, b64)
      if (!stale) setIcon(b64 || null)
    })
    return () => {
      stale = true
    }
  }, [path])
  return icon
}
