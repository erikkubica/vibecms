import { useState, useMemo } from "react";
import {
  Square,
  LayoutTemplate,
  Type,
  Image,
  MousePointerClick,
  Images,
  Play,
  List,
  Quote,
  MapPin,
  Code as CodeIcon,
  SeparatorHorizontal as SeparatorIcon,
  FileText,
  Newspaper,
  ShoppingBag,
  Calendar,
  Users,
  Folder,
  Bookmark,
  Tag,
  Star,
  Heart,
  Search,
  type LucideIcon,
} from "lucide-react";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export const BLOCK_ICON_MAP: Record<string, LucideIcon> = {
  "square": Square,
  "layout-template": LayoutTemplate,
  "type": Type,
  "image": Image,
  "mouse-pointer-click": MousePointerClick,
  "images": Images,
  "play": Play,
  "list": List,
  "quote": Quote,
  "map-pin": MapPin,
  "code": CodeIcon,
  "separator": SeparatorIcon,
  "file-text": FileText,
  "newspaper": Newspaper,
  "shopping-bag": ShoppingBag,
  "calendar": Calendar,
  "users": Users,
  "folder": Folder,
  "bookmark": Bookmark,
  "tag": Tag,
  "star": Star,
  "heart": Heart,
};

export interface PickerItem {
  id: string | number;
  slug: string;
  label: string;
  description?: string;
  icon?: string;
  preview_image?: string;
  badge?: string;
}

interface BlockPickerProps {
  open: boolean;
  onClose: () => void;
  onSelect: (item: PickerItem) => void;
  items: PickerItem[];
  title: string;
  description: string;
  emptyMessage: string;
}

export default function BlockPicker({
  open,
  onClose,
  onSelect,
  items,
  title,
  description,
  emptyMessage,
}: BlockPickerProps) {
  const [search, setSearch] = useState("");

  const filtered = useMemo(() => {
    if (!search.trim()) return items;
    const q = search.toLowerCase();
    return items.filter(
      (item) =>
        item.label.toLowerCase().includes(q) ||
        (item.description?.toLowerCase().includes(q))
    );
  }, [items, search]);

  function handleSelect(item: PickerItem) {
    onSelect(item);
    setSearch("");
  }

  function handleOpenChange(v: boolean) {
    if (!v) {
      setSearch("");
      onClose();
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-5xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
          <Input
            placeholder="Search..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 rounded-lg border-slate-300"
          />
        </div>

        {/* Grid */}
        <div className="flex-1 overflow-y-auto min-h-0 -mx-1 px-1 py-2">
          {filtered.length > 0 ? (
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
              {filtered.map((item) => {
                const IconComp = (item.icon && BLOCK_ICON_MAP[item.icon]) || Square;
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => handleSelect(item)}
                    className="group rounded-xl border border-slate-200 bg-white text-left transition-all hover:border-indigo-300 hover:shadow-md overflow-hidden"
                  >
                    {/* Preview area — 4:3 aspect ratio */}
                    <div className="aspect-[4/3] bg-slate-50 flex items-center justify-center border-b border-slate-100 group-hover:bg-indigo-50/50 transition-colors overflow-hidden">
                      {item.preview_image ? (
                        <img
                          src={item.preview_image}
                          alt={item.label}
                          className="h-full w-full object-cover"
                        />
                      ) : (
                        <IconComp className="h-10 w-10 text-slate-300 group-hover:text-indigo-400 transition-colors" />
                      )}
                    </div>
                    {/* Info */}
                    <div className="p-3 space-y-0.5">
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium text-slate-800 truncate flex-1">
                          {item.label}
                        </p>
                        {item.badge && (
                          <span className="text-[10px] font-medium text-slate-400 bg-slate-100 rounded-full px-2 py-0.5 shrink-0">
                            {item.badge}
                          </span>
                        )}
                      </div>
                      {item.description && (
                        <p className="text-xs text-slate-400 line-clamp-2">
                          {item.description}
                        </p>
                      )}
                    </div>
                  </button>
                );
              })}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-16 text-slate-400">
              {search.trim() ? (
                <>
                  <Search className="h-10 w-10 mb-3 text-slate-300" />
                  <p className="text-sm">No results for &ldquo;{search}&rdquo;</p>
                </>
              ) : (
                <p className="text-sm">{emptyMessage}</p>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
