/**
 * Tailwind CSS safelist for extension micro-frontends.
 *
 * Extensions render inside the admin shell and share its stylesheet.
 * Tailwind only generates classes it finds in scanned source files,
 * but extension source lives outside admin-ui and isn't scanned.
 *
 * This file ensures commonly needed utilities are always generated.
 * Extensions can use any class listed here without worrying about
 * whether the host app happens to use it too.
 *
 * To add classes: just include the class name anywhere in this file.
 * Tailwind's JIT scanner picks up class-like tokens automatically.
 */

export const EXTENSION_SAFELIST = [
  // Grid layouts
  "grid",
  "grid-cols-1", "grid-cols-2", "grid-cols-3", "grid-cols-4", "grid-cols-5", "grid-cols-6",
  "sm:grid-cols-2", "sm:grid-cols-3", "sm:grid-cols-4", "sm:grid-cols-5", "sm:grid-cols-6",
  "md:grid-cols-2", "md:grid-cols-3", "md:grid-cols-4", "md:grid-cols-5", "md:grid-cols-6",
  "lg:grid-cols-2", "lg:grid-cols-3", "lg:grid-cols-4", "lg:grid-cols-5", "lg:grid-cols-6",
  "gap-1", "gap-2", "gap-3", "gap-4", "gap-6",
  "col-span-1", "col-span-2", "col-span-3", "col-span-full",

  // Aspect ratios
  "aspect-square", "aspect-video", "aspect-auto",

  // Cursor
  "cursor-grab", "cursor-grabbing", "cursor-pointer", "cursor-default", "cursor-move",
  "active:cursor-grabbing",

  // Object fit
  "object-cover", "object-contain", "object-center",

  // Positioning
  "absolute", "relative", "fixed", "sticky",
  "inset-0", "inset-x-0", "inset-y-0",
  "top-0", "top-1", "top-2", "right-0", "right-1", "right-2",
  "bottom-0", "bottom-1", "bottom-2", "left-0", "left-1", "left-2",
  "top-full", "bottom-full",
  "z-10", "z-20", "z-30", "z-40", "z-50",

  // Overflow
  "overflow-hidden", "overflow-visible", "overflow-auto", "overflow-y-auto", "overflow-x-auto",

  // Visibility / opacity
  "opacity-0", "opacity-50", "opacity-100",
  "group-hover:opacity-100",
  "invisible", "visible",

  // Transitions
  "transition-all", "transition-colors", "transition-opacity", "transition-transform",

  // Transforms
  "scale-95", "scale-100", "scale-105",
  "scale-[1.02]",
  "rotate-90", "rotate-180",

  // Sizing
  "h-full", "w-full", "h-screen", "w-screen",
  "min-h-0", "min-w-0", "max-h-64", "max-h-96",
  "size-3", "size-4", "size-5", "size-6", "size-8", "size-10",

  // Spacing
  "space-y-1", "space-y-2", "space-y-3", "space-y-4",
  "space-x-1", "space-x-2", "space-x-3", "space-x-4",

  // Flex
  "flex", "flex-1", "flex-col", "flex-row", "flex-wrap",
  "items-center", "items-start", "items-end",
  "justify-center", "justify-between", "justify-start", "justify-end",
  "shrink-0", "grow",

  // Borders
  "border", "border-2", "border-dashed",
  "border-slate-200", "border-slate-300", "border-indigo-200", "border-indigo-400",
  "rounded", "rounded-md", "rounded-lg", "rounded-xl", "rounded-full",

  // Backgrounds
  "bg-white", "bg-black/40", "bg-black/50", "bg-black/60",
  "bg-slate-50", "bg-slate-100",
  "bg-indigo-50", "bg-indigo-600",
  "bg-red-50", "bg-red-600",
  "bg-emerald-50", "bg-emerald-100",
  "hover:bg-white/20", "hover:bg-indigo-600", "hover:bg-red-600", "hover:bg-red-500/80",

  // Text
  "text-white", "text-slate-400", "text-slate-500", "text-slate-600", "text-slate-700",
  "text-indigo-600", "text-indigo-700", "text-red-500",
  "text-xs", "text-sm", "text-[9px]", "text-[10px]", "text-[11px]",
  "font-medium", "font-semibold", "font-mono",
  "truncate", "line-clamp-2",

  // Padding / margin
  "p-1", "p-2", "p-3", "p-4", "p-5", "p-6",
  "px-1", "px-1.5", "px-2", "px-2.5", "px-3", "px-4",
  "py-0.5", "py-1", "py-2", "py-3", "py-4",
  "mt-1", "mb-1", "ml-auto", "mr-2",

  // Drag and drop
  "draggable",

  // Group
  "group",
] as const;
