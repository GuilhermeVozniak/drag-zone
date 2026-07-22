import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TaskList } from "@/features/tasks/TaskList";
import { type ActionSpec, backend, type Target, type TaskState } from "@/lib/backend";

vi.mock("@/lib/backend");

const targets = [{ id: "t1", actionId: "zip" } as Target];
const specFor = vi.fn(() => ({ id: "zip", icon: "archive" }) as ActionSpec);
const task = (over: Partial<TaskState>): TaskState =>
  ({
    id: "k1",
    targetId: "t1",
    title: "Zipping",
    status: "running",
    percent: 50,
    ...over,
  }) as TaskState;

beforeEach(() => vi.clearAllMocks());

describe("TaskList", () => {
  it("renders a running task with title and detail", () => {
    render(<TaskList tasks={[task({ detail: "a.txt" })]} targets={targets} specFor={specFor} />);
    expect(screen.getByText("Zipping — a.txt")).toBeInTheDocument();
  });

  it("cancels a running task via the round button", async () => {
    const user = userEvent.setup();
    render(<TaskList tasks={[task({})]} targets={targets} specFor={specFor} />);
    await user.click(screen.getByTitle("Cancel"));
    expect(backend.tasks.cancel).toHaveBeenCalledWith("k1");
  });

  it("dismisses a finished task", async () => {
    const user = userEvent.setup();
    render(
      <TaskList
        tasks={[task({ status: "done", percent: 100 })]}
        targets={targets}
        specFor={specFor}
      />,
    );
    await user.click(screen.getByTitle("Dismiss"));
    expect(backend.tasks.dismiss).toHaveBeenCalledWith("k1");
  });

  it("opens a result URL when present", async () => {
    const user = userEvent.setup();
    render(
      <TaskList
        tasks={[task({ status: "done", percent: 100, resultUrl: "https://x.test/a" })]}
        targets={targets}
        specFor={specFor}
      />,
    );
    await user.click(screen.getByRole("button", { name: "https://x.test/a" }));
    expect(backend.shares.open).toHaveBeenCalledWith("https://x.test/a");
  });

  it('shows an error task as "title: error" in red', () => {
    render(
      <TaskList
        tasks={[task({ status: "error", title: "Zip", error: "boom" })]}
        targets={targets}
        specFor={specFor}
      />,
    );
    const line = screen.getByText("Zip: boom");
    expect(line).toBeInTheDocument();
    expect(line.className).toContain("text-red-400");
  });

  it("shows a cancelled task neutrally, never as an error", () => {
    render(
      <TaskList
        tasks={[task({ status: "cancelled", title: "Zip" })]}
        targets={targets}
        specFor={specFor}
      />,
    );
    const line = screen.getByText("Zip — cancelled");
    expect(line).toBeInTheDocument();
    expect(line.className).not.toContain("text-red-400");
  });

  it("offers dismiss (not cancel) for a cancelled task", async () => {
    const user = userEvent.setup();
    render(
      <TaskList tasks={[task({ status: "cancelled" })]} targets={targets} specFor={specFor} />,
    );
    await user.click(screen.getByTitle("Dismiss"));
    expect(backend.tasks.dismiss).toHaveBeenCalledWith("k1");
    expect(backend.tasks.cancel).not.toHaveBeenCalled();
  });

  it("caps the list at four rows", () => {
    const many = Array.from({ length: 6 }, (_, i) => task({ id: `k${i}`, title: `Task ${i}` }));
    render(<TaskList tasks={many} targets={targets} specFor={specFor} />);
    expect(screen.getAllByTitle("Cancel")).toHaveLength(4);
  });
});
