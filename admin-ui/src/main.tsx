import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, useNavigate } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "@/components/ui/sonner";
import App from "@/App";
import { queryClient } from "@/sdui/query-client";
import { useSSE } from "@/hooks/use-sse";
import { registerBuiltinComponents } from "@/sdui/register-builtins";
import { setNavigate } from "@/sdui/action-handler";
import { ConfirmDialogHost } from "@/sdui/confirm-dialog";
import "./index.css";

// Register SDUI built-in components before any rendering.
registerBuiltinComponents();

// Expose shared libraries for extension micro-frontends
import * as React from "react";
import * as ReactDOM from "react-dom/client";
import * as ReactRouterDOM from "react-router-dom";
import * as Sonner from "sonner";
import * as LucideReact from "lucide-react";

// List-page design system — shared so extensions render identical headers/toolbars
import {
  ListPageShell,
  ListHeader,
  ListToolbar,
  ListSearch,
  ListCard,
  ListTable,
  Th,
  Tr,
  Td,
  StatusPill,
  Chip,
  SlugLink,
  TitleCell,
  RowActions,
  ListFooter,
  EmptyState,
  LoadingRow,
} from "@/components/ui/list-page";

// Generic layout helpers
import { AccordionRow } from "@/components/ui/accordion-row";
import { SectionHeader } from "@/components/ui/section-header";

// shadcn/ui components — explicit named imports for reliable sharing
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
  CardAction,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogTrigger,
  DialogPortal,
  DialogOverlay,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuPortal,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuItem,
  DropdownMenuCheckboxItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
} from "@/components/ui/dropdown-menu";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverAnchor,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableFooter,
  TableCaption,
} from "@/components/ui/table";
import { Switch } from "@/components/ui/switch";

// Block picker — generic item picker dialog, data-agnostic
import BlockPicker from "@/components/ui/block-picker";

// Rich editors — exposed so extensions don't re-bundle heavy deps
import CodeEditor from "@/components/ui/code-editor";
import RichTextEditor from "@/components/ui/rich-text-editor";
import CodeViewer from "@/components/ui/code-viewer";
import { CodeWindow } from "@/components/ui/code-window";

// shadcn Command (combobox primitive)
import {
  Command,
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandShortcut,
  CommandSeparator,
} from "@/components/ui/command";

// API client
import * as apiClient from "@/api/client";

// Host components for extension slot rendering
import { ExtensionSlot } from "@/components/extension-slot";
import { useExtensions } from "@/hooks/use-extensions";

declare global {
  interface Window {
    __VIBECMS_SHARED__: Record<string, unknown>;
  }
}

window.__VIBECMS_SHARED__ = {
  React,
  ReactDOM,
  ReactRouterDOM,
  Sonner,
  icons: LucideReact,
  ui: {
    // List-page design system
    ListPageShell,
    ListHeader,
    ListToolbar,
    ListSearch,
    ListCard,
    ListTable,
    Th,
    Tr,
    Td,
    StatusPill,
    Chip,
    SlugLink,
    TitleCell,
    RowActions,
    ListFooter,
    EmptyState,
    LoadingRow,
    // Layout helpers
    AccordionRow,
    SectionHeader,
    // shadcn/ui primitives
    Button,
    Card,
    CardContent,
    CardHeader,
    CardTitle,
    CardDescription,
    CardFooter,
    CardAction,
    Input,
    Label,
    Badge,
    Checkbox,
    Separator,
    Dialog,
    DialogClose,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
    DialogTrigger,
    DialogPortal,
    DialogOverlay,
    DropdownMenu,
    DropdownMenuPortal,
    DropdownMenuTrigger,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuLabel,
    DropdownMenuItem,
    DropdownMenuCheckboxItem,
    DropdownMenuRadioGroup,
    DropdownMenuRadioItem,
    DropdownMenuSeparator,
    DropdownMenuShortcut,
    DropdownMenuSub,
    DropdownMenuSubTrigger,
    DropdownMenuSubContent,
    Popover,
    PopoverTrigger,
    PopoverContent,
    PopoverAnchor,
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
    Tabs,
    TabsList,
    TabsTrigger,
    TabsContent,
    Textarea,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
    TableFooter,
    TableCaption,
    Switch,
    BlockPicker,
    // Rich editors
    CodeEditor,
    RichTextEditor,
    CodeViewer,
    CodeWindow,
    // Command (combobox)
    Command,
    CommandDialog,
    CommandInput,
    CommandList,
    CommandEmpty,
    CommandGroup,
    CommandItem,
    CommandShortcut,
    CommandSeparator,
  },
  api: apiClient,
  ExtensionSlot,
  useExtensions,
};

// ---------------------------------------------------------------------------
// SDUI integration components — must live inside BrowserRouter so they can
// call useNavigate() from the router context.
// ---------------------------------------------------------------------------

/** Reads navigate from router context and wires it into the SDUI action handler. */
function NavigateBridge() {
  const navigate = useNavigate();
  React.useEffect(() => {
    setNavigate(navigate);
  }, [navigate]);
  return null;
}

/** Activates the SSE connection for real-time state invalidation. */
function SduiProviders({ children }: { children: React.ReactNode }) {
  useSSE();
  return <>{children}</>;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <NavigateBridge />
        <SduiProviders>
          <App />
          <ConfirmDialogHost />
          <Toaster position="top-right" richColors />
        </SduiProviders>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
