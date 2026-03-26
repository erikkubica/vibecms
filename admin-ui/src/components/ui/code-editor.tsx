import { useMemo, useState, useCallback } from "react";
import CodeMirror, { type Extension } from "@uiw/react-codemirror";
import { html } from "@codemirror/lang-html";
import { oneDark } from "@codemirror/theme-one-dark";
import { autocompletion, type CompletionContext } from "@codemirror/autocomplete";
import { keymap } from "@codemirror/view";
import { format } from "prettier/standalone";
import * as htmlParser from "prettier/plugins/html";

interface CodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
  height?: string;
  variables?: string[];
}

export default function CodeEditor({
  value,
  onChange,
  disabled = false,
  placeholder,
  height = "300px",
  variables = [],
}: CodeEditorProps) {
  const [formatting, setFormatting] = useState(false);

  const handleFormat = useCallback(async () => {
    if (disabled || !value.trim()) return;
    setFormatting(true);
    try {
      // Protect Go template tags from prettier mangling
      let code = value;
      const tags: string[] = [];
      code = code.replace(/\{\{[^}]*\}\}/g, (match) => {
        tags.push(match);
        return `<!--TMPL${tags.length - 1}-->`;
      });

      const formatted = await format(code, {
        parser: "html",
        plugins: [htmlParser],
        printWidth: 120,
        tabWidth: 2,
        useTabs: false,
      });

      // Restore Go template tags
      const restored = formatted.replace(/<!--TMPL(\d+)-->/g, (_, i) => tags[Number(i)]);
      onChange(restored.trimEnd());
    } catch {
      // If formatting fails (invalid HTML), silently ignore
    } finally {
      setFormatting(false);
    }
  }, [value, onChange, disabled]);

  const extensions = useMemo(() => {
    const exts: Extension[] = [
      html(),
      keymap.of([{
        key: "Shift-Alt-f",
        run: () => { handleFormat(); return true; },
      }]),
    ];

    if (variables.length > 0) {
      const varCompletions = variables.map((v) => ({
        label: `{{.${v}}}`,
        type: "variable" as const,
        info: `Insert template variable ${v}`,
        apply: `{{.${v}}}`,
      }));

      // Also add common Go template constructs
      const templateCompletions = [
        { label: "{{if .var}}", type: "keyword" as const, info: "Conditional block", apply: "{{if .}}" },
        { label: "{{else}}", type: "keyword" as const, info: "Else branch", apply: "{{else}}" },
        { label: "{{end}}", type: "keyword" as const, info: "End block", apply: "{{end}}" },
        { label: "{{range .var}}", type: "keyword" as const, info: "Range loop", apply: "{{range .}}" },
        { label: "{{if .var}}...{{end}}", type: "keyword" as const, info: "Conditional with end", apply: "{{if .}}\n  \n{{end}}" },
      ];

      const allCompletions = [...varCompletions, ...templateCompletions];

      exts.push(
        autocompletion({
          override: [
            (context: CompletionContext) => {
              const word = context.matchBefore(/\{\{\.?\w*\}?\}?|\w+/);
              if (!word) return null;
              return {
                from: word.from,
                options: allCompletions,
                validFor: /^.*$/,
              };
            },
          ],
        })
      );
    }

    return exts;
  }, [variables]);

  return (
    <div className="overflow-hidden rounded-lg border border-slate-300 focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/20">
      <div className="flex items-center justify-end gap-2 border-b border-slate-700 bg-[#282c34] px-3 py-1.5">
        <button
          type="button"
          onClick={handleFormat}
          disabled={disabled || formatting}
          className="flex items-center gap-1.5 rounded px-2.5 py-1 text-xs font-medium text-slate-400 transition-colors hover:bg-slate-700 hover:text-slate-200 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {formatting ? "Formatting..." : "Format"}
          <kbd className="hidden sm:inline rounded bg-slate-700 px-1.5 py-0.5 text-[10px] text-slate-500">Shift+Alt+F</kbd>
        </button>
      </div>
      <CodeMirror
        value={value}
        onChange={onChange}
        height={height}
        theme={oneDark}
        extensions={extensions}
        readOnly={disabled}
        placeholder={placeholder}
        basicSetup={{
          lineNumbers: true,
          foldGutter: true,
          bracketMatching: true,
          closeBrackets: true,
          autocompletion: true,
          highlightActiveLine: true,
          indentOnInput: true,
        }}
      />
    </div>
  );
}
