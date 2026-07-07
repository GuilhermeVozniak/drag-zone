import { X } from "lucide-react";
import { ActionTileIcon } from "@/components/ActionIcon";
import { Progress } from "@/components/ui/progress";
import type { ActionSpec, Target, TaskState } from "@/lib/backend";
import { backend } from "@/lib/backend";
import { cn } from "@/lib/utils";

interface TaskListProps {
  tasks: TaskState[];
  targets: Target[];
  specFor: (actionId: string) => ActionSpec | undefined;
}

/**
 * The TASK PROGRESS section rows: action icon, status label above a blue
 * progress bar, and a circular cancel/dismiss button — like Dropzone 4.
 */
export function TaskList({ tasks, targets, specFor }: TaskListProps) {
  return (
    <div className="flex flex-col gap-2 px-4 pb-2">
      {tasks.slice(0, 4).map((task) => {
        const target = targets.find((t) => t.id === task.targetId);
        const spec = target ? specFor(target.actionId) : undefined;
        const running = task.status === "running";
        return (
          <div key={task.id} className="flex items-center gap-2.5">
            <ActionTileIcon
              actionId={target?.actionId ?? ""}
              icon={spec?.icon}
              className="size-7 shrink-0"
            />
            <div className="min-w-0 flex-1">
              <p
                className={cn(
                  "truncate pb-1 text-[11px]",
                  task.status === "error" ? "text-red-400" : "text-neutral-300",
                )}
              >
                {task.status === "error"
                  ? `${task.title}: ${task.error}`
                  : task.detail
                    ? `${task.title} — ${task.detail}`
                    : `${task.title}…`}
              </p>
              <Progress
                value={task.percent < 0 ? null : task.percent}
                className="h-1.5 bg-black/40 [&>div]:bg-sky-500"
              />
              {task.resultUrl && (
                <button
                  onClick={() => backend.shares.open(task.resultUrl!)}
                  className="truncate pt-0.5 text-[10px] text-sky-400 hover:underline"
                >
                  {task.resultUrl}
                </button>
              )}
            </div>
            <button
              onClick={() =>
                running ? backend.tasks.cancel(task.id) : backend.tasks.dismiss(task.id)
              }
              className="flex size-5 shrink-0 items-center justify-center rounded-full border border-white/20 hover:bg-white/10"
              title={running ? "Cancel" : "Dismiss"}
            >
              <X className="size-3 text-neutral-300" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
