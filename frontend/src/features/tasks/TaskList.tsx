import type { TaskState } from "@/lib/backend"
import { backend } from "@/lib/backend"
import { cn } from "@/lib/utils"
import { CircleAlert, CircleCheck, X } from "lucide-react"
import { Progress } from "@/components/ui/progress"

export function TaskList({ tasks }: { tasks: TaskState[] }) {
  if (tasks.length === 0) return null
  return (
    <div className="mx-3 flex flex-col gap-1.5 border-t border-white/10 pt-2">
      {tasks.slice(0, 4).map((task) => (
        <div key={task.id} className="group rounded-lg bg-white/[0.05] px-2.5 py-1.5">
          <div className="flex items-center gap-1.5">
            {task.status === "done" && (
              <CircleCheck className="size-3.5 shrink-0 text-emerald-400" />
            )}
            {task.status === "error" && (
              <CircleAlert className="size-3.5 shrink-0 text-red-400" />
            )}
            <span className="truncate text-[11px] font-medium text-neutral-200">
              {task.title}
            </span>
            <span
              className={cn(
                "ml-auto truncate text-[10px]",
                task.status === "error" ? "text-red-400" : "text-neutral-500"
              )}
            >
              {task.status === "error" ? task.error : task.detail}
            </span>
            {task.status !== "running" && (
              <button
                onClick={() => backend.tasks.dismiss(task.id)}
                className="hidden shrink-0 group-hover:block"
              >
                <X className="size-3 text-neutral-500 hover:text-neutral-200" />
              </button>
            )}
          </div>
          {task.status === "running" && (
            <Progress
              value={task.percent < 0 ? null : task.percent}
              className="mt-1 h-1"
            />
          )}
          {task.resultUrl && (
            <button
              onClick={() => window.open(task.resultUrl)}
              className="mt-0.5 truncate text-[10px] text-sky-400 hover:underline"
            >
              {task.resultUrl}
            </button>
          )}
        </div>
      ))}
    </div>
  )
}
