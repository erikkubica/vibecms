import { useState, useMemo } from "react";
import { Check, ChevronsUpDown, Type, AlignLeft, Hash, Calendar, ListOrdered, Image, ToggleLeft, Link2, Layers, Repeat, FileSearch, Palette, Mail, Globe, FileText as RichTextIcon, SlidersHorizontal, File, Images, CircleDot, CheckSquare } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useExtensions } from "@/hooks/use-extensions";

interface FieldTypeOption {
  value: string;
  label: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
  group: string;
}

const FIELD_TYPE_OPTIONS: FieldTypeOption[] = [
  // Basic
  { value: "text", label: "Text", description: "Single-line text input", icon: Type, group: "Basic" },
  { value: "textarea", label: "Textarea", description: "Multi-line text input", icon: AlignLeft, group: "Basic" },
  { value: "richtext", label: "Rich Text", description: "WYSIWYG rich text editor", icon: RichTextIcon, group: "Basic" },
  { value: "number", label: "Number", description: "Numeric input with constraints", icon: Hash, group: "Basic" },
  { value: "range", label: "Range Slider", description: "Slider with min/max values", icon: SlidersHorizontal, group: "Basic" },
  { value: "email", label: "Email", description: "Email address input", icon: Mail, group: "Basic" },
  { value: "url", label: "URL", description: "Web address input", icon: Globe, group: "Basic" },
  { value: "date", label: "Date", description: "Date picker", icon: Calendar, group: "Basic" },
  { value: "color", label: "Color Picker", description: "Color selection with hex value", icon: Palette, group: "Basic" },
  // Choice
  { value: "toggle", label: "Toggle", description: "On/off boolean switch", icon: ToggleLeft, group: "Choice" },
  { value: "select", label: "Select", description: "Dropdown with predefined options", icon: ListOrdered, group: "Choice" },
  { value: "radio", label: "Radio Buttons", description: "Single choice from options", icon: CircleDot, group: "Choice" },
  { value: "checkbox", label: "Checkbox Group", description: "Multiple choice from options", icon: CheckSquare, group: "Choice" },
  // Media
  { value: "image", label: "Image", description: "Single image upload", icon: Image, group: "Media" },
  { value: "gallery", label: "Gallery", description: "Multiple image uploads", icon: Images, group: "Media" },
  { value: "file", label: "File", description: "File upload with type filtering", icon: File, group: "Media" },
  // Relational
  { value: "link", label: "Link", description: "URL with text, alt, and target", icon: Link2, group: "Relational" },
  { value: "node", label: "Node Selector", description: "Reference to content nodes", icon: FileSearch, group: "Relational" },
  // Layout
  { value: "group", label: "Group", description: "Container for nested fields", icon: Layers, group: "Layout" },
  { value: "repeater", label: "Repeater", description: "Repeatable set of fields", icon: Repeat, group: "Layout" },
];

interface FieldTypePickerProps {
  value: string;
  onValueChange: (value: string) => void;
  className?: string;
  compact?: boolean;
}

export function getFieldTypeOption(value: string) {
  return FIELD_TYPE_OPTIONS.find((o) => o.value === value);
}

export function getFieldTypeGroups() {
  return [...new Set(FIELD_TYPE_OPTIONS.map((o) => o.group))];
}

export { FIELD_TYPE_OPTIONS };

export default function FieldTypePicker({ value, onValueChange, className, compact }: FieldTypePickerProps) {
  const [open, setOpen] = useState(false);
  const { getFieldTypes } = useExtensions();
  const extFieldTypes = getFieldTypes();

  // Build merged options: core types (minus those "supported" by extensions) + extension types
  const mergedOptions = useMemo(() => {
    const supportedSet = new Set<string>();
    for (const eft of extFieldTypes) {
      if (eft.supports) eft.supports.forEach((s) => supportedSet.add(s));
    }

    const coreFiltered = FIELD_TYPE_OPTIONS.filter((o) => !supportedSet.has(o.value));
    const extOptions: FieldTypeOption[] = extFieldTypes.map((eft) => ({
      value: eft.type,
      label: eft.label,
      description: eft.description,
      icon: eft.icon,
      group: eft.group,
    }));

    return [...coreFiltered, ...extOptions];
  }, [extFieldTypes]);

  const selected = mergedOptions.find((o) => o.value === value);
  const groups = [...new Set(mergedOptions.map((o) => o.group))];

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn(
            "w-full justify-between font-normal",
            compact ? "h-9 text-sm" : "h-10",
            "rounded-lg border-slate-300 hover:bg-slate-50",
            className
          )}
        >
          {selected ? (
            <span className="flex items-center gap-2 truncate">
              <selected.icon className={cn("shrink-0 text-slate-500", compact ? "h-3.5 w-3.5" : "h-4 w-4")} />
              <span className="truncate">{selected.label}</span>
            </span>
          ) : (
            <span className="text-slate-400">Select field type...</span>
          )}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[320px] p-0" align="start">
        <Command>
          <CommandInput placeholder="Search field types..." />
          <CommandList>
            <CommandEmpty>No field type found.</CommandEmpty>
            {groups.map((group) => (
              <CommandGroup key={group} heading={group}>
                {mergedOptions.filter((o) => o.group === group).map((option) => {
                  const Icon = option.icon;
                  return (
                    <CommandItem
                      key={option.value}
                      value={option.value}
                      keywords={[option.label, option.description, option.group]}
                      onSelect={() => {
                        onValueChange(option.value);
                        setOpen(false);
                      }}
                    >
                      <div className={cn(
                        "flex h-8 w-8 shrink-0 items-center justify-center rounded-md border",
                        value === option.value
                          ? "border-indigo-200 bg-indigo-50 text-indigo-600"
                          : "border-slate-200 bg-slate-50 text-slate-500"
                      )}>
                        <Icon className="h-4 w-4" />
                      </div>
                      <div className="flex flex-col gap-0.5 min-w-0">
                        <span className="text-sm font-medium text-slate-800">{option.label}</span>
                        <span className="text-xs text-slate-400 truncate">{option.description}</span>
                      </div>
                      {value === option.value && (
                        <Check className="ml-auto h-4 w-4 shrink-0 text-indigo-600" />
                      )}
                    </CommandItem>
                  );
                })}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
