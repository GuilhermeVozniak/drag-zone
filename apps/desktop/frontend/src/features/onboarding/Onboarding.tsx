import {
  ArrowLeft,
  ArrowRight,
  Boxes,
  Keyboard,
  Layers,
  type LucideIcon,
  MousePointerSquareDashed,
  Puzzle,
} from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface Slide {
  icon: LucideIcon;
  title: string;
  body: string;
}

// First-run tour, mirroring Dropzone 4's six-slide welcome carousel.
const SLIDES: Slide[] = [
  {
    icon: Boxes,
    title: "Welcome to DragZone",
    body: "A menu bar shelf for your files. Drop things here to act on them, or stash them for later — without hunting through Finder windows.",
  },
  {
    icon: MousePointerSquareDashed,
    title: "Drop files onto actions",
    body: "Drag files onto a grid tile to zip, AirDrop, upload, copy, or run any installed action. The grid opens when you drag toward the menu bar.",
  },
  {
    icon: Layers,
    title: "Stash in the Drop Bar",
    body: "Drop files into the Drop Bar to collect them from anywhere, then drag them all back out together — perfect for gathering attachments.",
  },
  {
    icon: Keyboard,
    title: "Always a keystroke away",
    body: "Click the tray icon or press F3 to summon the grid; F4 pops out the Drop Bar. Give any tile a single-key shortcut in its settings.",
  },
  {
    icon: Puzzle,
    title: "Add-ons & the dz CLI",
    body: "Install more actions from Settings › Add-on Actions, and drive DragZone from the terminal with the bundled dz command-line tool.",
  },
];

/** First-run welcome carousel; onDone persists the dismissal via settings. */
export function Onboarding({ onDone }: { onDone: () => void }) {
  const [index, setIndex] = useState(0);
  const slide = SLIDES[index];
  const Icon = slide.icon;
  const last = index === SLIDES.length - 1;

  return (
    <div className="flex h-full flex-col px-6 pb-6 pt-4 text-center">
      <button
        onClick={onDone}
        className="self-end text-[11px] text-neutral-400 hover:text-neutral-200"
      >
        Skip
      </button>

      <div className="flex flex-1 flex-col items-center justify-center gap-4">
        <div className="flex size-20 items-center justify-center rounded-2xl bg-gradient-to-br from-sky-500/20 to-indigo-500/20 ring-1 ring-white/10">
          <Icon className="size-9 text-sky-400" strokeWidth={1.5} />
        </div>
        <h2 className="text-lg font-semibold text-neutral-100">{slide.title}</h2>
        <p className="max-w-[290px] text-[13px] leading-relaxed text-neutral-400">{slide.body}</p>
      </div>

      <div className="mb-4 flex items-center justify-center gap-1.5">
        {SLIDES.map((_, n) => (
          <button
            key={n}
            onClick={() => setIndex(n)}
            aria-label={`Slide ${n + 1}`}
            className={cn(
              "size-1.5 rounded-full transition-colors",
              n === index ? "bg-sky-400" : "bg-white/20 hover:bg-white/40",
            )}
          />
        ))}
      </div>

      <div className="flex items-center justify-between">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setIndex((i) => i - 1)}
          className={cn(index === 0 && "invisible")}
        >
          <ArrowLeft className="size-4" /> Back
        </Button>
        {last ? (
          <Button size="sm" onClick={onDone}>
            Get Started
          </Button>
        ) : (
          <Button size="sm" onClick={() => setIndex((i) => i + 1)}>
            Next <ArrowRight className="size-4" />
          </Button>
        )}
      </div>
    </div>
  );
}
