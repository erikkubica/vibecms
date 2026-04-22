// UI component shim for extension micro-frontends.
// Each export is a thin wrapper that forwards to __VIBECMS_SHARED__.ui at call time.
// This avoids timing issues where the shim module evaluates before the SPA initializes.

function getUI(name) {
  const c = window.__VIBECMS_SHARED__?.ui?.[name];
  if (!c) {
    console.warn(`@vibecms/ui: component "${name}" not found in shared UI`);
    return (props) => null;
  }
  return c;
}

// React components need to be stable references for hooks to work.
// We create wrapper components that forward all props.
function wrap(name) {
  const Component = function (props) {
    const Real = getUI(name);
    const React = window.__VIBECMS_SHARED__?.React;
    return React ? React.createElement(Real, props) : null;
  };
  Component.displayName = name;
  return Component;
}

export const Button = wrap("Button");
export const Card = wrap("Card");
export const CardContent = wrap("CardContent");
export const CardHeader = wrap("CardHeader");
export const CardTitle = wrap("CardTitle");
export const CardDescription = wrap("CardDescription");
export const CardFooter = wrap("CardFooter");
export const CardAction = wrap("CardAction");
export const Input = wrap("Input");
export const Label = wrap("Label");
export const Badge = wrap("Badge");
export const Dialog = wrap("Dialog");
export const DialogContent = wrap("DialogContent");
export const DialogHeader = wrap("DialogHeader");
export const DialogTitle = wrap("DialogTitle");
export const DialogDescription = wrap("DialogDescription");
export const DialogFooter = wrap("DialogFooter");
export const Select = wrap("Select");
export const SelectContent = wrap("SelectContent");
export const SelectItem = wrap("SelectItem");
export const SelectTrigger = wrap("SelectTrigger");
export const SelectValue = wrap("SelectValue");
export const Tabs = wrap("Tabs");
export const TabsList = wrap("TabsList");
export const TabsTrigger = wrap("TabsTrigger");
export const TabsContent = wrap("TabsContent");
export const Textarea = wrap("Textarea");
export const Table = wrap("Table");
export const TableBody = wrap("TableBody");
export const TableCell = wrap("TableCell");
export const TableHead = wrap("TableHead");
export const TableHeader = wrap("TableHeader");
export const TableRow = wrap("TableRow");
export const TableFooter = wrap("TableFooter");
export const TableCaption = wrap("TableCaption");
export const Separator = wrap("Separator");
export const Checkbox = wrap("Checkbox");
export const Switch = wrap("Switch");
