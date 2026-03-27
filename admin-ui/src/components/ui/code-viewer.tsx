/**
 * Shared read-only code viewer with syntax highlighting.
 * Uses CodeMirror with oneDark theme — same engine as the code editor.
 * Auto-detects language from file extension or explicit prop.
 */
import { useMemo } from "react";
import CodeMirror, { type Extension } from "@uiw/react-codemirror";
import { oneDark } from "@codemirror/theme-one-dark";
import { html } from "@codemirror/lang-html";
import { javascript } from "@codemirror/lang-javascript";
import { css } from "@codemirror/lang-css";
import { json } from "@codemirror/lang-json";
import { sql } from "@codemirror/lang-sql";
import { markdown } from "@codemirror/lang-markdown";
import { go } from "@codemirror/lang-go";
import { yaml } from "@codemirror/lang-yaml";
import { EditorView } from "@codemirror/view";

const readOnlyTheme = EditorView.theme({
  "&": { backgroundColor: "transparent !important" },
  ".cm-gutters": { backgroundColor: "transparent", borderRight: "1px solid #333" },
  ".cm-activeLineGutter": { backgroundColor: "transparent" },
});

function getLanguageExtension(lang: string): Extension | null {
  switch (lang.toLowerCase()) {
    case "html":
    case "htm":
    case "xml":
    case "svg":
      return html();
    case "javascript":
    case "js":
    case "jsx":
      return javascript({ jsx: true });
    case "typescript":
    case "ts":
    case "tsx":
      return javascript({ jsx: true, typescript: true });
    case "css":
    case "scss":
      return css();
    case "json":
    case "jsonc":
      return json();
    case "sql":
      return sql();
    case "markdown":
    case "md":
      return markdown();
    case "go":
      return go();
    case "yaml":
    case "yml":
      return yaml();
    // Tengo uses JS-like syntax (closest match)
    case "tengo":
    case "tgo":
      return javascript();
    case "shell":
    case "sh":
    case "bash":
      return null; // No built-in shell, fallback to no highlighting
    default:
      return null;
  }
}

/** Detect language from filename extension. */
export function detectLanguage(filename: string): string {
  const ext = filename.split(".").pop()?.toLowerCase() ?? "";
  const map: Record<string, string> = {
    go: "go", ts: "typescript", tsx: "tsx", js: "javascript", jsx: "jsx",
    json: "json", html: "html", htm: "html", css: "css", scss: "scss",
    md: "markdown", txt: "text", tgo: "tengo", tengo: "tengo",
    yaml: "yaml", yml: "yaml", toml: "toml", py: "python",
    sh: "shell", bash: "shell", sql: "sql", xml: "xml", svg: "svg",
  };
  return map[ext] ?? "text";
}

interface CodeViewerProps {
  /** The code to display. */
  value: string;
  /** Language for syntax highlighting. Auto-detected from filename if not provided. */
  language?: string;
  /** Filename — used for auto-detecting language if `language` is not set. */
  filename?: string;
  /** Height of the viewer. Defaults to "100%" (fill parent). */
  height?: string;
}

export default function CodeViewer({
  value,
  language,
  filename,
  height = "100%",
}: CodeViewerProps) {
  const lang = language ?? (filename ? detectLanguage(filename) : "text");

  const extensions = useMemo(() => {
    const exts: Extension[] = [readOnlyTheme];
    const langExt = getLanguageExtension(lang);
    if (langExt) exts.push(langExt);
    return exts;
  }, [lang]);

  return (
    <CodeMirror
      value={value}
      height={height}
      theme={oneDark}
      extensions={extensions}
      readOnly
      editable={false}
      basicSetup={{
        lineNumbers: true,
        foldGutter: true,
        bracketMatching: true,
        highlightActiveLine: false,
        highlightActiveLineGutter: false,
      }}
    />
  );
}
