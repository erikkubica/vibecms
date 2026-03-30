// Icons shim for extension micro-frontends.
// Uses lazy accessors to avoid timing issues where the shim evaluates before main.tsx.

function getIcon(name) {
  const icons = window.__VIBECMS_SHARED__?.icons;
  if (icons && icons[name]) return icons[name];
  // Return a tiny placeholder so the component still renders.
  const React = window.__VIBECMS_SHARED__?.React;
  const Fallback = function(props) {
    if (!React) return null;
    return React.createElement("span", {
      className: props?.className || "",
      "aria-hidden": "true",
    });
  };
  Fallback.displayName = "Icon." + name;
  return Fallback;
}

function lazyIcon(name) {
  const Component = function(props) {
    const Real = getIcon(name);
    const React = window.__VIBECMS_SHARED__?.React;
    return React ? React.createElement(Real, props) : null;
  };
  Component.displayName = name;
  return Component;
}

// Default export: Proxy that returns lazy icons for any name
export default new Proxy({}, {
  get(_, prop) {
    return lazyIcon(prop);
  },
});

export const Settings = lazyIcon("Settings");
export const Loader2 = lazyIcon("Loader2");
export const Check = lazyIcon("Check");
export const X = lazyIcon("X");
export const Plus = lazyIcon("Plus");
export const Trash2 = lazyIcon("Trash2");
export const Edit = lazyIcon("Edit");
export const Save = lazyIcon("Save");
export const Mail = lazyIcon("Mail");
export const Send = lazyIcon("Send");
export const Server = lazyIcon("Server");
export const Key = lazyIcon("Key");
export const Globe = lazyIcon("Globe");
export const Shield = lazyIcon("Shield");
export const AlertCircle = lazyIcon("AlertCircle");
export const ChevronRight = lazyIcon("ChevronRight");
export const ChevronDown = lazyIcon("ChevronDown");
export const ExternalLink = lazyIcon("ExternalLink");
export const RefreshCw = lazyIcon("RefreshCw");
export const Puzzle = lazyIcon("Puzzle");
export const FolderOpen = lazyIcon("FolderOpen");
export const FileText = lazyIcon("FileText");
export const Pencil = lazyIcon("Pencil");
export const Eye = lazyIcon("Eye");
export const ArrowLeft = lazyIcon("ArrowLeft");
export const Upload = lazyIcon("Upload");
export const Search = lazyIcon("Search");
export const Image = lazyIcon("Image");
export const Film = lazyIcon("Film");
export const Music = lazyIcon("Music");
export const File = lazyIcon("File");
export const Copy = lazyIcon("Copy");
export const Power = lazyIcon("Power");
export const Grid3x3 = lazyIcon("Grid3x3");
export const List = lazyIcon("List");
export const ArrowUpDown = lazyIcon("ArrowUpDown");
export const LayoutGrid = lazyIcon("LayoutGrid");
export const Table = lazyIcon("Table");
export const Download = lazyIcon("Download");
export const MoreHorizontal = lazyIcon("MoreHorizontal");
export const ChevronLeft = lazyIcon("ChevronLeft");
export const ChevronsLeft = lazyIcon("ChevronsLeft");
export const ChevronsRight = lazyIcon("ChevronsRight");
export const RotateCcw = lazyIcon("RotateCcw");
export const Layout = lazyIcon("Layout");
