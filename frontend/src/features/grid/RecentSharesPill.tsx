import { useEffect, useState } from "react"
import { backend, events, type Share } from "@/lib/backend"
import { ChevronDown } from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

/**
 * The "Recently Shared" pill in the grid header: a menu of recently
 * uploaded/shared URLs (click to reopen). Hidden until something was shared.
 */
export function RecentSharesPill() {
  const [shares, setShares] = useState<Share[]>([])

  useEffect(() => {
    backend.shares.list().then((v) => setShares(v ?? []))
    return events.onSharesChanged((v) => setShares(v ?? []))
  }, [])

  if (shares.length === 0) return null

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button className="flex items-center gap-1 rounded-full border border-white/15 px-2.5 py-0.5 text-[11px] text-neutral-300 hover:bg-white/10">
          Recently Shared
          <ChevronDown className="size-3" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="center" className="max-w-[280px]">
        {shares.map((s, i) => (
          <DropdownMenuItem key={i} onClick={() => backend.shares.open(s.url)}>
            <span className="truncate">
              <span className="text-neutral-400">{s.title}: </span>
              {s.url}
            </span>
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => backend.shares.clear()}>
          Clear Menu
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
