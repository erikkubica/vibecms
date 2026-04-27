import "@testing-library/jest-dom";
import React from "react";

// @vibecms/icons is aliased to src/__tests__/__mocks__/@vibecms/icons.ts in vitest.config.ts.
// All named exports are provided there as static no-op span components.

// Minimal React shims for shadcn/ui primitives consumed via window.__VIBECMS_SHARED__.ui

function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return React.createElement("input", props);
}
function Label({
  children,
  ...rest
}: React.LabelHTMLAttributes<HTMLLabelElement>) {
  return React.createElement("label", rest, children);
}
function Button({
  children,
  variant: _v,
  size: _s,
  asChild: _a,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: string;
  size?: string;
  asChild?: boolean;
}) {
  return React.createElement("button", rest, children);
}
function Textarea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return React.createElement("textarea", props);
}
function Switch({
  checked,
  onCheckedChange,
  disabled,
}: {
  checked: boolean;
  onCheckedChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return React.createElement("button", {
    role: "switch",
    "aria-checked": checked,
    disabled,
    "data-testid": "switch",
    onClick: () => onCheckedChange(!checked),
  });
}

// Collect SelectItem children and render as a native <select>
function Select({
  children,
  value,
  onValueChange,
}: {
  children: React.ReactNode;
  value?: string;
  onValueChange?: (v: string) => void;
}) {
  const options: { value: string; label: string }[] = [];

  const walk = (node: React.ReactNode) => {
    React.Children.forEach(node, (child) => {
      if (!React.isValidElement(child)) return;
      const el = child as React.ReactElement<any>;
      if ((el.type as any) === SelectItem) {
        options.push({
          value: el.props.value,
          label: typeof el.props.children === "string" ? el.props.children : el.props.value,
        });
      } else if (el.props?.children) {
        walk(el.props.children);
      }
    });
  };
  walk(children);

  return React.createElement(
    "select",
    { value, onChange: (e: React.ChangeEvent<HTMLSelectElement>) => onValueChange?.(e.target.value) },
    options.map((o) =>
      React.createElement("option", { key: o.value, value: o.value }, o.label),
    ),
  );
}
function SelectTrigger({ children }: { children: React.ReactNode }) {
  return React.createElement(React.Fragment, null, children);
}
function SelectValue({ placeholder }: { placeholder?: string }) {
  return React.createElement("span", null, placeholder);
}
function SelectContent({ children }: { children: React.ReactNode }) {
  return React.createElement(React.Fragment, null, children);
}
function SelectItem({
  children,
  value: _v,
}: {
  children: React.ReactNode;
  value: string;
}) {
  return React.createElement(React.Fragment, null, children);
}
function Badge({
  children,
  variant: _v,
  className,
}: {
  children: React.ReactNode;
  variant?: string;
  className?: string;
}) {
  return React.createElement("span", { className }, children);
}
function Dialog({
  open,
  onOpenChange: _oc,
  children,
}: {
  open: boolean;
  onOpenChange?: (v: boolean) => void;
  children: React.ReactNode;
}) {
  if (!open) return null;
  return React.createElement("div", { role: "dialog" }, children);
}
function DialogContent({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return React.createElement("div", { className }, children);
}
function DialogHeader({ children }: { children: React.ReactNode }) {
  return React.createElement("div", null, children);
}
function DialogTitle({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return React.createElement("h2", { className }, children);
}
function DialogFooter({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return React.createElement("div", { className }, children);
}
function Card({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return React.createElement("div", { className }, children);
}
function CardContent({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return React.createElement("div", { className }, children);
}
function Popover({
  open,
  onOpenChange,
  children,
}: {
  open?: boolean;
  onOpenChange?: (v: boolean) => void;
  children: React.ReactNode;
}) {
  return React.createElement(
    "div",
    { "data-popover": true, "data-open": open },
    children,
  );
}
function PopoverTrigger({
  children,
}: {
  children: React.ReactNode;
  asChild?: boolean;
}) {
  return React.createElement(React.Fragment, null, children);
}
function PopoverContent({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
  align?: string;
}) {
  return React.createElement("div", { className }, children);
}

// ---- New list-page + layout primitives needed by redesigned components ----

function SectionHeader({
  title,
  icon,
  actions,
}: {
  title: string;
  icon?: React.ReactNode;
  actions?: React.ReactNode;
}) {
  return React.createElement(
    "div",
    { "data-testid": "section-header" },
    icon,
    React.createElement("h2", null, title),
    actions,
  );
}

function AccordionRow({
  headerLeft,
  headerRight,
  children,
  open,
  onToggle,
}: {
  headerLeft: React.ReactNode;
  headerRight?: React.ReactNode;
  children?: React.ReactNode;
  open: boolean;
  onToggle: () => void;
}) {
  return React.createElement(
    "div",
    { "data-testid": "accordion-row" },
    React.createElement(
      "div",
      { onClick: onToggle, "data-testid": "accordion-header" },
      headerLeft,
      headerRight,
    ),
    open ? React.createElement("div", { "data-testid": "accordion-body" }, children) : null,
  );
}

function ListPageShell({ children }: { children: React.ReactNode }) {
  return React.createElement("div", { "data-testid": "list-page-shell" }, children);
}

function ListHeader({
  title,
  count,
  extra,
  newLabel,
  onNew,
}: {
  title: string;
  count?: number;
  extra?: React.ReactNode;
  newLabel?: string;
  onNew?: () => void;
  tabs?: any[];
  activeTab?: string;
  onTabChange?: (v: string) => void;
  newHref?: string;
}) {
  return React.createElement(
    "div",
    { "data-testid": "list-header" },
    React.createElement("h1", null, title),
    count !== undefined ? React.createElement("span", null, `${count} items`) : null,
    extra,
    onNew
      ? React.createElement("button", { type: "button", onClick: onNew }, newLabel ?? "New")
      : null,
  );
}

function ListToolbar({ children }: { children: React.ReactNode }) {
  return React.createElement("div", { "data-testid": "list-toolbar" }, children);
}

function ListSearch({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
}) {
  return React.createElement("input", {
    type: "text",
    value,
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => onChange(e.target.value),
    placeholder,
  });
}

function ListCard({ children }: { children: React.ReactNode }) {
  return React.createElement("div", { "data-testid": "list-card" }, children);
}

function ListTable({ children }: { children: React.ReactNode; minWidth?: number }) {
  return React.createElement("table", null, children);
}

function ListFooter() { return null; }

function Th({ children }: { children?: React.ReactNode; width?: number | string; align?: string; className?: string }) {
  return React.createElement("th", null, children);
}

function Tr({ children, className }: { children: React.ReactNode; className?: string }) {
  return React.createElement("tr", { className }, children);
}

function Td({ children, className, align }: { children?: React.ReactNode; className?: string; align?: string; onClick?: (e: any) => void }) {
  return React.createElement("td", { className }, children);
}

function StatusPill({ status, label }: { status: string; label?: string }) {
  return React.createElement("span", { "data-testid": "status-pill", "data-status": status }, label ?? status);
}

function Chip({ children }: { children: React.ReactNode }) {
  return React.createElement("span", { "data-testid": "chip" }, children);
}

function EmptyState({
  icon: _Icon,
  title,
  description,
  action,
}: {
  icon: any;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return React.createElement(
    "div",
    { "data-testid": "empty-state" },
    React.createElement("p", null, title),
    description ? React.createElement("p", null, description) : null,
    action,
  );
}

function LoadingRow() {
  return React.createElement("div", { "data-testid": "loading-row" }, "Loading…");
}

function Checkbox({
  checked,
  onCheckedChange,
  "aria-label": ariaLabel,
}: {
  checked: boolean;
  onCheckedChange: (v: boolean) => void;
  "aria-label"?: string;
}) {
  return React.createElement("input", {
    type: "checkbox",
    checked,
    "aria-label": ariaLabel,
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => onCheckedChange(e.target.checked),
  });
}

const ui = {
  Input,
  Label,
  Button,
  Textarea,
  Switch,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
  Badge,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Card,
  CardContent,
  Popover,
  PopoverTrigger,
  PopoverContent,
  // New primitives
  SectionHeader,
  AccordionRow,
  ListPageShell,
  ListHeader,
  ListToolbar,
  ListSearch,
  ListCard,
  ListTable,
  ListFooter,
  Th,
  Tr,
  Td,
  StatusPill,
  Chip,
  EmptyState,
  LoadingRow,
  Checkbox,
};

const Sonner = {
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
};

(window as any).__VIBECMS_SHARED__ = { ui, Sonner };
