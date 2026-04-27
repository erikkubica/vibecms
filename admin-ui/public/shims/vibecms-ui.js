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

// List-page design system
export const ListPageShell = wrap("ListPageShell");
export const ListHeader = wrap("ListHeader");
export const ListToolbar = wrap("ListToolbar");
export const ListSearch = wrap("ListSearch");
export const ListCard = wrap("ListCard");
export const ListTable = wrap("ListTable");
export const Th = wrap("Th");
export const Tr = wrap("Tr");
export const Td = wrap("Td");
export const StatusPill = wrap("StatusPill");
export const Chip = wrap("Chip");
export const SlugLink = wrap("SlugLink");
export const TitleCell = wrap("TitleCell");
export const RowActions = wrap("RowActions");
export const ListFooter = wrap("ListFooter");
export const EmptyState = wrap("EmptyState");
export const LoadingRow = wrap("LoadingRow");

// Layout helpers
export const AccordionRow = wrap("AccordionRow");
export const SectionHeader = wrap("SectionHeader");

// shadcn/ui primitives
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
export const Checkbox = wrap("Checkbox");
export const Separator = wrap("Separator");
export const Dialog = wrap("Dialog");
export const DialogClose = wrap("DialogClose");
export const DialogContent = wrap("DialogContent");
export const DialogHeader = wrap("DialogHeader");
export const DialogTitle = wrap("DialogTitle");
export const DialogDescription = wrap("DialogDescription");
export const DialogFooter = wrap("DialogFooter");
export const DialogTrigger = wrap("DialogTrigger");
export const DialogPortal = wrap("DialogPortal");
export const DialogOverlay = wrap("DialogOverlay");
export const DropdownMenu = wrap("DropdownMenu");
export const DropdownMenuPortal = wrap("DropdownMenuPortal");
export const DropdownMenuTrigger = wrap("DropdownMenuTrigger");
export const DropdownMenuContent = wrap("DropdownMenuContent");
export const DropdownMenuGroup = wrap("DropdownMenuGroup");
export const DropdownMenuLabel = wrap("DropdownMenuLabel");
export const DropdownMenuItem = wrap("DropdownMenuItem");
export const DropdownMenuCheckboxItem = wrap("DropdownMenuCheckboxItem");
export const DropdownMenuRadioGroup = wrap("DropdownMenuRadioGroup");
export const DropdownMenuRadioItem = wrap("DropdownMenuRadioItem");
export const DropdownMenuSeparator = wrap("DropdownMenuSeparator");
export const DropdownMenuShortcut = wrap("DropdownMenuShortcut");
export const DropdownMenuSub = wrap("DropdownMenuSub");
export const DropdownMenuSubTrigger = wrap("DropdownMenuSubTrigger");
export const DropdownMenuSubContent = wrap("DropdownMenuSubContent");
export const Popover = wrap("Popover");
export const PopoverTrigger = wrap("PopoverTrigger");
export const PopoverContent = wrap("PopoverContent");
export const PopoverAnchor = wrap("PopoverAnchor");
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
export const Switch = wrap("Switch");

export const BlockPicker = wrap("BlockPicker");

// Rich editors
export const CodeEditor = wrap("CodeEditor");
export const RichTextEditor = wrap("RichTextEditor");
export const CodeViewer = wrap("CodeViewer");
export const CodeWindow = wrap("CodeWindow");

// shadcn Command (combobox)
export const Command = wrap("Command");
export const CommandDialog = wrap("CommandDialog");
export const CommandInput = wrap("CommandInput");
export const CommandList = wrap("CommandList");
export const CommandEmpty = wrap("CommandEmpty");
export const CommandGroup = wrap("CommandGroup");
export const CommandItem = wrap("CommandItem");
export const CommandShortcut = wrap("CommandShortcut");
export const CommandSeparator = wrap("CommandSeparator");
