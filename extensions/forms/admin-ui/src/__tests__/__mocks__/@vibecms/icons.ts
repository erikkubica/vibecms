import React from "react";

// Proxy that creates a no-op span component for every named icon export
const iconsProxy = new Proxy(
  {} as Record<string, React.FC<{ className?: string }>>,
  {
    get(_target, prop: string) {
      const Icon = ({ className }: { className?: string }) =>
        React.createElement("span", { "data-testid": `icon-${prop}`, className });
      Icon.displayName = prop;
      return Icon;
    },
  },
);

export default iconsProxy;

// Re-export all as named exports via the proxy
export const Trash2 = iconsProxy.Trash2;
export const Archive = iconsProxy.Archive;
export const CheckCircle = iconsProxy.CheckCircle;
export const Circle = iconsProxy.Circle;
export const Filter = iconsProxy.Filter;
export const Mail = iconsProxy.Mail;
export const ChevronDown = iconsProxy.ChevronDown;
export const ChevronUp = iconsProxy.ChevronUp;
export const User = iconsProxy.User;
export const Users = iconsProxy.Users;
export const Send = iconsProxy.Send;
export const Plus = iconsProxy.Plus;
export const PlusSquare = iconsProxy.PlusSquare;
export const X = iconsProxy.X;
export const ExternalLink = iconsProxy.ExternalLink;
export const Copy = iconsProxy.Copy;
export const Download = iconsProxy.Download;
export const Upload = iconsProxy.Upload;
export const Settings = iconsProxy.Settings;
export const Eye = iconsProxy.Eye;
export const EyeOff = iconsProxy.EyeOff;
export const Search = iconsProxy.Search;
export const AlertCircle = iconsProxy.AlertCircle;
export const Info = iconsProxy.Info;
export const Check = iconsProxy.Check;
export const Loader2 = iconsProxy.Loader2;
export const MoreHorizontal = iconsProxy.MoreHorizontal;
export const GripVertical = iconsProxy.GripVertical;
