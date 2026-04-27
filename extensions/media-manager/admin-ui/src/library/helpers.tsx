import { useState } from "react";
import { AlertCircle, Image as ImageIcon, FileText, Film, Music, File } from "@vibecms/icons";

export interface MediaFile {
  id: number;
  filename: string;
  original_name: string;
  mime_type: string;
  size: number;
  path: string;
  url: string;
  width: number | null;
  height: number | null;
  alt: string;
  created_at: string;
  updated_at: string;
  is_optimized: boolean;
  original_size: number;
  original_path: string;
  original_width: number | null;
  original_height: number | null;
  optimization_savings: number;
}

export interface PaginationMeta {
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export function humanFileSize(bytes: number): string {
  if (!bytes || bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function isImage(mime: string): boolean {
  return mime.startsWith("image/");
}
export function isVideo(mime: string): boolean {
  return mime.startsWith("video/");
}
export function isAudio(mime: string): boolean {
  return mime.startsWith("audio/");
}

export function getFileExtension(name: string): string {
  const parts = name.split(".");
  return parts.length > 1 ? parts[parts.length - 1].toUpperCase() : "";
}

export function mimeLabel(mime: string): string {
  const sub = mime.split("/")[1];
  if (!sub) return mime;
  return sub.replace(/^x-/, "").toUpperCase();
}

export function imageSize(url: string, size: string, updatedAt?: string): string {
  if (!url.startsWith("/media/")) return url;
  let result = "/media/cache/" + size + "/" + url.slice(7);
  if (updatedAt) result += "?v=" + new Date(updatedAt).getTime();
  return result;
}

export function savedPercent(file: MediaFile): number {
  if (!file.original_size || file.original_size <= 0) return 0;
  return Math.round((1 - file.size / file.original_size) * 100);
}

export function copyToClipboard(text: string): Promise<void> {
  if (navigator.clipboard && window.isSecureContext) return navigator.clipboard.writeText(text);
  const ta = document.createElement("textarea");
  ta.value = text;
  ta.style.position = "fixed";
  ta.style.opacity = "0";
  document.body.appendChild(ta);
  ta.select();
  document.execCommand("copy");
  document.body.removeChild(ta);
  return Promise.resolve();
}

export function FileTypeIcon({ mime, className }: { mime: string; className?: string }) {
  if (isImage(mime)) return <ImageIcon className={className} />;
  if (isVideo(mime)) return <Film className={className} />;
  if (isAudio(mime)) return <Music className={className} />;
  if (mime.includes("pdf") || mime.includes("document") || mime.includes("text"))
    return <FileText className={className} />;
  return <File className={className} />;
}

export function BrokenMediaFallback({ className }: { className?: string }) {
  return (
    <div className={`flex flex-col items-center justify-center gap-2 w-full h-full bg-slate-50 text-slate-400 ${className || ""}`}>
      <AlertCircle className="h-8 w-8 text-slate-300" />
      <span className="text-xs font-medium text-slate-400">Failed to load</span>
    </div>
  );
}

export function MediaImage({
  src,
  alt,
  className,
  style,
  onError,
}: {
  src: string;
  alt: string;
  className?: string;
  style?: React.CSSProperties;
  onError?: () => void;
}) {
  const [broken, setBroken] = useState(false);
  if (broken) return <BrokenMediaFallback className={className} />;
  return (
    <img
      src={src}
      alt={alt}
      className={className}
      style={style}
      loading="lazy"
      onError={() => {
        setBroken(true);
        onError?.();
      }}
    />
  );
}

export function MediaVideo({ src, className, controls }: { src: string; className?: string; controls?: boolean }) {
  const [broken, setBroken] = useState(false);
  if (broken) return <BrokenMediaFallback />;
  return <video src={src} className={className} controls={controls} onError={() => setBroken(true)} />;
}

export function fmtDate(iso: string): string {
  return new Date(iso).toLocaleDateString();
}
