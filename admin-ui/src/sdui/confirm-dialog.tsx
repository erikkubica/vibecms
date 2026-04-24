import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

interface ConfirmOptions {
  title?: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "default" | "destructive";
}

type Resolver = (value: boolean) => void;

interface PendingRequest extends ConfirmOptions {
  resolve: Resolver;
}

// Module-level queue so non-React callers (the action handler) can await a
// dialog. The ConfirmDialogHost component below reads from this queue.
let pending: PendingRequest | null = null;
let notify: (() => void) | null = null;

/**
 * Open the global confirm dialog. Resolves `true` when the user confirms,
 * `false` when they cancel or dismiss. Must be called from a tree that has
 * rendered <ConfirmDialogHost /> (mounted in main.tsx).
 */
export function confirmDialog(opts: ConfirmOptions): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    // If a dialog is already open, reject the old one as cancelled so we
    // never get two stacked dialogs.
    if (pending) {
      pending.resolve(false);
    }
    pending = { ...opts, resolve };
    notify?.();
  });
}

export function ConfirmDialogHost(): ReactNode {
  const [, setTick] = useState(0);
  const currentRef = useRef<PendingRequest | null>(null);

  useEffect(() => {
    notify = () => {
      currentRef.current = pending;
      setTick((t) => t + 1);
    };
    return () => {
      notify = null;
    };
  }, []);

  const req = currentRef.current;
  const open = !!req && pending === req;

  const close = (value: boolean) => {
    if (!req) return;
    req.resolve(value);
    if (pending === req) pending = null;
    currentRef.current = null;
    setTick((t) => t + 1);
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && close(false)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{req?.title ?? "Are you sure?"}</DialogTitle>
          <DialogDescription>{req?.message}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => close(false)}>
            {req?.cancelLabel ?? "Cancel"}
          </Button>
          <Button
            variant={req?.variant === "destructive" ? "destructive" : "default"}
            onClick={() => close(true)}
          >
            {req?.confirmLabel ?? "Confirm"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
