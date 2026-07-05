import { iconFor } from "@/lib/icons"
import { cn } from "@/lib/utils"

/** Renders an action's icon: a bundled PNG (data: URI) or a lucide glyph. */
export function ActionIcon({
  icon,
  className,
}: {
  icon: string | undefined
  className?: string
}) {
  if (icon?.startsWith("data:")) {
    return (
      <img src={icon} alt="" draggable={false} className={cn("object-contain", className)} />
    )
  }
  const Icon = iconFor(icon ?? "file")
  return <Icon className={cn("text-neutral-100", className)} strokeWidth={1.75} />
}
