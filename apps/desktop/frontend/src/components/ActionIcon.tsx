import { iconFor, tileStyleFor } from "@/lib/icons"
import { cn } from "@/lib/utils"

/**
 * An action's large borderless tile icon, Dropzone-style: a bundle PNG when
 * the action ships one (data: URI), otherwise a colored shape with a white
 * glyph keyed off the action ID.
 */
export function ActionTileIcon({
  actionId,
  icon,
  className,
}: {
  actionId: string
  icon: string | undefined
  className?: string
}) {
  if (icon?.startsWith("data:")) {
    return (
      <img src={icon} alt="" draggable={false} className={cn("object-contain", className)} />
    )
  }
  const { glyph: Glyph, shape } = tileStyleFor(actionId, icon ?? "file")
  return (
    <span className={cn("flex items-center justify-center shadow-sm", shape, className)}>
      <Glyph className="size-[55%] text-white" strokeWidth={1.9} />
    </span>
  )
}

/** Small monochrome icon used in lists (e.g. the action catalogue). */
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
