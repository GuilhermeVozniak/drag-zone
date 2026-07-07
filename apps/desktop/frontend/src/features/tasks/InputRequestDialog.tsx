import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { backend, events } from "@/lib/backend";

interface InputRequest {
  id: string;
  title: string;
  prompt: string;
  /** When present, the user picks a button instead of typing (e.g. file
   *  conflicts: Keep Both / Replace / Stop). */
  choices?: string[];
}

/** Answers dz.inputbox text prompts and action choice prompts (conflicts). */
export function InputRequestDialog() {
  const [queue, setQueue] = useState<InputRequest[]>([]);
  const [value, setValue] = useState("");
  const current = queue[0];

  useEffect(
    () =>
      events.onInputRequest((req) => {
        setQueue((q) => [...q, req]);
      }),
    [],
  );

  const dequeue = () => {
    setQueue((q) => q.slice(1));
    setValue("");
  };

  // Text prompt: OK returns the typed value; Cancel returns not-answered.
  const answer = (ok: boolean) => {
    if (!current) return;
    backend.answerInput(current.id, ok ? value : "", ok);
    dequeue();
  };

  // Choice prompt: the picked label is the answer; closing = not-answered.
  const choose = (choice: string | null) => {
    if (!current) return;
    backend.answerInput(current.id, choice ?? "", choice != null);
    dequeue();
  };

  const isChoice = (current?.choices?.length ?? 0) > 0;

  return (
    <Dialog
      open={!!current}
      onOpenChange={(open) => !open && (isChoice ? choose(null) : answer(false))}
    >
      <DialogContent className="border-white/10 bg-neutral-900 text-neutral-100 sm:max-w-[340px]">
        <DialogHeader>
          <DialogTitle className="text-sm">{current?.title}</DialogTitle>
        </DialogHeader>
        <Label className="text-xs text-neutral-300">{current?.prompt}</Label>
        {isChoice ? (
          <DialogFooter className="gap-2 sm:justify-end">
            {current!.choices!.map((choice, i) => (
              <Button
                key={choice}
                size="sm"
                autoFocus={i === 0}
                variant={choice === "Stop" ? "destructive" : i === 0 ? "default" : "secondary"}
                onClick={() => choose(choice)}
              >
                {choice}
              </Button>
            ))}
          </DialogFooter>
        ) : (
          <>
            <Input
              autoFocus
              value={value}
              onChange={(e) => setValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") answer(true);
              }}
            />
            <DialogFooter>
              <Button variant="ghost" size="sm" onClick={() => answer(false)}>
                Cancel
              </Button>
              <Button size="sm" onClick={() => answer(true)}>
                OK
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
