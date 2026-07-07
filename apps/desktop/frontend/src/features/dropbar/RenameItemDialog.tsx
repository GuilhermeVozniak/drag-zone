import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { backend, type DropBarItem } from "@/lib/backend";

interface RenameItemDialogProps {
  item: DropBarItem;
  /** The label being edited, or null when the dialog is closed. */
  value: string | null;
  onValueChange: (value: string | null) => void;
}

/** Renames a Drop Bar item; "Reset" restores the label derived from content. */
export function RenameItemDialog({ item, value, onValueChange }: RenameItemDialogProps) {
  const commit = (label: string) => {
    backend.dropBar.rename(item.id, label);
    onValueChange(null);
  };

  return (
    <Dialog open={value !== null} onOpenChange={(open) => !open && onValueChange(null)}>
      <DialogContent className="border-white/10 bg-neutral-900 text-neutral-100 sm:max-w-[300px]">
        <DialogHeader>
          <DialogTitle className="text-sm">Rename Item</DialogTitle>
        </DialogHeader>
        <Input
          autoFocus
          value={value ?? ""}
          onChange={(e) => onValueChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && value !== null) commit(value);
          }}
        />
        <DialogFooter>
          <Button variant="ghost" size="sm" onClick={() => commit("")}>
            Reset
          </Button>
          <Button size="sm" onClick={() => value !== null && commit(value)}>
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
